[CmdletBinding()]
param(
    [switch]$SkipTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$PortableRoot = Join-Path $RepoRoot 'dist/windows-portable'
$ZipOutputDir = Join-Path $RepoRoot 'dist/windows-zip'
$ZipPath = Join-Path $ZipOutputDir 'ACGWarehouse-windows-x64-portable.zip'
$PyInstallerWorkDir = Join-Path $RepoRoot 'dist/.pyinstaller/work'
$PyInstallerDistRoot = Join-Path $PortableRoot 'runtime'
$FlutterProjectDir = Join-Path $RepoRoot 'flutter_app'
$FlutterReleaseDir = Join-Path $FlutterProjectDir 'build/windows/x64/runner/Release'
$FlutterOutputExecutable = Join-Path $FlutterReleaseDir 'gallery.exe'
$FlutterBuildCommand = 'flutter build windows --release'
$GoOutputExecutable = Join-Path $PortableRoot 'runtime/bin/acgwarehouse-server.exe'
$SidecarSpecPath = Join-Path $RepoRoot 'services/python-sidecar/sidecar.spec'
$ConfigSourceDir = Join-Path $RepoRoot 'deploy/config'
$PackagedConfigPath = Join-Path $PortableRoot 'config.yaml'

function Reset-Directory {
    param([Parameter(Mandatory = $true)][string]$Path)

    if (Test-Path -LiteralPath $Path) {
        Remove-Item -LiteralPath $Path -Recurse -Force
    }
    New-Item -ItemType Directory -Path $Path -Force | Out-Null
}

function Ensure-Directory {
    param([Parameter(Mandatory = $true)][string]$Path)

    New-Item -ItemType Directory -Path $Path -Force | Out-Null
}

function Assert-Command {
    param([Parameter(Mandatory = $true)][string]$Name)

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command not found: $Name"
    }
}

function Ensure-PythonModule {
    param([Parameter(Mandatory = $true)][string]$ModuleName)

    & python -c "import $ModuleName"
    if ($LASTEXITCODE -eq 0) {
        return
    }

    Write-Host "Installing missing Python module: $ModuleName"
    & python -m pip install --disable-pip-version-check $ModuleName
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to install required Python module: $ModuleName"
    }
}

function Invoke-External {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter()][string[]]$Arguments = @(),
        [Parameter()][string]$WorkingDirectory = $RepoRoot
    )

    Push-Location $WorkingDirectory
    try {
        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
        }
    }
    finally {
        Pop-Location
    }
}

function Copy-DirectoryContents {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination
    )

    Ensure-Directory -Path $Destination
    Copy-Item -Path (Join-Path $Source '*') -Destination $Destination -Recurse -Force
}

function New-ZipFromDirectory {
    param(
        [Parameter(Mandatory = $true)][string]$SourceDirectory,
        [Parameter(Mandatory = $true)][string]$DestinationZip
    )

    Add-Type -AssemblyName System.IO.Compression
    Add-Type -AssemblyName System.IO.Compression.FileSystem

    if (Test-Path -LiteralPath $DestinationZip) {
        Remove-Item -LiteralPath $DestinationZip -Force
    }

    $archive = [System.IO.Compression.ZipFile]::Open($DestinationZip, [System.IO.Compression.ZipArchiveMode]::Create)
    try {
        $rootInfo = Get-Item -LiteralPath $SourceDirectory
        foreach ($entry in Get-ChildItem -LiteralPath $SourceDirectory -Force) {
            Add-ZipEntry -Archive $archive -Item $entry -Root $rootInfo.FullName
        }
    }
    finally {
        $archive.Dispose()
    }
}

