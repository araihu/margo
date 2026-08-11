param(
  [switch]$ProvisionNode,
  [switch]$Check,
  [string]$CacheRoot,
  [string]$BrowserReceipt,
  [switch]$EmitEnvironmentJson
)
$ErrorActionPreference = "Stop"
if ($ProvisionNode -eq $Check) { throw "margo.node_mode_required" }
if (-not [IO.Path]::IsPathRooted($CacheRoot) -or -not [IO.Path]::IsPathRooted($BrowserReceipt)) { throw "margo.node_absolute_path_required" }
$scriptRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$lock = Get-Content -Raw (Join-Path $scriptRoot "node-toolchain.lock") | ConvertFrom-Json
if ($lock.schemaVersion -ne "margo/node-toolchain/v1") { throw "margo.node_lock_schema" }
$runner = "windows-x64"
$entry = @($lock.runners | Where-Object id -eq $runner)
if ($entry.Count -ne 1) { throw "margo.node_runner_unrecorded" }
$entry = $entry[0]
$installRoot = Join-Path $scriptRoot (".cache\node\{0}\{1}" -f $lock.nodeVersion, $runner)
$downloads = Join-Path $scriptRoot (".cache\downloads\node\{0}" -f $lock.nodeVersion)
$keyring = Join-Path $scriptRoot ".cache\node-keyring"
$archive = Join-Path $downloads ([IO.Path]::GetFileName($entry.archiveURL))
$node = [IO.Path]::GetFullPath((Join-Path $installRoot (Join-Path $entry.archiveRoot $entry.nodePath)))
$npm = [IO.Path]::GetFullPath((Join-Path $installRoot (Join-Path $entry.archiveRoot $entry.npmCLIPath)))

function Assert-SHA256([string]$Path, [string]$Expected, [string]$Code) {
  if (-not (Test-Path $Path) -or (Get-FileHash -Algorithm SHA256 $Path).Hash.ToLowerInvariant() -ne $Expected) { throw $Code }
}
function Assert-BrowserReceipt {
  $record = Get-Content -Raw $BrowserReceipt | ConvertFrom-Json
  $browserLock = Get-Content -Raw (Join-Path $scriptRoot "browser-lock.json") | ConvertFrom-Json
  $browserEntry = @($browserLock.runners | Where-Object id -eq $runner)
  $actualFields = @($record.PSObject.Properties.Name | Sort-Object) -join "`n"
  $expectedFieldList = @("archiveURL","archiveSHA256","executable","executableSHA256","revision","runner","schemaVersion","version")
  $expectedFields = @($expectedFieldList | Sort-Object) -join "`n"
  if ($actualFields -ne $expectedFields -or $browserEntry.Count -ne 1) { throw "margo.browser_receipt_fields" }
  if ($record.schemaVersion -ne "margo/browser-install/v1" -or $record.runner -ne $runner -or $record.revision -ne "1193" -or $record.version -ne "140.0.7339.186" -or $record.archiveURL -ne $browserEntry[0].urls[0] -or $record.archiveSHA256 -ne $browserEntry[0].sha256) { throw "margo.browser_receipt_mismatch" }
  if ([IO.Path]::GetFullPath($record.executable) -ne $record.executable) { throw "margo.browser_executable_path_mismatch" }
  Assert-SHA256 $record.executable $record.executableSHA256 "margo.browser_executable_digest_mismatch"
  return $record
}

