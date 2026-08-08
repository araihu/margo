import { strict as assert } from "node:assert";
import { execFileSync, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { chmod, cp, mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

const browserRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const executablePath = "chrome-mac/Chromium.app/Contents/MacOS/Chromium";

const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");

const tempRoot = await mkdtemp(join(tmpdir(), "margo-install-browser-contract-"));
try {
  const scriptPath = join(tempRoot, "install-browser.sh");
  await cp(join(browserRoot, "install-browser.sh"), scriptPath);
  await chmod(scriptPath, 0o755);

  const fixtureRoot = join(tempRoot, "fixture");
  const executable = join(fixtureRoot, executablePath);
  await mkdir(dirname(executable), { recursive: true });
  await writeFile(executable, "#!/bin/sh\nprintf '%s\\n' fixture-chromium\n");
  await chmod(executable, 0o755);

  const archive = join(tempRoot, ".cache", "downloads", "chromium-fixture.zip");
  await mkdir(dirname(archive), { recursive: true });
  execFileSync("zip", ["-q", "-r", archive, "chrome-mac"], { cwd: fixtureRoot });
  const archiveBytes = await readFile(archive);

  await writeFile(join(tempRoot, "browser-lock.json"), `${JSON.stringify({
    schemaVersion: "margo/browser-lock/v1",
    revision: "contract",
    version: "contract-version",
    networkPolicy: "explicit-provision-only",
    runners: [{
      id: "darwin-arm64",
      archive: "chromium-fixture.zip",
      urls: ["https://example.invalid/chromium-fixture.zip"],
      sha256: sha256(archiveBytes),
      executablePath,
    }],
  }, null, 2)}\n`);

  const receipt = join(tempRoot, ".cache", "playwright", "darwin-arm64", "contract", "browser-receipt.json");
  const result = spawnSync(scriptPath, ["--provision", "--runner", "darwin-arm64", "--receipt", receipt], {
    cwd: tempRoot,
    encoding: "utf8",
  });
  assert.equal(result.status, 0, `installer failed\nstdout=${result.stdout}\nstderr=${result.stderr}`);
  await readFile(join(tempRoot, ".cache", "playwright", "darwin-arm64", "contract", executablePath));
  const installedReceipt = JSON.parse(await readFile(receipt, "utf8"));
  assert.equal(installedReceipt.runner, "darwin-arm64");
  assert.equal(installedReceipt.revision, "contract");
  assert.equal(installedReceipt.executableSHA256, sha256(await readFile(join(tempRoot, ".cache", "playwright", "darwin-arm64", "contract", executablePath))));
} finally {
  await rm(tempRoot, { recursive: true, force: true });
}

process.stdout.write("margo.install_browser_contract_ok\n");
