param(
  [switch]$Check,
  [string]$EnvironmentJson,
  [string]$Grep,
  [string]$ContrastHtml,
  [ValidateSet("light", "dark", "both")][string]$ContrastMode = "both",
  [ValidateSet("json", "text")][string]$ContrastFormat = "json",
  [string]$ContrastOutput,
  [switch]$ContrastOnly
)
$ErrorActionPreference = "Stop"
if (-not $Check) { throw "margo.harness_check_required" }
if (-not [IO.Path]::IsPathRooted($EnvironmentJson) -or -not (Test-Path $EnvironmentJson)) { throw "margo.harness_env_missing" }
if ($ContrastOnly -and -not $ContrastHtml) { throw "margo.contrast_lint_html_required" }
if ($ContrastHtml -and (-not [IO.Path]::IsPathRooted($ContrastHtml) -or -not (Test-Path $ContrastHtml))) { throw "margo.contrast_lint_html_missing" }
if ($ContrastOutput -and -not [IO.Path]::IsPathRooted($ContrastOutput)) { throw "margo.contrast_lint_output_absolute_required" }
$scriptRoot = [IO.Path]::GetFullPath($PSScriptRoot)
$environment = Get-Content -Raw $EnvironmentJson | ConvertFrom-Json
$keys = @("MARGO_NODE_BIN","MARGO_NPM_BIN","MARGO_PLAYWRIGHT_CLI","MARGO_CHROMIUM_EXECUTABLE","MARGO_NPM_CACHE","MARGO_NPM_CACHE_RECEIPT","MARGO_NODE_SHA256","MARGO_NPM_SHA256","MARGO_CHROMIUM_SHA256","MARGO_CHROMIUM_REVISION","MARGO_CHROMIUM_VERSION")
$actualKeys = @($environment.PSObject.Properties.Name | Sort-Object) -join "`n"
$expectedKeys = @($keys | Sort-Object) -join "`n"
if ($actualKeys -ne $expectedKeys) { throw "margo.harness_env_schema" }
foreach ($key in $keys) {
  $existing = [Environment]::GetEnvironmentVariable($key)
  if ($existing -and $existing -ne $environment.$key) { throw "margo.harness_ambient_conflict:$key" }
  [Environment]::SetEnvironmentVariable($key, [string]$environment.$key)
}
foreach ($path in @($environment.MARGO_NODE_BIN,$environment.MARGO_NPM_BIN,$environment.MARGO_CHROMIUM_EXECUTABLE,$environment.MARGO_NPM_CACHE,$environment.MARGO_NPM_CACHE_RECEIPT)) {
  if (-not [IO.Path]::IsPathRooted($path)) { throw "margo.harness_path_not_absolute" }
}
function Assert-SHA256([string]$Path, [string]$Expected, [string]$Code) {
  if (-not (Test-Path $Path) -or (Get-FileHash -Algorithm SHA256 $Path).Hash.ToLowerInvariant() -ne $Expected) { throw $Code }
}
Assert-SHA256 $environment.MARGO_NODE_BIN $environment.MARGO_NODE_SHA256 "margo.harness_node_digest_mismatch"
Assert-SHA256 $environment.MARGO_NPM_BIN $environment.MARGO_NPM_SHA256 "margo.harness_npm_digest_mismatch"
Assert-SHA256 $environment.MARGO_CHROMIUM_EXECUTABLE $environment.MARGO_CHROMIUM_SHA256 "margo.harness_chromium_digest_mismatch"
if ($environment.MARGO_CHROMIUM_REVISION -ne "1169" -or $environment.MARGO_CHROMIUM_VERSION -ne "136.0.7103.25") { throw "margo.harness_chromium_identity_mismatch" }
& $environment.MARGO_NODE_BIN (Join-Path $scriptRoot "populate-npm-cache.mjs") --check --lock (Join-Path $scriptRoot "package-lock.json") --cache $environment.MARGO_NPM_CACHE --receipt $environment.MARGO_NPM_CACHE_RECEIPT
if ($LASTEXITCODE -ne 0) { throw "margo.npm_cache_check_failed" }
$env:PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1"
$env:PLAYWRIGHT_BROWSERS_PATH = "0"
$env:PW_DISABLE_TS_ESM = "1"
& $environment.MARGO_NODE_BIN $environment.MARGO_NPM_BIN ci --offline --cache $environment.MARGO_NPM_CACHE --ignore-scripts --no-audit --no-fund --logs-max=0 --update-notifier=false
if ($LASTEXITCODE -ne 0) { throw "margo.harness_npm_ci_failed" }
if (-not (Test-Path $environment.MARGO_PLAYWRIGHT_CLI)) { throw "margo.harness_playwright_cli_missing" }
if ($ContrastHtml) {
  $contrastArgs = @((Join-Path $scriptRoot "lint-contrast.mjs"), "--html", $ContrastHtml, "--mode", $ContrastMode, "--format", $ContrastFormat)
  if ($ContrastOutput) { $contrastArgs += @("--output", $ContrastOutput) }
  & $environment.MARGO_NODE_BIN @contrastArgs
  if ($LASTEXITCODE -ne 0) { throw "margo.contrast_lint_failed" }
}
if ($ContrastOnly) {
  Write-Output ("margo.harness_contrast_ok node={0} chromium_revision={1} chromium_version={2} network=0" -f (& $environment.MARGO_NODE_BIN --version), $environment.MARGO_CHROMIUM_REVISION, $environment.MARGO_CHROMIUM_VERSION)
  exit 0
}
& $environment.MARGO_NODE_BIN $environment.MARGO_PLAYWRIGHT_CLI test --config (Join-Path $scriptRoot "playwright.config.mjs") --grep $Grep
if ($LASTEXITCODE -ne 0) { throw "margo.harness_playwright_failed" }
Write-Output ("margo.harness_ok node={0} npm={1} chromium_revision={2} chromium_version={3} network=0" -f (& $environment.MARGO_NODE_BIN --version), (& $environment.MARGO_NODE_BIN $environment.MARGO_NPM_BIN --version), $environment.MARGO_CHROMIUM_REVISION, $environment.MARGO_CHROMIUM_VERSION)