function Assert-ToolchainProvenance {
  $keyFile = Join-Path $keyring "nodejs-release-keys.kbx"
  $manifestFile = Join-Path $downloads "SHASUMS256.txt"
  $signatureFile = Join-Path $downloads "SHASUMS256.txt.sig"
  $npmSource = Join-Path $downloads "npm.tgz"
  Assert-SHA256 $keyFile $lock.manifest.keySourceSHA256 "margo.node_keyring_digest_mismatch"
  Assert-SHA256 $manifestFile $lock.manifest.sha256 "margo.node_manifest_digest_mismatch"
  Assert-SHA256 $signatureFile $lock.manifest.signatureSHA256 "margo.node_signature_digest_mismatch"
  Assert-SHA256 $archive $entry.archiveSHA256 "margo.node_archive_digest_mismatch"
  if (-not (Test-Path $npmSource)) { throw "margo.npm_source_missing" }
  $npmDigest = [Convert]::ToBase64String([Security.Cryptography.SHA512]::HashData([IO.File]::ReadAllBytes($npmSource)))
  if ("sha512-$npmDigest" -ne $lock.npmSource.integrity) { throw "margo.npm_source_integrity_mismatch" }
  $gpgStatus = & gpgv --homedir $keyring --keyring $keyFile --status-fd 1 $signatureFile $manifestFile 2>$null
  if ($LASTEXITCODE -ne 0 -or ($gpgStatus -join "`n") -notmatch ("VALIDSIG " + $lock.manifest.signingKeyFingerprint) -or ($gpgStatus -join "`n") -notmatch "GOODSIG") { throw "margo.node_signature_invalid" }
  if ((Get-Content -Raw $manifestFile) -notmatch ([regex]::Escape($entry.archiveSHA256 + "  " + [IO.Path]::GetFileName($entry.archiveURL)))) { throw "margo.node_archive_not_signed" }
}

if ($ProvisionNode) {
  New-Item -ItemType Directory -Force $installRoot, $downloads, $keyring | Out-Null
  $keyFile = Join-Path $keyring "nodejs-release-keys.kbx"
  $manifestFile = Join-Path $downloads "SHASUMS256.txt"
  $signatureFile = Join-Path $downloads "SHASUMS256.txt.sig"
  $npmSource = Join-Path $downloads "npm.tgz"
  Invoke-WebRequest -UseBasicParsing -Uri $lock.manifest.keySourceURL -OutFile $keyFile
  Invoke-WebRequest -UseBasicParsing -Uri $lock.manifest.url -OutFile $manifestFile
  Invoke-WebRequest -UseBasicParsing -Uri $lock.manifest.signatureURL -OutFile $signatureFile
  Invoke-WebRequest -UseBasicParsing -Uri $lock.npmSource.url -OutFile $npmSource
  Invoke-WebRequest -UseBasicParsing -Uri $entry.archiveURL -OutFile $archive
  if (-not (Test-Path (Join-Path $installRoot $entry.archiveRoot))) { Expand-Archive -LiteralPath $archive -DestinationPath $installRoot }
}

Assert-ToolchainProvenance
Assert-SHA256 $node $entry.nodeSHA256 "margo.node_executable_digest_mismatch"
Assert-SHA256 $npm $entry.npmSHA256 "margo.npm_executable_digest_mismatch"
if ((& $node --version) -ne $lock.nodeVersion) { throw "margo.node_version_mismatch" }
if ((& $node $npm --version) -ne $lock.npmVersion) { throw "margo.npm_version_mismatch" }
$browser = Assert-BrowserReceipt

if ($Check) {
  $cacheReceipt = Join-Path $CacheRoot "receipt.json"
  if (-not (Test-Path $cacheReceipt)) { throw "margo.npm_cache_receipt_missing" }
  $env:MARGO_NODE_BIN = $node
  $env:MARGO_NPM_BIN = $npm
  & $node (Join-Path $scriptRoot "populate-npm-cache.mjs") --check --lock (Join-Path $scriptRoot "package-lock.json") --cache $CacheRoot --receipt $cacheReceipt
  if ($LASTEXITCODE -ne 0) { throw "margo.npm_cache_check_failed" }
}

if ($EmitEnvironmentJson) {
  $environment = [ordered]@{
    MARGO_NODE_BIN = $node
    MARGO_NPM_BIN = $npm
    MARGO_PLAYWRIGHT_CLI = [IO.Path]::GetFullPath((Join-Path $scriptRoot "node_modules\@playwright\test\cli.js"))
    MARGO_CHROMIUM_EXECUTABLE = $browser.executable
    MARGO_NPM_CACHE = [IO.Path]::GetFullPath($CacheRoot)
    MARGO_NPM_CACHE_RECEIPT = [IO.Path]::GetFullPath((Join-Path $CacheRoot "receipt.json"))
    MARGO_NODE_SHA256 = $entry.nodeSHA256
    MARGO_NPM_SHA256 = $entry.npmSHA256
    MARGO_CHROMIUM_SHA256 = $browser.executableSHA256
    MARGO_CHROMIUM_REVISION = $browser.revision
    MARGO_CHROMIUM_VERSION = $browser.version
  }
  $environment | ConvertTo-Json -Compress
}
