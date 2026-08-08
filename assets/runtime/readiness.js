(() => {
  "use strict";

  const PROTOCOL = "margo-runtime/v1";
  const INSTANCE_PATTERN = /^ri-[0-9a-z]{8,32}$/;
  const DIAGNOSTIC_PATTERN = /^[a-z0-9_]+(?:\.[a-z0-9_]+)*$/;
  const DEFAULT_TASK_TIMEOUT_MS = 5000;
  const DEFAULT_FONT_TIMEOUT_MS = 5000;
  const DEFAULT_DOM_TIMEOUT_MS = 5000;
  const MAX_LAYOUT_FRAMES = 8;
  const QUANTIZATION = 64;
  const registeredInstances = new Set();
  const registeredExecutions = new Set();

  class ReadinessError extends Error {
    constructor(code, message) {
      super(message);
      this.name = "MargoReadinessError";
      this.code = code;
    }
  }

  function fail(code, message) {
    throw new ReadinessError(code, message);
  }

  function isObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
  }

  function clone(value) {
    try {
      return structuredClone(value);
    } catch {
      fail("runtime.readiness_clone_failed", "readiness input is not cloneable");
    }
  }

  function freeze(value, seen = new WeakSet()) {
    if (value === null || typeof value !== "object" || seen.has(value)) return value;
    seen.add(value);
    for (const child of Object.values(value)) freeze(child, seen);
    return Object.freeze(value);
  }

  function codeOf(error, fallback) {
    const code = error && typeof error.code === "string" ? error.code : "";
    return DIAGNOSTIC_PATTERN.test(code) ? code : fallback;
  }

  function compareText(left, right) {
    return left < right ? -1 : left > right ? 1 : 0;
  }

  function requireRuntime(value) {
    const runtime = value ?? globalThis.MargoRuntime;
    if (!runtime || runtime.protocol !== PROTOCOL || typeof runtime.validateDescriptor !== "function" || typeof runtime.validateReport !== "function") {
      fail("runtime.readiness_protocol_missing", "margo-runtime/v1 validation API is required");
    }
    return runtime;
  }

  function requireInstance(value) {
    if (typeof value !== "string" || !INSTANCE_PATTERN.test(value)) fail("runtime.instance_invalid", "render instance ID is invalid");
    return value;
  }

  function requireExecution(value) {
    if (typeof value !== "string" || value.length === 0 || value.length > 128) fail("runtime.execution_invalid", "execution ID is invalid");
    return value;
  }

  function requireWrapper(value) {
    const wrapper = value ?? document.body ?? document.documentElement;
    if (!(wrapper instanceof Element)) fail("runtime.wrapper_invalid", "readiness wrapper must be a DOM element");
    return wrapper;
  }

  function exactKeys(value, expected) {
    if (!isObject(value)) fail("runtime.report_forged", "runtime task report must be an object");
    const actual = Object.keys(value).sort();
    const wanted = [...expected].sort();
    if (actual.length !== wanted.length || actual.some((key, index) => key !== wanted[index])) {
      fail("runtime.report_forged", "runtime task report fields do not match the v1 schema");
    }
  }

  function withTimeout(promise, timeout, code) {
    if (!Number.isFinite(timeout) || timeout <= 0) return Promise.resolve(promise);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new ReadinessError(code, "readiness dependency timed out")), timeout);
      Promise.resolve(promise).then(
        (value) => { clearTimeout(timer); resolve(value); },
        (error) => { clearTimeout(timer); reject(error); },
      );
    });
  }

  function waitForDOMContentLoaded(timeout) {
    if (document.readyState !== "loading") return Promise.resolve();
    return new Promise((resolve, reject) => {
      let timer;
      const onReady = () => {
        if (timer) clearTimeout(timer);
        resolve();
      };
      document.addEventListener("DOMContentLoaded", onReady, { once: true });
      if (Number.isFinite(timeout) && timeout > 0) {
        timer = setTimeout(() => {
          document.removeEventListener("DOMContentLoaded", onReady);
          reject(new ReadinessError("runtime.domcontentloaded_timeout", "DOMContentLoaded timed out"));
        }, timeout);
      }
    });
  }

  function quantize(value) {
    if (typeof value !== "number") return value;
    if (!Number.isFinite(value)) fail("runtime.layout_invalid", "layout metric must be finite");
    const result = Math.round(value * QUANTIZATION) / QUANTIZATION;
    return Object.is(result, -0) ? 0 : result;
  }

  function quantizeMetrics(value, seen = new WeakSet()) {
    if (typeof value === "number") return quantize(value);
    if (value === null || typeof value !== "object") return value;
    if (seen.has(value)) fail("runtime.layout_invalid", "layout metrics must not be cyclic");
    seen.add(value);
    if (Array.isArray(value)) return value.map((entry) => quantizeMetrics(entry, seen));
    const result = {};
    for (const key of Object.keys(value).sort()) result[key] = quantizeMetrics(value[key], seen);
    return result;
  }

  function canonicalMetrics(value) {
    return JSON.stringify(quantizeMetrics(value));
  }

  function requireLayoutMetrics(value) {
    if (!isObject(value)) fail("runtime.layout_invalid", "layout metrics must be an object");
    const width = value.scrollWidth;
    const height = value.scrollHeight;
    if (typeof width !== "number" || !Number.isFinite(width) || width < 0 || typeof height !== "number" || !Number.isFinite(height) || height < 0) {
      fail("runtime.layout_invalid", "layout scroll metrics must be finite and non-negative");
    }
    return value;
  }

  function reportLayout(metrics) {
    return {
      scrollWidth: Math.max(0, Math.round(metrics.scrollWidth)),
      scrollHeight: Math.max(0, Math.round(metrics.scrollHeight)),
    };
  }

  function normalizeFonts(value) {
    if (value === undefined) return [];
    if (!Array.isArray(value)) fail("runtime.fonts_invalid", "font checks must be an array");
    const seen = new Set();
    const result = value.map((font) => {
      if (!isObject(font) || typeof font.family !== "string" || font.family === "" || typeof font.query !== "string" || font.query === "") {
        fail("runtime.fonts_invalid", "font check is malformed");
      }
      const key = `${font.family}\u0000${font.query}`;
      if (seen.has(key)) fail("runtime.font_duplicate", "font check is duplicated");
      seen.add(key);
      return { family: font.family, query: font.query };
    });
    result.sort((left, right) => compareText(left.family, right.family) || compareText(left.query, right.query));
    return result;
  }

  function normalizeBlocked(value) {
    if (!Array.isArray(value)) fail("runtime.network_observation_invalid", "blocked request evidence must be an array");
    const result = value.map((request) => {
      if (!isObject(request) || typeof request.url !== "string" || request.url === "" || typeof request.resourceType !== "string" || request.resourceType === "") {
        fail("runtime.network_observation_invalid", "blocked request evidence is malformed");
      }
      return { url: request.url, resourceType: request.resourceType };
    });
    result.sort((left, right) => compareText(left.url, right.url) || compareText(left.resourceType, right.resourceType));
    return result;
  }

  function normalizeTaskReport(task, value) {
    const candidate = value && value.report !== undefined ? value.report : value;
    exactKeys(candidate, ["errorCode", "id", "inputSHA256", "kind", "outputBytes", "outputSHA256", "status"]);
    if (candidate.id !== task.id || candidate.kind !== task.kind || candidate.inputSHA256 !== task.inputSHA256) {
      fail("runtime.report_forged", "runtime task identity does not match the descriptor");
    }
    if (candidate.status === "succeeded") {
      if (typeof candidate.outputSHA256 !== "string" || !/^[0-9a-f]{64}$/.test(candidate.outputSHA256) || candidate.errorCode !== "" || !Number.isSafeInteger(candidate.outputBytes) || candidate.outputBytes < 0) {
        fail("runtime.report_forged", "runtime task success evidence is malformed");
      }
    } else if (candidate.status === "failed") {
      if (candidate.outputSHA256 !== "" || candidate.outputBytes !== 0 || typeof candidate.errorCode !== "string" || !DIAGNOSTIC_PATTERN.test(candidate.errorCode)) {
        fail("runtime.report_forged", "runtime task failure evidence is malformed");
      }
    } else {
      fail("runtime.report_forged", "runtime task report is not terminal");
    }
    return {
      id: candidate.id,
      kind: candidate.kind,
      inputSHA256: candidate.inputSHA256,
      outputSHA256: candidate.outputSHA256,
      outputBytes: candidate.outputBytes,
      status: candidate.status,
      errorCode: candidate.errorCode,
    };
  }

  function failedTask(task, errorCode) {
    return {
      id: task.id,
      kind: task.kind,
      inputSHA256: task.inputSHA256,
      outputSHA256: "",
      outputBytes: 0,
      status: "failed",
      errorCode: DIAGNOSTIC_PATTERN.test(errorCode) ? errorCode : "runtime.task_failed",
    };
  }

  function taskOrder(tasks) {
    const byID = new Map(tasks.map((task) => [task.id, task]));
    const visited = new Set();
    const order = [];
    function visit(task) {
      if (visited.has(task.id)) return;
      visited.add(task.id);
      for (const dependency of [...task.dependsOn].sort(compareText)) visit(byID.get(dependency));
      order.push(task);
    }
    for (const task of [...tasks].sort((left, right) => compareText(left.id, right.id))) visit(task);
    return order;
  }

  function makeReport(descriptor, executionID, tasks, fontChecks, blockedRequests, layout, status, diagnostic) {
    return freeze({
      protocol: PROTOCOL,
      documentFingerprint: descriptor.documentFingerprint,
      renderInstanceID: descriptor.renderInstanceID,
      executionID,
      status,
      tasks,
      fontChecks,
      blockedRequests,
      layout,
      diagnostic: diagnostic ? { code: diagnostic, severity: "error" } : null,
    });
  }

  function create(options) {
    if (!isObject(options)) fail("runtime.readiness_options_invalid", "readiness options must be an object");
    const runtime = requireRuntime(options.runtime);
    const descriptor = clone(options.descriptor);
    runtime.validateDescriptor(descriptor);
    const instance = requireInstance(descriptor.renderInstanceID);
    const executionID = requireExecution(options.executionID);
    if (registeredInstances.has(instance)) fail("runtime.instance_duplicate", "render instance is already registered");
    if (registeredExecutions.has(executionID)) fail("runtime.execution_duplicate", "execution ID is already registered");
    if (typeof options.runTask !== "function") fail("runtime.task_runner_missing", "readiness task runner is required");
    const wrapper = requireWrapper(options.wrapper);
    const fonts = normalizeFonts(options.fontChecks);
    const fontCheck = options.fontCheck ?? ((query) => document.fonts?.check(query));
    if (typeof fontCheck !== "function") fail("runtime.font_checker_missing", "font checker is required");
    const collectLayout = options.collectLayout ?? (() => ({ scrollWidth: wrapper.scrollWidth, scrollHeight: wrapper.scrollHeight }));
    if (typeof collectLayout !== "function") fail("runtime.layout_collector_missing", "layout collector is required");
    const collectBlocked = options.blockedRequests ?? options.collectBlockedRequests ?? (() => []);
    if (typeof collectBlocked !== "function" && !Array.isArray(collectBlocked)) fail("runtime.network_observer_missing", "blocked request collector is required");
    const requestFrame = options.requestFrame ?? ((callback) => requestAnimationFrame(callback));
    if (typeof requestFrame !== "function") fail("runtime.frame_scheduler_missing", "frame scheduler is required");
    const taskTimeout = options.taskTimeoutMs ?? DEFAULT_TASK_TIMEOUT_MS;
    const fontTimeout = options.fontTimeoutMs ?? DEFAULT_FONT_TIMEOUT_MS;
    const domTimeout = options.domContentLoadedTimeoutMs ?? DEFAULT_DOM_TIMEOUT_MS;
    registeredInstances.add(instance);
    registeredExecutions.add(executionID);

    let runPromise;
    async function execute() {
      const taskReports = new Map();
      let fontChecks = [];
      let blockedRequests = [];
      let layout = { scrollWidth: 0, scrollHeight: 0 };
      let diagnostic = "";
      try {
        await waitForDOMContentLoaded(domTimeout);
        for (const task of taskOrder(descriptor.tasks)) {
          try {
            const value = await withTimeout(options.runTask(clone(task), { descriptor: clone(descriptor), executionID }), taskTimeout, "runtime.task_timeout");
            const report = normalizeTaskReport(task, value);
            taskReports.set(task.id, report);
            if (report.status === "failed") {
              diagnostic = report.errorCode;
              break;
            }
          } catch (error) {
            diagnostic = codeOf(error, "runtime.task_failed");
            taskReports.set(task.id, failedTask(task, diagnostic));
            break;
          }
        }
        if (diagnostic !== "") {
          for (const task of descriptor.tasks) if (!taskReports.has(task.id)) taskReports.set(task.id, failedTask(task, "runtime.dependency_failed"));
        } else {
          if (document.fonts && document.fonts.ready) {
            await withTimeout(document.fonts.ready, fontTimeout, "runtime.font_timeout");
          } else if (fonts.length > 0) {
            fail("runtime.font_api_missing", "document.fonts is unavailable");
          }
          fontChecks = fonts.map((font) => {
            let loaded = false;
            try { loaded = Boolean(fontCheck(font.query, font)); } catch { loaded = false; }
            return { ...font, loaded };
          });
          if (fontChecks.some((font) => !font.loaded)) fail("runtime.font_unavailable", "a declared font did not load");

          let previous = "";
          let lastMetrics = null;
          let stable = false;
          for (let frame = 0; frame < MAX_LAYOUT_FRAMES; frame += 1) {
            await new Promise((resolve) => requestFrame(resolve));
            const metrics = requireLayoutMetrics(await collectLayout());
            const quantized = quantizeMetrics(metrics);
            const canonical = canonicalMetrics(quantized);
            lastMetrics = quantized;
            if (canonical === previous) { stable = true; break; }
            previous = canonical;
          }
          if (!stable || lastMetrics === null) fail("runtime.layout_unstable", "layout did not stabilize within eight frames");
          layout = reportLayout(lastMetrics);
          blockedRequests = normalizeBlocked(typeof collectBlocked === "function" ? await collectBlocked() : collectBlocked);
          if (blockedRequests.length > 0) fail("runtime.network_blocked", "attributed blocked requests remain");
        }
      } catch (error) {
        diagnostic = codeOf(error, "runtime.readiness_failed");
      }

      const orderedTasks = descriptor.tasks.map((task) => taskReports.get(task.id) ?? failedTask(task, diagnostic || "runtime.task_aborted"));
      const report = makeReport(descriptor, executionID, orderedTasks, fontChecks, blockedRequests, layout, diagnostic === "" ? "ready" : "failed", diagnostic);
      try {
        runtime.validateReport(descriptor, executionID, report);
      } catch (error) {
        const code = codeOf(error, "runtime.report_malformed");
        return makeReport(descriptor, executionID, descriptor.tasks.map((task) => failedTask(task, code)), fontChecks, blockedRequests, layout, "failed", code);
      }
      return report;
    }

    return Object.freeze({
      run() {
        if (!runPromise) runPromise = execute();
        return runPromise;
      },
    });
  }

  globalThis.MargoReadiness = Object.freeze({ create });
})();
