import { cp, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";

const browserRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const keyring = join(browserRoot, ".cache", "node-keyring", "nodejs-release-keys.kbx");
const manifest = join(browserRoot, ".cache", "downloads", "node", "v26.5.0", "SHASUMS256.txt");
const signature = join(browserRoot, ".cache", "downloads", "node", "v26.5.0", "SHASUMS256.txt.sig");
const fingerprint = "C82FA3AE1CBEDC6BE46B9360C43CEC45C17AB93C";
const scratchRoots = [];

test.afterAll(async () => {
  await Promise.all(scratchRoots.map((path) => rm(path, { recursive: true, force: true })));
});

function gpgv(home, ring, sig) {
  return new Promise((resolvePromise, reject) => {
    const child = spawn("gpgv", ["--homedir", home, "--keyring", ring, "--status-fd", "1", sig, manifest], { stdio: "pipe", shell: false });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => { stdout += chunk; });
    child.stderr.on("data", (chunk) => { stderr += chunk; });
    child.once("error", reject);
    child.once("exit", (code) => resolvePromise({ code, stdout, stderr }));
  });
}

test("@margo-harness only the locked Node signer and valid signature proceed", async () => {
  const trustedHome = dirname(keyring);
  const valid = await gpgv(trustedHome, keyring, signature);
  expect(valid.code).toBe(0);
  expect(valid.stdout).toContain(`[GNUPG:] VALIDSIG ${fingerprint} `);
  expect(valid.stdout).toContain("[GNUPG:] GOODSIG");

  const wrongHome = await mkdtemp(join(tmpdir(), "margo-wrong-node-key-"));
  scratchRoots.push(wrongHome);
  const wrongKeyring = join(wrongHome, "wrong-keyring.kbx");
  await writeFile(wrongKeyring, Buffer.alloc(0));
  expect((await gpgv(wrongHome, wrongKeyring, signature)).code).not.toBe(0);

  const badHome = await mkdtemp(join(tmpdir(), "margo-bad-node-signature-"));
  scratchRoots.push(badHome);
  const badSignature = join(badHome, "SHASUMS256.txt.sig");
  const badBytes = Buffer.from(await readFile(signature));
  badBytes[badBytes.length - 1] ^= 0xff;
  await writeFile(badSignature, badBytes);
  const copiedKeyring = join(badHome, "nodejs-release-keys.kbx");
  await cp(keyring, copiedKeyring);
  expect((await gpgv(badHome, copiedKeyring, badSignature)).code).not.toBe(0);
});
