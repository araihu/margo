import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { cp, mkdtemp, readFile, realpath, rm, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";

const browserRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const sourceCache = process.env.MARGO_NPM_CACHE;
const nodeBin = process.env.MARGO_NODE_BIN;
const npmBin = process.env.MARGO_NPM_BIN;
const lockPath = join(browserRoot, "package-lock.json");
const verifier = join(browserRoot, "populate-npm-cache.mjs");
const scratchRoots = [];

test.afterAll(async () => {
  await Promise.all(scratchRoots.map((path) => rm(path, { recursive: true, force: true })));
});

function canonicalJSON(value) {
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  if (value !== null && typeof value === "object") {
    const keys = Object.keys(value).sort();
    return `{${keys.map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

async function runCheck(cache) {
  return new Promise((resolvePromise, reject) => {
    const child = spawn(nodeBin, [verifier, "--check", "--lock", lockPath, "--cache", cache, "--receipt", join(cache, "receipt.json")], {
      env: { ...process.env, MARGO_NODE_BIN: nodeBin, MARGO_NPM_BIN: npmBin },
      stdio: "pipe",
      shell: false,
    });
    let stderr = "";
    child.stderr.on("data", (chunk) => { stderr += chunk; });
    child.once("error", reject);
    child.once("exit", (code) => resolvePromise({ code, stderr: stderr.trim() }));
  });
}

async function bindReceipt(cache) {
  const path = join(cache, "receipt.json");
  const receipt = JSON.parse(await readFile(path, "utf8"));
  receipt.cacheRoot = await realpath(cache);
  const { receiptSHA256: ignored, ...preimage } = receipt;
  receipt.receiptSHA256 = sha256(Buffer.from(canonicalJSON(preimage)));
  await writeFile(path, canonicalJSON(receipt));
  return receipt;
}

async function cacheCopy(bind = true) {
  const parent = await mkdtemp(join(tmpdir(), "margo-cache-contract-"));
  scratchRoots.push(parent);
  const cache = join(parent, "cache");
  await cp(sourceCache, cache, { recursive: true });
  if (bind) await bindReceipt(cache);
  return cache;
}

function cacacheForVerifiedNpm() {
  const npmRoot = resolve(dirname(npmBin), "..");
  const require = createRequire(join(npmRoot, "package.json"));
  return require(join(npmRoot, "node_modules", "cacache"));
}

test("@margo-harness cache receipt and exact entries fail closed", async ({}, testInfo) => {
  testInfo.setTimeout(60_000);

  const valid = await cacheCopy();
  await expect(runCheck(valid)).resolves.toMatchObject({ code: 0, stderr: "" });

  const wrongRoot = await cacheCopy(false);
  await expect(runCheck(wrongRoot)).resolves.toMatchObject({ code: 1, stderr: "margo.npm_cache_root_mismatch" });

  const stale = await cacheCopy();
  const staleReceipt = JSON.parse(await readFile(join(stale, "receipt.json"), "utf8"));
  staleReceipt.lockfileSHA256 = "0".repeat(64);
  const { receiptSHA256: staleDigest, ...stalePreimage } = staleReceipt;
  staleReceipt.receiptSHA256 = sha256(Buffer.from(canonicalJSON(stalePreimage)));
  await writeFile(join(stale, "receipt.json"), canonicalJSON(staleReceipt));
  await expect(runCheck(stale)).resolves.toMatchObject({ code: 1, stderr: "margo.npm_cache_lock_mismatch" });

  const missing = await cacheCopy();
  const missingReceipt = JSON.parse(await readFile(join(missing, "receipt.json"), "utf8"));
  const cacache = cacacheForVerifiedNpm();
  const missingInfo = await cacache.get.info(join(missing, "_cacache"), missingReceipt.packages[1].cacheKey);
  await unlink(missingInfo.path);
  const missingResult = await runCheck(missing);
  expect(missingResult.code).toBe(1);
  expect(missingResult.stderr).toContain("margo.npm_cache_package_missing_or_tampered");

  const tampered = await cacheCopy();
  const tamperedReceipt = JSON.parse(await readFile(join(tampered, "receipt.json"), "utf8"));
  const tamperedInfo = await cacache.get.info(join(tampered, "_cacache"), tamperedReceipt.packages[1].cacheKey);
  await writeFile(tamperedInfo.path, Buffer.alloc(tamperedReceipt.packages[1].bytes, 0x58));
  const tamperedResult = await runCheck(tampered);
  expect(tamperedResult.code).toBe(1);
  expect(tamperedResult.stderr).toContain("margo.npm_cache_package_missing_or_tampered");

  const extra = await cacheCopy();
  await cacache.put(join(extra, "_cacache"), "margo:unexpected", Buffer.from("unexpected"));
  await expect(runCheck(extra)).resolves.toMatchObject({ code: 1, stderr: "margo.npm_cache_entry_set_mismatch" });
});
