param(
    [Parameter(Position = 0)]
    [string]$VersionArg = ""
)

$ErrorActionPreference = 'Stop'

$Repo = "neko233-com/rsync233"
$Branch = "main"
$VersionFile = "version.txt"

function Normalize-Version([string]$Value) {
    $v = $Value.Trim()
    while ($v.StartsWith('v') -or $v.StartsWith('V')) { $v = $v.Substring(1) }
    return $v
}

function Format-Version([string]$Value) {
    return "v$(Normalize-Version $Value)"
}

function Next-PatchVersion([string]$Current) {
    if ([string]::IsNullOrWhiteSpace($Current)) { return "v0.1.0" }
    $base = Normalize-Version $Current
    $parts = $base -split '\.'
    if ($parts.Length -lt 3) { return "v0.1.0" }
    $patch = [int]$parts[2] + 1
    return "v$($parts[0]).$($parts[1]).$patch"
}

$current = ""
if (Test-Path -LiteralPath $VersionFile) {
    $current = (Get-Content -LiteralPath $VersionFile -Raw).Trim()
}

$newVersion = if ($VersionArg) {
    Format-Version $VersionArg
} else {
    Next-PatchVersion $current
}

Write-Host "========================================"
Write-Host "  rsync233 Release Deploy"
Write-Host "========================================"
Write-Host "Repository:  $Repo"
Write-Host "Branch:      $Branch"
Write-Host "Version:     $newVersion"
Write-Host "========================================"

Write-Host "[1/6] Running tests..."
cmd /c test.cmd
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "[2/6] Updating $VersionFile..."
Set-Content -LiteralPath $VersionFile -Value $newVersion -NoNewline

Write-Host "[3/6] Building release artifacts..."
cmd /c deploy.cmd $newVersion
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "[4/6] Committing release metadata..."
git add -A
git diff --cached --quiet
if ($LASTEXITCODE -eq 1) {
    git commit -m "chore: release $newVersion"
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} else {
    Write-Host "No metadata changes to commit."
}

Write-Host "[5/6] Tagging $newVersion..."
git rev-parse -q --verify "refs/tags/$newVersion" *> $null
if ($LASTEXITCODE -eq 0) {
    git tag -d $newVersion | Out-Null
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
git tag -a $newVersion -m "Release $newVersion"
if ($LASTEXITCODE -ne 0) {
    git tag -f -a $newVersion -m "Release $newVersion"
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "[6/6] Pushing branch and tag..."
git push origin $Branch
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
git push origin $newVersion
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Release tag pushed. Check: https://github.com/$Repo/actions"
