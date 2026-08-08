import { describe, test } from "node:test";
import assert from "node:assert/strict";
import path from "node:path";
import { parseArgs } from "./generate-evidence.mjs";

describe("browser evidence command", () => {
  test("requires absolute HTML and evidence paths", () => {
    const options = parseArgs([
      "--html", "/tmp/margo.html",
      "--evidence", "/tmp/margo-evidence.json",
      "--mode", "dark",
    ]);
    assert.equal(options.html, "/tmp/margo.html");
    assert.equal(options.evidence, "/tmp/margo-evidence.json");
    assert.equal(options.mode, "dark");
    assert.equal(path.isAbsolute(options.root), true);
  });

  test("rejects relative paths and unknown modes", () => {
    assert.throws(() => parseArgs(["--html", "margo.html", "--evidence", "/tmp/evidence.json"]), /margo\.browser_evidence_absolute_required:html/);
    assert.throws(() => parseArgs(["--html", "/tmp/margo.html", "--evidence", "/tmp/evidence.json", "--mode", "sepia"]), /margo\.browser_evidence_mode_invalid:sepia/);
  });
});
