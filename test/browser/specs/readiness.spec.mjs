import { expect, test } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");
const runtimePath = path.join(root, "assets/runtime/margo-runtime.js");
const readinessPath = path.join(root, "assets/runtime/readiness.js");
const readinessVectors = JSON.parse(fs.readFileSync(path.join(root, "testdata/runtime/readiness-vectors.json"), "utf8"));

const documentFingerprint = `01${"00".repeat(31)}`;

function descriptor(instance, taskCount = 2) {
  return {
    protocol: "margo-runtime/v1",
    documentFingerprint,
    renderInstanceID: instance,
    tasks: Array.from({ length: taskCount }, (_, index) => ({
      id: `${instance}:mermaid:${index.toString(10).padStart(8, "0")}:${String.fromCharCode(97 + index).repeat(64)}`,
      kind: "mermaid",
      inputSHA256: String(index + 1).repeat(64),
      dependsOn: index === 0 ? [] : [
        `${instance}:mermaid:${(index - 1).toString(10).padStart(8, "0")}:${String.fromCharCode(96 + index).repeat(64)}`,
      ],
    })),
  };
}

async function loadReadiness(page) {
  await page.goto(pathToFileURL(path.join(root, "test/browser/fixtures/probe.html")).href);
  await page.addScriptTag({ path: runtimePath });
  await page.addScriptTag({ path: readinessPath });
  return page.evaluate(() => Object.keys(globalThis.MargoReadiness ?? {}).sort());
}

test("@readiness exposes a terminal collector contract", async ({ page }) => {
  const keys = await loadReadiness(page);
  expect(keys).toEqual(["create"]);
});

test("@readiness one failed placement does not block its sibling", async ({ page }) => {
  await loadReadiness(page);
  const reports = await page.evaluate(async (documentFingerprintValue) => {
    const makeDescriptor = (instance, failed = false) => ({
      protocol: "margo-runtime/v1",
      documentFingerprint: documentFingerprintValue,
      renderInstanceID: instance,
      tasks: [{
        id: `${instance}:mermaid:00000000:${failed ? "b".repeat(64) : "a".repeat(64)}`,
        kind: "mermaid",
        inputSHA256: "1".repeat(64),
        dependsOn: [],
      }],
    });
    const runTask = async (task) => ({
      id: task.id,
      kind: task.kind,
      inputSHA256: task.inputSHA256,
      outputSHA256: "a".repeat(64),
      outputBytes: 12,
      status: "succeeded",
      errorCode: "",
    });
    const valid = globalThis.MargoReadiness.create({
      descriptor: makeDescriptor("ri-0000000a"),
      executionID: "exec-a",
      wrapper: document.body,
      runTask,
      collectLayout: () => ({ scrollWidth: 100, scrollHeight: 200 }),
    });
    const failing = globalThis.MargoReadiness.create({
      descriptor: makeDescriptor("ri-0000000b", true),
      executionID: "exec-b",
      wrapper: document.body,
      runTask: async () => { throw Object.assign(new Error("synthetic task failure"), { code: "runtime.synthetic_failure" }); },
      collectLayout: () => ({ scrollWidth: 100, scrollHeight: 200 }),
    });
    return Promise.all([valid.run(), failing.run()]);
  }, documentFingerprint);
  expect(reports.map((report) => report.status)).toEqual(["ready", "failed"]);
  expect(reports[0].executionID).toBe("exec-a");
  expect(reports[1].diagnostic.code).toBe("runtime.synthetic_failure");
});

test("@readiness rejects forged task evidence before terminal acceptance", async ({ page }) => {
  await loadReadiness(page);
  const result = await page.evaluate((documentFingerprintValue) => {
    const instance = "ri-0000000c";
    const task = {
      id: `${instance}:mermaid:00000000:${"a".repeat(64)}`,
      kind: "mermaid",
      inputSHA256: "1".repeat(64),
      dependsOn: [],
    };
    const collector = globalThis.MargoReadiness.create({
      descriptor: { protocol: "margo-runtime/v1", documentFingerprint: documentFingerprintValue, renderInstanceID: instance, tasks: [task] },
      executionID: "exec-forged",
      wrapper: document.body,
      runTask: async () => ({ ...task, outputSHA256: "f".repeat(64), outputBytes: 1, status: "succeeded", errorCode: "" }),
      collectLayout: () => ({ scrollWidth: 100, scrollHeight: 200 }),
    });
    return collector.run();
  }, documentFingerprint);
  expect(result.status).toBe("failed");
  expect(result.diagnostic.code).toBe("runtime.report_forged");
});

test("@readiness failure vectors remain explicit and terminal", async ({ page }) => {
  await loadReadiness(page);
  const result = await page.evaluate(({ fingerprint, vectors }) => {
    const make = (instance, overrides = {}) => globalThis.MargoReadiness.create({
      descriptor: {
        protocol: "margo-runtime/v1",
        documentFingerprint: fingerprint,
        renderInstanceID: instance,
        tasks: [{ id: `${instance}:mermaid:00000000:${"a".repeat(64)}`, kind: "mermaid", inputSHA256: "1".repeat(64), dependsOn: [] }],
      },
      executionID: `exec-${instance}`,
      wrapper: document.body,
      runTask: async (task) => ({ id: task.id, kind: task.kind, inputSHA256: task.inputSHA256, outputSHA256: "a".repeat(64), outputBytes: 1, status: "succeeded", errorCode: "" }),
      collectLayout: () => ({ scrollWidth: 100, scrollHeight: 200 }),
      ...overrides,
    });
    const missing = make("ri-00000012", { runTask: async () => undefined });
    const timeout = make("ri-00000013", { taskTimeoutMs: 1, runTask: () => new Promise(() => {}) });
    const font = make("ri-00000014", { fontChecks: [{ family: "MissingFont", query: "16px MissingFont" }], fontCheck: () => false });
    const network = make("ri-00000015", { blockedRequests: [{ url: "https://margo.invalid/blocked", resourceType: "script" }] });
    let tick = 0;
    const unstable = make("ri-00000016", { collectLayout: () => ({ scrollWidth: 100 + tick++, scrollHeight: 200 }) });
    return Promise.all([missing.run(), timeout.run(), font.run(), network.run(), unstable.run()]).then((reports) => ({
      schemaVersion: vectors.schemaVersion,
      ids: vectors.cases.map((entry) => entry.id),
      codes: reports.map((report) => report.diagnostic?.code),
      terminal: reports.every((report) => report.status === "failed" && Object.isFrozen(report)),
    }));
  }, { fingerprint: documentFingerprint, vectors: readinessVectors });
  expect(result.schemaVersion).toBe("margo/runtime-readiness-fixtures/v1");
  expect(result.ids).toEqual(readinessVectors.cases.map((entry) => entry.id));
  expect(result.codes).toEqual(readinessVectors.cases.map((entry) => entry.expectedCode));
  expect(result.terminal).toBe(true);
});
