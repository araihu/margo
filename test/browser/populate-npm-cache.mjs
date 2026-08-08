import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { mkdir, readFile, readdir, realpath, writeFile } from "node:fs/promises";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { spawn } from "node:child_process";

const RECEIPT_SCHEMA = "margo/npm-cache/v1";

function fail(code) {
  throw new Error(code);
}

function parseArgs(argv) {
  const result = { provision: false, check: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--provision") result.provision = true;
    else if (arg === "--check") result.check = true;
    else if (["--lock", "--cache", "--receipt"].includes(arg)) result[arg.slice(2)] = argv[++index];
    else fail(`margo.npm_cache_argument_unknown:${arg}`);
  }
  if (result.provision === result.check) fail("margo.npm_cache_mode_required");
  for (const key of ["lock", "cache", "receipt"]) {
    if (!result[key]) fail(`margo.npm_cache_${key}_required`);
  }
  if (!isAbsolute(result.cache) || !isAbsolute(result.receipt)) fail("margo.npm_cache_absolute_path_required");
  if (result.receipt !== join(result.cache, "receipt.json")) fail("margo.npm_cache_receipt_path_mismatch");
  return result;
}

function packageName(lockPath, value) {
  if (lockPath === "") return value.name;
  return lockPath.split("node_modules/").at(-1);
}

function lockPackages(lock) {
  const rows = [];
  const identities = new Set();
  for (const [lockPath, value] of Object.entries(lock.packages ?? {})) {
    const name = packageName(lockPath, value);
    if (!name || !value.version) fail(`margo.npm_cache_lock_identity_missing:${lockPath}`);
    const identity = `${name}@${value.version}`;
    if (identities.has(identity)) fail(`margo.npm_cache_lock_duplicate:${identity}`);
    identities.add(identity);
    if (lockPath === "") {
      rows.push({ name, version: value.version, lockPath, omittedReason: "root-project" });
      continue;
    }
    if (!value.resolved || !value.integrity) fail(`margo.npm_cache_lock_provenance_missing:${lockPath}`);
    const url = new URL(value.resolved);
    if (url.protocol !== "https:" || url.hostname !== "registry.npmjs.org" || url.toString() !== value.resolved) {
      fail(`margo.npm_cache_registry_forbidden:${value.resolved}`);
    }
    if (!value.integrity.startsWith("sha512-")) fail(`margo.npm_cache_integrity_invalid:${lockPath}`);
    rows.push({ name, version: value.version, lockPath, resolvedURL: value.resolved, integrity: value.integrity });
  }
  rows.sort((left, right) => left.lockPath < right.lockPath ? -1 : left.lockPath > right.lockPath ? 1 : 0);
  if (rows.length < 2 || rows[0].lockPath !== "") fail("margo.npm_cache_lock_empty");
  return rows;
}

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

function sha512(bytes) {
  return createHash("sha512").update(bytes).digest();
}

