import { describe, test } from "node:test";
import assert from "node:assert/strict";
import { formatText, parseArgs } from "./lint-contrast.mjs";

describe("contrast lint command", () => {
  test("parses the checked runner contract", () => {
    assert.deepEqual(parseArgs([
      "--html", "/tmp/custom.html",
      "--mode", "dark",
      "--format", "text",
      "--output", "/tmp/report.txt",
    ]), {
      html: "/tmp/custom.html",
      mode: "dark",
      format: "text",
      output: "/tmp/report.txt",
    });
  });

  test("rejects relative source paths", () => {
    assert.throws(() => parseArgs(["--html", "custom.html"]), /margo\.contrast_lint_html_absolute_required/);
  });

  test("formats stable human-readable evidence", () => {
    const report = {
      source: { bytes: 3, sha256: "abc" },
      media: "print",
      rules: { normalTextMinimumRatio: 4.5, largeTextMinimumRatio: 3, network: "deny" },
      modes: [{ mode: "light", checked: 2, failures: [] }],
      network: { blocked: [] },
      status: "pass",
    };
    assert.equal(formatText(report), [
      "margo contrast lint: PASS",
      "source.sha256 abc",
      "source.bytes 3",
      "media print",
      "rules.normal 4.5:1",
      "rules.large 3:1",
      "rules.network deny",
      "light checked=2 failures=0",
      "network.blocked 0",
      "",
    ].join("\n"));
  });
});
