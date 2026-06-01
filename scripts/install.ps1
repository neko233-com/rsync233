param(
    [Parameter(Position = 0)]
    [string]$Version = "latest"
)

$ErrorActionPreference = 'Stop'

# rsync233 - Windows installer (no GitHub API; avoids rate limits)
# Usage: irm https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.ps1 | iex
# Or:    irm .../install.ps1 -OutFile install.ps1; .\install.ps1 v1.0.0

$BinaryName = "rsync233"
$Repo = "neko233-com/rsync233"
$InstallDir = Join-Path $env:LOCALAPPDATA $BinaryName

function Get-NormalizedVersion([string]$Value) {
    $v = $Value.Trim()
    while ($v.StartsWith('v') -or $v.StartsWith('V')) { $v = $v.Substring(1) }
    return $v
}

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($arch -eq "ARM64") { return "arm64" }
    return "amd64"
}

function Test-PathInUserPath([string]$Dir) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrWhiteSpace($userPath)) { return $false }

    $normalizedDir = (Resolve-Path -LiteralPath $Dir).Path.TrimEnd('\')
    foreach ($entry in $userPath -split ';') {
        if ([string]::IsNullOrWhiteSpace($entry)) { continue }
        try {
            $normalizedEntry = (Resolve-Path -LiteralPath $entry -ErrorAction Stop).Path.TrimEnd('\')
            if ($normalizedEntry -ieq $normalizedDir) { return $true }
        } catch {
            if ($entry.TrimEnd('\') -ieq $normalizedDir) { return $true }
        }
    }
    return $false
}

function Add-ToUserPath([string]$Dir) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $normalizedDir = (Resolve-Path -LiteralPath $Dir).Path

    if (Test-PathInUserPath $normalizedDir) { return $false }

    $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
        $normalizedDir
    } else {
        "$normalizedDir;$userPath"
    }

    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    return $true
}

function Add-BinaryLink([string]$Source, [string]$TargetDir) {
    $linkPath = Join-Path $TargetDir "$BinaryName.exe"
    if (Test-Path -LiteralPath $linkPath) {
        Remove-Item -LiteralPath $linkPath -Force
    }

    try {
        New-Item -ItemType HardLink -Path $linkPath -Target $Source -Force | Out-Null
        return $linkPath
    } catch {
        Copy-Item -LiteralPath $Source -Destination $linkPath -Force
        return $linkPath
    }
}

function Notify-PathChanged {
    $signature = @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
    try {
        Add-Type -MemberDefinition $signature -Name NativeMethods -Namespace Win32 -ErrorAction Stop
        $null = [UIntPtr]::Zero
        [Win32.NativeMethods]::SendMessageTimeout(
            [IntPtr]0xffff, 0x1A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$null) | Out-Null
    } catch {
        # Best-effort only; a new terminal still picks up registry PATH.
    }
}

function Assert-ReleaseExists([string]$VersionLabel, [string]$Arch) {
    if ($VersionLabel -eq "latest") {
        $apiUrl = "https://api.github.com/repos/$Repo/releases/latest"
    } else {
        $apiUrl = "https://api.github.com/repos/$Repo/releases/tags/$VersionLabel"
    }

    try {
        Invoke-WebRequest -UseBasicParsing -Headers @{
            Accept = "application/vnd.github+json"
            "User-Agent" = $BinaryName
        } -Uri $apiUrl | Out-Null
    } catch {
        $response = $_.Exception.Response
        if ($null -ne $response) {
            try {
                if ([int]$response.StatusCode -eq 404) {
                    throw "No GitHub Release asset was found for $VersionLabel on windows/$Arch. This installer downloads published release binaries, but this repository does not currently have that release. Install from source instead: go install github.com/$Repo/cmd/$BinaryName@latest"
                }
            } catch {
                if ($_.Exception.Message -like "No GitHub Release asset was found*") {
                    throw
                }
            }
        }

        # GitHub's API may return 403 for anonymous requests or transient network errors
        # even when the public release asset URL is reachable, so only 404 is fatal here.
        return
    }
}

function Invoke-ReleaseDownload([string]$Url, [string]$Dest, [string]$VersionLabel, [string]$Arch) {
    Assert-ReleaseExists -VersionLabel $VersionLabel -Arch $Arch

    try {
        Invoke-WebRequest -Uri $Url -OutFile $Dest
    } catch {
        throw "Download failed from $Url. $($_.Exception.Message)"
    }
}

$Arch = Get-Arch
$Asset = "${BinaryName}-windows-${Arch}.exe"
if ($Version -eq "latest" -or [string]::IsNullOrWhiteSpace($Version)) {
    $url = "https://github.com/$Repo/releases/latest/download/$Asset"
    $versionLabel = "latest"
} else {
    $Version = Get-NormalizedVersion $Version
    $url = "https://github.com/$Repo/releases/download/v$Version/$Asset"
    $versionLabel = "v$Version"
}

$dest = Join-Path $InstallDir "$BinaryName.exe"

Write-Host "Installing ${BinaryName} $versionLabel for windows/$Arch..."
Write-Host "Downloading $url..."

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Invoke-ReleaseDownload -Url $url -Dest $dest -VersionLabel $versionLabel -Arch $Arch

$pathCandidates = @(
    (Join-Path $env:USERPROFILE ".local\bin"),
    (Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Links"),
    (Join-Path $env:USERPROFILE "go\bin")
)

$linked = $false
foreach ($candidate in $pathCandidates) {
    if (-not (Test-Path -LiteralPath $candidate)) { continue }
    if (-not (Test-PathInUserPath $candidate)) { continue }

    $linkPath = Add-BinaryLink -Source $dest -TargetDir $candidate
    Write-Host "Linked $linkPath -> $dest"
    $linked = $true
    break
}

if (-not $linked) {
    if (Add-ToUserPath $InstallDir) {
        Write-Host "Added $InstallDir to the front of user PATH."
    } else {
        Write-Host "$InstallDir is already in user PATH."
    }
}

Notify-PathChanged
$env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [Environment]::GetEnvironmentVariable("Path", "User")

Write-Host ""
Write-Host "Installed to $dest"
Write-Host "Restart your terminal, then run: rsync233 version"