function canonicalJSON(value) {
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  if (value !== null && typeof value === "object") {
    const keys = Object.keys(value).sort();
    return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

function assertExactKeys(value, expected, code) {
  const actual = Object.keys(value).sort().join("\n");
  const wanted = [...expected].sort().join("\n");
  if (actual !== wanted) fail(code);
}

function npmInternals() {
  const npmBin = process.env.MARGO_NPM_BIN;
  if (!npmBin || !isAbsolute(npmBin)) fail("margo.npm_cache_verified_npm_required");
  const npmRoot = resolve(dirname(npmBin), "..");
  const require = createRequire(join(npmRoot, "package.json"));
  return {
    cacache: require(join(npmRoot, "node_modules", "cacache")),
    cacheKey: require(join(npmRoot, "node_modules", "make-fetch-happen", "lib", "cache", "key.js")),
  };
}

async function runNpm(args, env = process.env) {
  const nodeBin = process.env.MARGO_NODE_BIN;
  const npmBin = process.env.MARGO_NPM_BIN;
  if (!nodeBin || !isAbsolute(nodeBin) || !npmBin || !isAbsolute(npmBin)) fail("margo.npm_cache_verified_npm_required");
  return new Promise((resolvePromise, reject) => {
    const child = spawn(nodeBin, [npmBin, ...args], { env, stdio: "pipe", shell: false });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => { stdout += chunk; });
    child.stderr.on("data", (chunk) => { stderr += chunk; });
    child.once("error", reject);
    child.once("exit", (code) => {
      if (code === 0) resolvePromise(stdout.trim());
      else reject(new Error(`margo.npm_cache_npm_failed:${code}:${stderr.trim()}`));
    });
  });
}

async function downloadPackage(entry) {
  const response = await fetch(entry.resolvedURL, { redirect: "error" });
  if (!response.ok) fail(`margo.npm_cache_download_failed:${response.status}:${entry.lockPath}`);
  const bytes = Buffer.from(await response.arrayBuffer());
  const digest = sha512(bytes);
  const expected = Buffer.from(entry.integrity.slice("sha512-".length), "base64");
  if (digest.length !== expected.length || !digest.equals(expected)) fail(`margo.npm_cache_integrity_mismatch:${entry.lockPath}`);
  return bytes;
}

function receiptWithoutDigest(receipt) {
  const { receiptSHA256: ignored, ...preimage } = receipt;
  return preimage;
}

async function verifyReceipt(args, lockBytes, lockRows) {
  let receiptBytes;
  try {
    receiptBytes = await readFile(args.receipt);
  } catch {
    fail("margo.npm_cache_receipt_missing");
  }
  const receipt = JSON.parse(receiptBytes);
  if (canonicalJSON(receipt) !== receiptBytes.toString("utf8")) fail("margo.npm_cache_receipt_noncanonical");
  assertExactKeys(receipt, ["schemaVersion", "lockfileSHA256", "nodeVersion", "npmVersion", "cacheRoot", "receiptSHA256", "packages"], "margo.npm_cache_receipt_fields");
  if (receipt.schemaVersion !== RECEIPT_SCHEMA) fail("margo.npm_cache_receipt_schema");
  if (receipt.cacheRoot !== await realpath(args.cache)) fail("margo.npm_cache_root_mismatch");
  if (receipt.lockfileSHA256 !== sha256(lockBytes)) fail("margo.npm_cache_lock_mismatch");
  if (receipt.nodeVersion !== process.version) fail("margo.npm_cache_node_version_mismatch");
  if (receipt.npmVersion !== await runNpm(["--version"])) fail("margo.npm_cache_npm_version_mismatch");
  if (!/^[0-9a-f]{64}$/.test(receipt.receiptSHA256)) fail("margo.npm_cache_receipt_digest_invalid");
  if (receipt.receiptSHA256 !== sha256(Buffer.from(canonicalJSON(receiptWithoutDigest(receipt))))) fail("margo.npm_cache_receipt_digest_mismatch");
  if (!Array.isArray(receipt.packages) || receipt.packages.length !== lockRows.length) fail("margo.npm_cache_package_count_mismatch");

  const expectedCacheKeys = new Set();
  const { cacache, cacheKey } = npmInternals();
  const cachePath = join(args.cache, "_cacache");
  for (let index = 0; index < lockRows.length; index += 1) {
    const expected = lockRows[index];
    const actual = receipt.packages[index];
    if (expected.lockPath === "") {
      assertExactKeys(actual, ["name", "version", "lockPath", "omittedReason"], "margo.npm_cache_root_receipt_fields");
      if (canonicalJSON(actual) !== canonicalJSON(expected)) fail("margo.npm_cache_root_receipt_mismatch");
      continue;
    }
    assertExactKeys(actual, ["name", "version", "lockPath", "resolvedURL", "integrity", "bytes", "sha256", "cacheKey"], "margo.npm_cache_package_receipt_fields");
    const key = cacheKey({ url: expected.resolvedURL });
    expectedCacheKeys.add(key);
    for (const field of ["name", "version", "lockPath", "resolvedURL", "integrity"]) {
      if (actual[field] !== expected[field]) fail(`margo.npm_cache_package_mismatch:${expected.lockPath}:${field}`);
    }
    if (actual.cacheKey !== key || !Number.isInteger(actual.bytes) || actual.bytes < 1 || !/^[0-9a-f]{64}$/.test(actual.sha256)) {
      fail(`margo.npm_cache_package_receipt_invalid:${expected.lockPath}`);
    }
    let cached;
    try {
      cached = await cacache.get(cachePath, key, { integrity: actual.integrity });
    } catch {
      fail(`margo.npm_cache_package_missing_or_tampered:${expected.lockPath}`);
    }
    if (cached.data.length !== actual.bytes || sha256(cached.data) !== actual.sha256) fail(`margo.npm_cache_package_bytes_mismatch:${expected.lockPath}`);
  }

  const actualCacheKeys = Object.keys(await cacache.ls(cachePath)).sort();
  const expectedKeys = [...expectedCacheKeys].sort();
  if (canonicalJSON(actualCacheKeys) !== canonicalJSON(expectedKeys)) fail("margo.npm_cache_entry_set_mismatch");
  return receipt;
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const lockBytes = await readFile(resolve(args.lock));
  const lockRows = lockPackages(JSON.parse(lockBytes));

  if (args.check) {
    await verifyReceipt(args, lockBytes, lockRows);
    return;
  }

  await mkdir(args.cache, { recursive: true });
  const cacheRoot = await realpath(args.cache);
  if (cacheRoot !== args.cache) fail("margo.npm_cache_path_not_canonical");
  if ((await readdir(cacheRoot)).length !== 0) fail("margo.npm_cache_root_not_empty");

  const { cacache, cacheKey } = npmInternals();
  const cachePath = join(cacheRoot, "_cacache");
  const receiptPackages = [];
  for (const entry of lockRows) {
    if (entry.lockPath === "") {
      receiptPackages.push(entry);
      continue;
    }
    const bytes = await downloadPackage(entry);
    const key = cacheKey({ url: entry.resolvedURL });
    await cacache.put(cachePath, key, bytes, {
      integrity: entry.integrity,
      size: bytes.length,
      metadata: {
        time: Date.now(),
        url: entry.resolvedURL,
        reqHeaders: {},
        resHeaders: { "cache-control": "public, max-age=31536000, immutable", "content-type": "application/octet-stream" },
        options: { compress: true },
      },
    });
    receiptPackages.push({ ...entry, bytes: bytes.length, sha256: sha256(bytes), cacheKey: key });
  }

  const preimage = {
    schemaVersion: RECEIPT_SCHEMA,
    lockfileSHA256: sha256(lockBytes),
    nodeVersion: process.version,
    npmVersion: await runNpm(["--version"]),
    cacheRoot,
    packages: receiptPackages,
  };
  const receipt = { ...preimage, receiptSHA256: sha256(Buffer.from(canonicalJSON(preimage))) };
  await writeFile(args.receipt, canonicalJSON(receipt), { flag: "wx" });
  await verifyReceipt(args, lockBytes, lockRows);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
});
