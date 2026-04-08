[CmdletBinding()]
param(
    [switch]$VerifyBundleLayout,
    [switch]$VerifyDocsOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$ZipPath = Join-Path $RepoRoot 'dist/windows-zip/ACGWarehouse-windows-x64-portable.zip'
$DocsPath = Join-Path $RepoRoot 'docs/windows-portable-package.md'

function New-SmokeFailure {
    param([Parameter(Mandatory = $true)][string]$Message)

    return "[package-smoke] $Message"
}

function Assert-PathExists {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Message
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        throw $Message
    }
}

function Assert-DocContains {
    param(
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][string]$Message
    )

    if (-not (Select-String -Path $DocsPath -Pattern $Pattern -SimpleMatch -Quiet)) {
        throw $Message
    }
}

function Verify-BundleLayout {
    param(
        [Parameter(Mandatory = $true)][string]$BundleRoot
    )

    $requiredPaths = @(
        'ACGWarehouse.exe',
        'data',
        'runtime/bin',
        'config',
        'storage',
        'library'
    )

    foreach ($relativePath in $requiredPaths) {
        Assert-PathExists -Path (Join-Path $BundleRoot $relativePath) -Message (New-SmokeFailure "Portable bundle is missing required path: $relativePath")
    }
}

function New-TemporaryDirectory {
    $path = Join-Path ([System.IO.Path]::GetTempPath()) ("acgwarehouse-package-smoke-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $path -Force | Out-Null
    return $path
}

function Get-FreeTcpPort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    try {
        $listener.Start()
        return $listener.LocalEndpoint.Port
    }
    finally {
        $listener.Stop()
    }
}

function Start-PackagedApplication {
    param(
        [Parameter(Mandatory = $true)][string]$ExecutablePath,
        [Parameter(Mandatory = $true)][string]$WorkingDirectory,
        [Parameter()][hashtable]$EnvironmentOverrides = @{}
    )

    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $ExecutablePath
    $startInfo.WorkingDirectory = $WorkingDirectory
    $startInfo.UseShellExecute = $false

    foreach ($entry in Get-ChildItem Env:) {
        $startInfo.EnvironmentVariables[$entry.Name] = $entry.Value
    }

    foreach ($key in $EnvironmentOverrides.Keys) {
        $startInfo.EnvironmentVariables[[string]$key] = [string]$EnvironmentOverrides[$key]
    }

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo

    if (-not $process.Start()) {
        throw (New-SmokeFailure "Failed to launch packaged application: $ExecutablePath")
    }

    return $process
}

function Wait-ForPath {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][int]$TimeoutSeconds,
        [Parameter(Mandatory = $true)][string]$FailureMessage
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-Path -LiteralPath $Path) {
            return
        }

        Start-Sleep -Milliseconds 250
    }

    throw (New-SmokeFailure $FailureMessage)
}

function Get-ManifestBaseUrl {
    param([Parameter(Mandatory = $true)][string]$ManifestPath)

    $manifest = Get-Content -LiteralPath $ManifestPath -Raw | ConvertFrom-Json
    $baseUrl = [string]$manifest.go.base_url
    if ([string]::IsNullOrWhiteSpace($baseUrl)) {
        throw (New-SmokeFailure "runtime/runtime-manifest.json does not contain go.base_url")
    }

    return $baseUrl.TrimEnd('/')
}

function Wait-ForHttpOk {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][int]$TimeoutSeconds
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastError = $null
    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
                return
            }
            $lastError = "Unexpected status code $($response.StatusCode) from $Url"
        }
        catch {
            $lastError = $_.Exception.Message
        }

        Start-Sleep -Milliseconds 500
    }

    throw (New-SmokeFailure "Health probe failed for $Url. Last error: $lastError")
}

function Stop-ProcessTree {
    param([Parameter(Mandatory = $true)][System.Diagnostics.Process]$Process)

    try {
        if ($Process.HasExited) {
            return
        }
    }
    catch {
        return
    }

    try {
        $null = $Process.CloseMainWindow()
        Start-Sleep -Seconds 2
    }
    catch {}

    try {
        if (-not $Process.HasExited) {
            & taskkill /PID $Process.Id /T /F | Out-Null
        }
    }
    catch {}
}

