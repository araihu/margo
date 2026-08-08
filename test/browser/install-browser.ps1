param(
  [switch]$Provision,
  [string]$Runner,
  [string]$Receipt
)
$ErrorActionPreference = "Stop"
if (-not $Provision) { throw "margo.browser_provision_required" }
if ($Runner -ne "windows-x64") { throw "margo.browser_runner_unrecorded" }
if (-not [IO.Path]::IsPathRooted($Receipt)) { throw "margo.browser_receipt_absolute_required" }

$scriptRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$lock = Get-Content -Raw (Join-Path $scriptRoot "browser-lock.json") | ConvertFrom-Json
if ($lock.schemaVersion -ne "margo/browser-lock/v1") { throw "margo.browser_lock_schema" }
$entry = @($lock.runners | Where-Object id -eq $Runner)
if ($entry.Count -ne 1) { throw "margo.browser_runner_unrecorded" }
$entry = $entry[0]
$downloads = Join-Path $scriptRoot ".cache\downloads"
$archive = Join-Path $downloads $entry.archive
$installRoot = Join-Path $scriptRoot (".cache\playwright\{0}\{1}" -f $Runner, $lock.revision)
New-Item -ItemType Directory -Force $downloads, (Split-Path $Receipt) | Out-Null

$validArchive = (Test-Path $archive) -and ((Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant() -eq $entry.sha256)
if (-not $validArchive) {
  $part = "$archive.part"
  $downloaded = $false
  foreach ($url in $entry.urls) {
    try {
      Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $part
      if ((Get-FileHash -Algorithm SHA256 $part).Hash.ToLowerInvariant() -eq $entry.sha256) {
        Move-Item -Force $part $archive
        $downloaded = $true
        break
      }
    } catch { Remove-Item $part -ErrorAction SilentlyContinue }
  }
  if (-not $downloaded) { throw "margo.browser_archive_unavailable" }
}
if ((Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant() -ne $entry.sha256) { throw "margo.browser_archive_digest_mismatch" }
if (-not (Test-Path $installRoot)) {
  $temporary = "$installRoot.tmp"
  New-Item -ItemType Directory -Force $temporary | Out-Null
  Expand-Archive -LiteralPath $archive -DestinationPath $temporary
  Move-Item $temporary $installRoot
}
$executable = [IO.Path]::GetFullPath((Join-Path $installRoot $entry.executablePath))
if (-not (Test-Path $executable)) { throw "margo.browser_executable_missing" }
$executableSHA = (Get-FileHash -Algorithm SHA256 $executable).Hash.ToLowerInvariant()
$record = [ordered]@{
  archiveURL = $entry.urls[0]
  archiveSHA256 = $entry.sha256
  executable = $executable
  executableSHA256 = $executableSHA
  revision = [string]$lock.revision
  runner = $Runner
  schemaVersion = "margo/browser-install/v1"
  version = [string]$lock.version
}
if (Test-Path $Receipt) { throw "margo.browser_receipt_exists" }
[IO.File]::WriteAllText($Receipt, ($record | ConvertTo-Json -Compress), [Text.UTF8Encoding]::new($false))