function Write-PackagedConfig {
    param(
        [Parameter(Mandatory = $true)][string]$Path
    )

    $content = @"
server:
  host: "127.0.0.1"
  port: 38080
  env: "production"

database:
  type: "sqlite"
  path: "./storage/acgwarehouse.db"
  connection_string: ""

storage:
  scan_roots:
    - "./library"

ai:
  provider: "qwen"
  api_key: ""
  model: "qwen-plus"
  requests_per_minute: 60
  max_concurrency: 3
  auto_ai_tag_on_import: true
  auto_scan_interval_minutes: 5
  auto_scan_batch_size: 100

admin:
  username: "admin"
  password: "admin"

worker_pool:
  worker_count: 4
  queue_size: 512
  refill_interval_seconds: 5
  refill_threshold: 0.5
  refill_batch_size: 0
"@

    Set-Content -LiteralPath $Path -Value $content -Encoding UTF8
}

function Add-ZipEntry {
    param(
        [Parameter(Mandatory = $true)]$Archive,
        [Parameter(Mandatory = $true)]$Item,
        [Parameter(Mandatory = $true)][string]$Root
    )

    $relativePath = $Item.FullName.Substring($Root.Length).TrimStart('\').Replace('\', '/')
    if ($Item.PSIsContainer) {
        $Archive.CreateEntry("$relativePath/") | Out-Null
        foreach ($child in Get-ChildItem -LiteralPath $Item.FullName -Force) {
            Add-ZipEntry -Archive $Archive -Item $child -Root $Root
        }
        return
    }

    [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($Archive, $Item.FullName, $relativePath, [System.IO.Compression.CompressionLevel]::Optimal) | Out-Null
}

Assert-Command -Name 'go'
Assert-Command -Name 'python'
Assert-Command -Name 'flutter'
Ensure-PythonModule -ModuleName 'PyInstaller'

Reset-Directory -Path $PortableRoot
Reset-Directory -Path $ZipOutputDir
Reset-Directory -Path $PyInstallerWorkDir

Ensure-Directory -Path (Join-Path $PortableRoot 'runtime/bin')
Ensure-Directory -Path (Join-Path $PortableRoot 'runtime/logs')
Ensure-Directory -Path (Join-Path $PortableRoot 'runtime/diagnostics')
Ensure-Directory -Path (Join-Path $PortableRoot 'data')
Ensure-Directory -Path (Join-Path $PortableRoot 'storage')
Ensure-Directory -Path (Join-Path $PortableRoot 'library')

Invoke-External -FilePath 'go' -Arguments @('build', '-o', $GoOutputExecutable, './cmd/server')

Invoke-External -FilePath 'python' -Arguments @(
    '-m',
    'PyInstaller',
    '--noconfirm',
    '--clean',
    '--distpath', $PyInstallerDistRoot,
    '--workpath', $PyInstallerWorkDir,
    $SidecarSpecPath
) -WorkingDirectory $RepoRoot

Write-Host "Running $FlutterBuildCommand"
Invoke-External -FilePath 'flutter' -Arguments @('build', 'windows', '--release') -WorkingDirectory $FlutterProjectDir

Copy-DirectoryContents -Source $FlutterReleaseDir -Destination $PortableRoot

if (-not (Test-Path -LiteralPath $FlutterOutputExecutable)) {
    throw "Expected Flutter Windows executable not found: $FlutterOutputExecutable"
}

$PortableExecutable = Join-Path $PortableRoot 'ACGWarehouse.exe'
if (Test-Path -LiteralPath $PortableExecutable) {
    Remove-Item -LiteralPath $PortableExecutable -Force
}
Move-Item -LiteralPath (Join-Path $PortableRoot 'gallery.exe') -Destination $PortableExecutable -Force

Copy-DirectoryContents -Source $ConfigSourceDir -Destination (Join-Path $PortableRoot 'config')
Write-PackagedConfig -Path $PackagedConfigPath

New-ZipFromDirectory -SourceDirectory $PortableRoot -DestinationZip $ZipPath

Write-Host "Windows portable package assembled at: $PortableRoot"
Write-Host "Windows portable ZIP created at: $ZipPath"
