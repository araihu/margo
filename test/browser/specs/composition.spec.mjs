import { expect, test } from "@playwright/test";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "../../..");
const runtimePath = path.join(root, "assets/runtime/margo-runtime.js");
const readinessPath = path.join(root, "assets/runtime/readiness.js");
const documentFingerprint = `02${"00".repeat(31)}`;

async function loadReadiness(page) {
  await page.goto(pathToFileURL(path.join(root, "test/browser/fixtures/probe.html")).href);
  await page.addScriptTag({ path: runtimePath });
  await page.addScriptTag({ path: readinessPath });
}

test("@composition keeps two placements and execution IDs independent", async ({ page }) => {
  await loadReadiness(page);
  const result = await page.evaluate((fingerprint) => {
    const make = (instance, executionID) => globalThis.MargoReadiness.create({
      descriptor: { protocol: "margo-runtime/v1", documentFingerprint: fingerprint, renderInstanceID: instance, tasks: [] },
      executionID,
      wrapper: document.body,
      runTask: async () => { throw new Error("must not run"); },
      collectLayout: () => ({ scrollWidth: 1, scrollHeight: 2 }),
    });
    const first = make("ri-0000000d", "exec-d");
    const second = make("ri-0000000e", "exec-e");
    return Promise.all([first.run(), second.run()]);
  }, documentFingerprint);
  expect(result.map((report) => [report.status, report.executionID])).toEqual([
    ["ready", "exec-d"],
    ["ready", "exec-e"],
  ]);
});

test("@composition detects duplicate placement and execution identities", async ({ page }) => {
  await loadReadiness(page);
  const codes = await page.evaluate((fingerprint) => {
    const options = {
      descriptor: { protocol: "margo-runtime/v1", documentFingerprint: fingerprint, renderInstanceID: "ri-0000000f", tasks: [] },
      executionID: "exec-f",
      wrapper: document.body,
      runTask: async () => ({ id: "unused", kind: "unused", inputSHA256: "1".repeat(64), outputSHA256: "a".repeat(64), outputBytes: 1, status: "succeeded", errorCode: "" }),
    };
    const first = globalThis.MargoReadiness.create(options);
    let duplicateInstance = "";
    let duplicateExecution = "";
    try {
      globalThis.MargoReadiness.create({ ...options, executionID: "exec-other" });
    } catch (error) {
      duplicateInstance = error.code;
    }
    try {
      globalThis.MargoReadiness.create({
        ...options,
        descriptor: { ...options.descriptor, renderInstanceID: "ri-00000010" },
      });
    } catch (error) {
      duplicateExecution = error.code;
    }
    return { duplicateInstance, duplicateExecution, first: Boolean(first) };
  }, documentFingerprint);
  expect(codes).toEqual({ duplicateInstance: "runtime.instance_duplicate", duplicateExecution: "runtime.execution_duplicate", first: true });
});

test("@composition stabilizes quantized layout within eight frames", async ({ page }) => {
  await loadReadiness(page);
  const result = await page.evaluate((fingerprint) => {
    let frame = 0;
    const collector = globalThis.MargoReadiness.create({
      descriptor: { protocol: "margo-runtime/v1", documentFingerprint: fingerprint, renderInstanceID: "ri-00000011", tasks: [] },
      executionID: "exec-layout",
      wrapper: document.body,
      runTask: async () => { throw new Error("must not run"); },
      collectLayout: () => ({ scrollWidth: 100.015625 + (frame === 0 ? 0 : 0), scrollHeight: 200.015625, cards: [{ x: 1.015625, y: 2.5 }] }),
      requestFrame: (callback) => requestAnimationFrame(() => { frame += 1; callback(); }),
    });
    return collector.run();
  }, documentFingerprint);
  expect(result.status).toBe("ready");
  expect(result.layout).toEqual({ scrollWidth: 100, scrollHeight: 200 });
});

test("@composition executes dependency order and caps unstable layout at eight frames", async ({ page }) => {
  await loadReadiness(page);
  const result = await page.evaluate((fingerprint) => {
    const instance = "ri-00000017";
    const first = `${instance}:mermaid:00000000:${"a".repeat(64)}`;
    const second = `${instance}:mermaid:00000001:${"b".repeat(64)}`;
    const order = [];
    let frames = 0;
    const collector = globalThis.MargoReadiness.create({
      descriptor: {
        protocol: "margo-runtime/v1",
        documentFingerprint: fingerprint,
        renderInstanceID: instance,
        tasks: [
          { id: second, kind: "mermaid", inputSHA256: "2".repeat(64), dependsOn: [first] },
          { id: first, kind: "mermaid", inputSHA256: "1".repeat(64), dependsOn: [] },
        ],
      },
      executionID: "exec-order",
      wrapper: document.body,
      runTask: async (task) => {
        order.push(task.id);
        return { id: task.id, kind: task.kind, inputSHA256: task.inputSHA256, outputSHA256: "a".repeat(64), outputBytes: 1, status: "succeeded", errorCode: "" };
      },
      collectLayout: () => ({ scrollWidth: 100 + frames, scrollHeight: 200 }),
      requestFrame: (callback) => requestAnimationFrame(() => { frames += 1; callback(); }),
    });
    return collector.run().then((report) => ({ report, order, frames }));
  }, documentFingerprint);
  expect(result.order).toEqual([
    "ri-00000017:mermaid:00000000:" + "a".repeat(64),
    "ri-00000017:mermaid:00000001:" + "b".repeat(64),
  ]);
  expect(result.frames).toBe(8);
  expect(result.report.status).toBe("failed");
  expect(result.report.diagnostic.code).toBe("runtime.layout_unstable");
});
