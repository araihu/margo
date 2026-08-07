import { expect, test } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const runtimePath = path.resolve(here, "../../../assets/runtime/margo-runtime.js");
const documentFingerprint = `01${"00".repeat(31)}`;
const instance = "ri-00000000";
const first = `${instance}:mermaid:00000000:${"a".repeat(64)}`;
const second = `${instance}:mermaid:00000001:${"b".repeat(64)}`;
const layout = `${instance}:deck-layout:00000002:${"c".repeat(64)}`;

function descriptor() {
  return {
    protocol: "margo-runtime/v1",
    documentFingerprint,
    renderInstanceID: instance,
    tasks: [
      { id: first, kind: "mermaid", inputSHA256: "1".repeat(64), dependsOn: [] },
      { id: second, kind: "mermaid", inputSHA256: "2".repeat(64), dependsOn: [] },
      { id: layout, kind: "deck-layout", inputSHA256: "3".repeat(64), dependsOn: [first, second] },
    ],
  };
}

async function loadRuntime(page) {
  await page.goto("about:blank");
  await page.addScriptTag({ path: runtimePath });
  return page.evaluate(() => Object.keys(globalThis.MargoRuntime).sort());
}

test("@runtime-schema Go and Chromium enforce descriptor graphs and transitions", async ({ page }) => {
  const keys = await loadRuntime(page);
  expect(keys).toEqual([
    "canonicalProjection",
    "createInstanceAllocator",
    "createRegistry",
    "protocol",
    "validateDescriptor",
    "validateReport",
  ]);

  const result = await page.evaluate(({ value, firstTask, secondTask, layoutTask }) => {
    const runtime = globalThis.MargoRuntime;
    runtime.validateDescriptor(value);
    const registry = runtime.createRegistry();
    const state = registry.register(value, "exec-a");
    state.start();
    let dependencyCode = "";
    try {
      state.startTask(layoutTask);
    } catch (error) {
      dependencyCode = error.code;
    }
    for (const [id, output] of [[firstTask, "a"], [secondTask, "b"]]) {
      state.startTask(id);
      state.succeedTask(id, output.repeat(64), 100);
    }
    state.startTask(layoutTask);
    state.succeedTask(layoutTask, "c".repeat(64), 102);
    state.ready();
    let terminalCode = "";
    try {
      state.fail("late");
    } catch (error) {
      terminalCode = error.code;
    }
    let duplicateCode = "";
    try {
      registry.register(value, "exec-b");
    } catch (error) {
      duplicateCode = error.code;
    }
    return { dependencyCode, terminalCode, duplicateCode };
  }, { value: descriptor(), firstTask: first, secondTask: second, layoutTask: layout });

  expect(result).toEqual({
    dependencyCode: "runtime.dependency_pending",
    terminalCode: "runtime.transition_invalid",
    duplicateCode: "runtime.instance_duplicate",
  });
});

test("@runtime-schema rejects duplicate, missing, cyclic, and malformed descriptors", async ({ page }) => {
  await loadRuntime(page);
  const codes = await page.evaluate((valid) => {
    const runtime = globalThis.MargoRuntime;
    const variants = [];
    const duplicate = structuredClone(valid);
    duplicate.tasks.push(structuredClone(duplicate.tasks[0]));
    variants.push(duplicate);
    const missing = structuredClone(valid);
    missing.tasks[2].dependsOn = [`${valid.renderInstanceID}:missing`];
    variants.push(missing);
    const cyclic = structuredClone(valid);
    cyclic.tasks[0].dependsOn = [cyclic.tasks[1].id];
    cyclic.tasks[1].dependsOn = [cyclic.tasks[0].id];
    variants.push(cyclic);
    const malformed = structuredClone(valid);
    malformed.renderInstanceID = "ri-INVALID";
    variants.push(malformed);
    return variants.map((value) => {
      try {
        runtime.validateDescriptor(value);
        return "accepted";
      } catch (error) {
        return error.code;
      }
    });
  }, descriptor());
  expect(codes).toEqual([
    "runtime.task_duplicate",
    "runtime.dependency_missing",
    "runtime.dependency_cycle",
    "runtime.instance_invalid",
  ]);
});

