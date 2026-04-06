[CmdletBinding()]
param(
    [switch]$SkipTests,
    [switch]$All,
    [switch]$Go,
    [switch]$Python,
    [switch]$Flutter
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$PortableRoot = Join-Path $RepoRoot 'dist/windows-portable'
$ZipOutputDir = Join-Path $RepoRoot 'dist/windows-zip'
$ZipPath = Join-Path $ZipOutputDir 'ACGWarehouse-windows-x64-portable.zip'
$BuildArtifactsRoot = Join-Path $RepoRoot 'dist/windows-build'
$GoBuildRoot = Join-Path $BuildArtifactsRoot 'go'
$PythonBuildRoot = Join-Path $BuildArtifactsRoot 'python'
$PyInstallerWorkDir = Join-Path $RepoRoot 'dist/.pyinstaller/work'
$PyInstallerDistRoot = $PythonBuildRoot
$FlutterProjectDir = Join-Path $RepoRoot 'flutter_app'
$FlutterReleaseDir = Join-Path $FlutterProjectDir 'build/windows/x64/runner/Release'
$FlutterOutputExecutable = Join-Path $FlutterReleaseDir 'gallery.exe'
$FlutterBuildCommand = 'flutter build windows --release'
$GoBuildExecutable = Join-Path $GoBuildRoot 'acgwarehouse-server.exe'
$PythonBuildDirectory = Join-Path $PythonBuildRoot 'python-sidecar'
$PythonBuildExecutable = Join-Path $PythonBuildDirectory 'acgwarehouse-sidecar.exe'
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

function Assert-PackagedRuntimeArtifacts {
    param(
        [Parameter(Mandatory = $true)][string]$BundleRoot,
        [Parameter(Mandatory = $true)][bool]$BuiltGo,
        [Parameter(Mandatory = $true)][bool]$BuiltPython
    )

    $requiredPaths = @(
        @{ Path = Join-Path $BundleRoot 'ACGWarehouse.exe'; Label = 'Flutter executable' },
        @{ Path = Join-Path $BundleRoot 'runtime/bin/acgwarehouse-server.exe'; Label = 'Go runtime executable' },
        @{ Path = Join-Path $BundleRoot 'runtime/python-sidecar/acgwarehouse-sidecar.exe'; Label = 'Python sidecar executable' }
    )

    $missing = @()
    foreach ($entry in $requiredPaths) {
        if (-not (Test-Path -LiteralPath $entry.Path)) {
            $missing += $entry.Label + ' (' + $entry.Path + ')'
        }
    }

    if ($missing.Count -eq 0) {
        return
    }

    $selectionHint = if (-not $BuiltGo -or -not $BuiltPython) {
        'package.ps1 resets dist/windows-portable before packaging, so selective runs like -Flutter alone cannot produce a launchable portable bundle. Rerun with the default command or include -Go -Python -Flutter together.'
    } else {
        'Rebuild the portable package and verify the runtime artifacts exist before launch.'
    }

    throw "Portable bundle is incomplete. Missing: $($missing -join '; '). $selectionHint"
}

function Assert-SourceArtifactExists {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][string]$RecoveryHint
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Missing reusable $Label at $Path. $RecoveryHint"
    }
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

cos:
  bucket_url: "https://acgwarehouse-1301393037.cos.ap-shanghai.myqcloud.com"
  secret_id: ""
  secret_key: ""

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

# Default behavior: compile all if no specific compile option specified
if (-not ($All -or $Go -or $Python -or $Flutter)) {
    $All = $true
}

Assert-Command -Name 'go'
Assert-Command -Name 'python'
Assert-Command -Name 'flutter'
Ensure-PythonModule -ModuleName 'PyInstaller'

Reset-Directory -Path $PortableRoot
Reset-Directory -Path $ZipOutputDir
Ensure-Directory -Path $BuildArtifactsRoot

Ensure-Directory -Path (Join-Path $PortableRoot 'runtime/bin')
Ensure-Directory -Path (Join-Path $PortableRoot 'runtime/logs')
Ensure-Directory -Path (Join-Path $PortableRoot 'runtime/diagnostics')
Ensure-Directory -Path (Join-Path $PortableRoot 'data')
Ensure-Directory -Path (Join-Path $PortableRoot 'storage')
Ensure-Directory -Path (Join-Path $PortableRoot 'library')

if ($All -or $Go) {
    Write-Host "Building Go server with optimizations..."
    # Build optimizations for reduced size and faster startup:
    # CGO_ENABLED=0: create static binary, avoid dynamic linker overhead
    # -ldflags "-s -w": strip symbol table and DWARF debug info (~30% size reduction)
    # -trimpath: remove file system paths for reproducible builds
    $env:CGO_ENABLED = '0'
    Invoke-External -FilePath 'go' -Arguments @(
        'build',
        '-ldflags', '-s -w',
        '-trimpath',
        '-o', $GoBuildExecutable,
        './cmd/server'
    )
}

Assert-SourceArtifactExists -Path $GoBuildExecutable -Label 'Go build artifact' -RecoveryHint 'Run the default packaging command once, or include -Go in this packaging run.'
Copy-Item -LiteralPath $GoBuildExecutable -Destination (Join-Path $PortableRoot 'runtime/bin/acgwarehouse-server.exe') -Force

if ($All -or $Python) {
    Write-Host "Building Python sidecar with PyInstaller..."
    Reset-Directory -Path $PyInstallerWorkDir
    Reset-Directory -Path $PyInstallerDistRoot
    Invoke-External -FilePath 'python' -Arguments @(
        '-m',
        'PyInstaller',
        '--noconfirm',
        '--clean',
        '--distpath', $PyInstallerDistRoot,
        '--workpath', $PyInstallerWorkDir,
        $SidecarSpecPath
    ) -WorkingDirectory $RepoRoot
}

Assert-SourceArtifactExists -Path $PythonBuildExecutable -Label 'Python sidecar build artifact' -RecoveryHint 'Run the default packaging command once, or include -Python in this packaging run.'
Copy-DirectoryContents -Source $PythonBuildDirectory -Destination (Join-Path $PortableRoot 'runtime/python-sidecar')

if ($All -or $Flutter) {
    Write-Host "Building Flutter Windows app..."
    Invoke-External -FilePath 'flutter' -Arguments @('build', 'windows', '--release') -WorkingDirectory $FlutterProjectDir
}

Assert-SourceArtifactExists -Path $FlutterOutputExecutable -Label 'Flutter Windows build artifact' -RecoveryHint 'Run the default packaging command once, or include -Flutter in this packaging run.'
Copy-DirectoryContents -Source $FlutterReleaseDir -Destination $PortableRoot

$PortableExecutable = Join-Path $PortableRoot 'ACGWarehouse.exe'
if (Test-Path -LiteralPath $PortableExecutable) {
    Remove-Item -LiteralPath $PortableExecutable -Force
}
Move-Item -LiteralPath (Join-Path $PortableRoot 'gallery.exe') -Destination $PortableExecutable -Force

Copy-DirectoryContents -Source $ConfigSourceDir -Destination (Join-Path $PortableRoot 'config')
Write-PackagedConfig -Path $PackagedConfigPath
Assert-PackagedRuntimeArtifacts -BundleRoot $PortableRoot -BuiltGo ([bool]($All -or $Go)) -BuiltPython ([bool]($All -or $Python))

New-ZipFromDirectory -SourceDirectory $PortableRoot -DestinationZip $ZipPath

Write-Host "Windows portable package assembled at: $PortableRoot"
Write-Host "Windows portable ZIP created at: $ZipPath"