function Invoke-PackagedSmoke {
    Assert-PathExists -Path $ZipPath -Message (New-SmokeFailure 'Portable ZIP artifact is missing. Expected dist/windows-zip/ACGWarehouse-windows-x64-portable.zip.')

    $extractRoot = New-TemporaryDirectory
    $process = $null

    try {
        Expand-Archive -LiteralPath $ZipPath -DestinationPath $extractRoot -Force
        Verify-BundleLayout -BundleRoot $extractRoot

        $appPath = Join-Path $extractRoot 'ACGWarehouse.exe'
        $manifestPath = Join-Path $extractRoot 'runtime/runtime-manifest.json'
        $logsDir = Join-Path $extractRoot 'runtime/logs'
        $diagnosticsDir = Join-Path $extractRoot 'runtime/diagnostics'
        $serverPort = Get-FreeTcpPort

        $process = Start-PackagedApplication -ExecutablePath $appPath -WorkingDirectory $extractRoot -EnvironmentOverrides @{
            'SERVER_HOST' = '127.0.0.1'
            'SERVER_PORT' = "$serverPort"
        }

        Wait-ForPath -Path $manifestPath -TimeoutSeconds 30 -FailureMessage 'Timed out waiting for runtime/runtime-manifest.json after launching ACGWarehouse.exe.'

        $baseUrl = Get-ManifestBaseUrl -ManifestPath $manifestPath
        Wait-ForHttpOk -Url "$baseUrl/health" -TimeoutSeconds 20

        Assert-PathExists -Path $logsDir -Message (New-SmokeFailure 'Packaged runtime/logs directory was not created.')
        Assert-PathExists -Path $diagnosticsDir -Message (New-SmokeFailure 'Packaged runtime/diagnostics directory was not created.')
    }
    finally {
        if ($null -ne $process) {
            Stop-ProcessTree -Process $process
        }

        if (Test-Path -LiteralPath $extractRoot) {
            Remove-Item -LiteralPath $extractRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Verify-Docs {
    Assert-PathExists -Path $DocsPath -Message 'Packaging guide is missing. Expected docs/windows-portable-package.md.'

    $requiredHeadings = @(
        '## Bundle Layout',
        '## Build Command',
        '## First Launch',
        '## In-Place Overwrite Upgrade',
        '## Troubleshooting',
        '## Log and Diagnostic Locations'
    )

    foreach ($heading in $requiredHeadings) {
        Assert-DocContains -Pattern $heading -Message "Packaging guide is missing required heading: $heading"
    }

    $requiredStrings = @(
        'package.ps1',
        'close the running app before overwrite',
        'config/',
        'storage/',
        'library/',
        'replace the Flutter executable + data/ assets + runtime/ binaries as a unit',
        'file locks',
        'delete only old runtime binaries after the app is closed',
        'stale runtime files',
        'runtime compatibility',
        'user-data preservation',
        'go',
        'startup_chain',
        'runtime/diagnostics/startup-error.json',
        'runtime/logs/go.log',
        'runtime/logs/flutter-bootstrap.log'
    )

    foreach ($requiredString in $requiredStrings) {
        Assert-DocContains -Pattern $requiredString -Message "Packaging guide is missing required text: $requiredString"
    }
}

if ($VerifyDocsOnly) {
    Verify-Docs
    Write-Host 'Windows packaging smoke checks passed.'
    exit 0
}

if ($VerifyBundleLayout) {
    Assert-PathExists -Path $ZipPath -Message (New-SmokeFailure 'Portable ZIP artifact is missing. Expected dist/windows-zip/ACGWarehouse-windows-x64-portable.zip.')
    $extractRoot = New-TemporaryDirectory
    try {
        Expand-Archive -LiteralPath $ZipPath -DestinationPath $extractRoot -Force
        Verify-BundleLayout -BundleRoot $extractRoot
    }
    finally {
        if (Test-Path -LiteralPath $extractRoot) {
            Remove-Item -LiteralPath $extractRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    Write-Host 'Windows packaging smoke checks passed.'
    exit 0
}

Invoke-PackagedSmoke

Write-Host 'Windows packaging smoke checks passed.'