test("@runtime-schema canonical projection excludes execution routing state", async ({ page }) => {
  await loadRuntime(page);
  const projection = await page.evaluate(({ value, executionID }) => {
    const report = {
      protocol: value.protocol,
      documentFingerprint: value.documentFingerprint,
      renderInstanceID: value.renderInstanceID,
      executionID,
      status: "ready",
      tasks: value.tasks.map((task, index) => ({
        id: task.id,
        kind: task.kind,
        inputSHA256: task.inputSHA256,
        outputSHA256: String.fromCharCode(97 + index).repeat(64),
        outputBytes: 100 + index,
        status: "succeeded",
        errorCode: "",
      })).reverse(),
      fontChecks: [{ family: "Inter", query: "12px Inter", loaded: true }],
      blockedRequests: [],
      layout: { scrollWidth: 1280, scrollHeight: 720 },
      diagnostic: null,
    };
    globalThis.MargoRuntime.validateReport(value, executionID, report);
    return globalThis.MargoRuntime.canonicalProjection(report);
  }, { value: descriptor(), executionID: "exec-a" });

  const want = `{"blockedRequests":[],"diagnosticCode":"","documentFingerprint":"${documentFingerprint}","fontChecks":[{"family":"Inter","loaded":true,"query":"12px Inter"}],"layout":{"scrollHeight":720,"scrollWidth":1280},"protocol":"margo-runtime/v1","renderInstanceID":"ri-00000000","status":"ready","tasks":[` +
    `{"errorCode":"","id":"${layout}","inputSHA256":"${"3".repeat(64)}","kind":"deck-layout","outputBytes":102,"outputSHA256":"${"c".repeat(64)}","status":"succeeded"},` +
    `{"errorCode":"","id":"${first}","inputSHA256":"${"1".repeat(64)}","kind":"mermaid","outputBytes":100,"outputSHA256":"${"a".repeat(64)}","status":"succeeded"},` +
    `{"errorCode":"","id":"${second}","inputSHA256":"${"2".repeat(64)}","kind":"mermaid","outputBytes":101,"outputSHA256":"${"b".repeat(64)}","status":"succeeded"}]}`;
  expect(projection).toBe(want);
  expect(projection).not.toContain("exec-a");
});

test("@runtime-schema rejects forged reports and nonterminal projections", async ({ page }) => {
  await loadRuntime(page);
  const result = await page.evaluate((value) => {
    const runtime = globalThis.MargoRuntime;
    const allocator = runtime.createInstanceAllocator();
    const allocated = Array.from({ length: 37 }, () => allocator.next());
    const report = {
      protocol: value.protocol,
      documentFingerprint: value.documentFingerprint,
      renderInstanceID: value.renderInstanceID,
      executionID: "exec-a",
      status: "ready",
      tasks: value.tasks.map((task, index) => ({
        id: task.id,
        kind: task.kind,
        inputSHA256: task.inputSHA256,
        outputSHA256: String.fromCharCode(97 + index).repeat(64),
        outputBytes: 100 + index,
        status: "succeeded",
        errorCode: "",
      })),
      fontChecks: [],
      blockedRequests: [],
      layout: { scrollWidth: 1280, scrollHeight: 720 },
      diagnostic: null,
    };
    const forged = structuredClone(report);
    forged.executionID = "exec-b";
    let forgedCode = "";
    try {
      runtime.validateReport(value, "exec-a", forged);
    } catch (error) {
      forgedCode = error.code;
    }
    const nonterminal = structuredClone(report);
    nonterminal.status = "running";
    let projectionCode = "";
    try {
      runtime.canonicalProjection(nonterminal);
    } catch (error) {
      projectionCode = error.code;
    }
    return {
      first: allocated[0],
      thirtyFifth: allocated[35],
      thirtySixth: allocated[36],
      forgedCode,
      projectionCode,
    };
  }, descriptor());
  expect(result).toEqual({
    first: "ri-00000000",
    thirtyFifth: "ri-0000000z",
    thirtySixth: "ri-00000010",
    forgedCode: "runtime.report_forged",
    projectionCode: "runtime.report_malformed",
  });
});
