(() => {
  "use strict";

  const protocol = "margo-runtime/v1";
  const instancePattern = /^ri-[0-9a-z]{8,32}$/;
  const kindPattern = /^[a-z][a-z0-9-]{0,31}$/;
  const digestPattern = /^[0-9a-f]{64}$/;

  class RuntimeError extends Error {
    constructor(code, message) {
      super(message);
      this.name = "MargoRuntimeError";
      this.code = code;
    }
  }

  function fail(code, message) {
    throw new RuntimeError(code, message);
  }

  function object(value, code) {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      fail(code, "expected object");
    }
    return value;
  }

  function exactKeys(value, keys, code) {
    object(value, code);
    const actual = Object.keys(value).sort();
    const expected = [...keys].sort();
    if (actual.length !== expected.length || actual.some((key, index) => key !== expected[index])) {
      fail(code, "object fields do not match the v1 schema");
    }
  }

  function validTaskID(value, instance) {
    const parts = typeof value === "string" ? value.split(":") : [];
    return parts.length === 4 && parts[0] === instance && kindPattern.test(parts[1]) && /^[0-9]{8}$/.test(parts[2]) && digestPattern.test(parts[3]);
  }

  function validateDescriptor(value) {
    exactKeys(value, ["protocol", "documentFingerprint", "renderInstanceID", "tasks"], "runtime.descriptor_malformed");
    if (value.protocol !== protocol) fail("runtime.protocol_invalid", "invalid protocol");
    if (!digestPattern.test(value.documentFingerprint) || /^0+$/.test(value.documentFingerprint)) fail("runtime.document_fingerprint_invalid", "invalid document fingerprint");
    if (!instancePattern.test(value.renderInstanceID)) fail("runtime.instance_invalid", "invalid render instance ID");
    if (!Array.isArray(value.tasks)) fail("runtime.descriptor_malformed", "tasks must be an array");
    const tasks = new Map();
    for (const task of value.tasks) {
      exactKeys(task, ["dependsOn", "id", "inputSHA256", "kind"], "runtime.task_invalid");
      if (!validTaskID(task.id, value.renderInstanceID) || !kindPattern.test(task.kind) || !digestPattern.test(task.inputSHA256) || !Array.isArray(task.dependsOn)) {
        fail("runtime.task_invalid", "invalid runtime task");
      }
      if (tasks.has(task.id)) fail("runtime.task_duplicate", "duplicate runtime task");
      for (let index = 1; index < task.dependsOn.length; index += 1) {
        if (task.dependsOn[index - 1] === task.dependsOn[index]) fail("runtime.dependency_duplicate", "duplicate dependency");
        if (task.dependsOn[index - 1] > task.dependsOn[index]) fail("runtime.dependency_unsorted", "unsorted dependencies");
      }
      tasks.set(task.id, task);
    }
    for (const task of tasks.values()) {
      for (const dependency of task.dependsOn) {
        if (!tasks.has(dependency)) fail("runtime.dependency_missing", "missing dependency");
      }
    }
    const states = new Map();
    function visit(id) {
      if (states.get(id) === "visiting") return true;
      if (states.get(id) === "visited") return false;
      states.set(id, "visiting");
      for (const dependency of tasks.get(id).dependsOn) {
        if (visit(dependency)) return true;
      }
      states.set(id, "visited");
      return false;
    }
    for (const id of [...tasks.keys()].sort()) {
      if (visit(id)) fail("runtime.dependency_cycle", "dependency cycle");
    }
    return true;
  }

  function validDiagnosticCode(value) {
    return typeof value === "string" && /^[a-z0-9_]+(?:\.[a-z0-9_]+)*$/.test(value);
  }

  function compareASCII(left, right) {
    return left < right ? -1 : left > right ? 1 : 0;
  }

  function validateReport(descriptor, executionID, report) {
    validateDescriptor(descriptor);
    exactKeys(report, ["blockedRequests", "diagnostic", "documentFingerprint", "executionID", "fontChecks", "layout", "protocol", "renderInstanceID", "status", "tasks"], "runtime.report_malformed");
    if (!executionID || report.protocol !== descriptor.protocol || report.documentFingerprint !== descriptor.documentFingerprint || report.renderInstanceID !== descriptor.renderInstanceID || report.executionID !== executionID) {
      fail("runtime.report_forged", "report identity mismatch");
    }
    if (!Array.isArray(report.tasks) || !Array.isArray(report.fontChecks) || !Array.isArray(report.blockedRequests) || !["ready", "failed"].includes(report.status)) {
      fail("runtime.report_malformed", "report is not terminal");
    }
    exactKeys(report.layout, ["scrollHeight", "scrollWidth"], "runtime.report_malformed");
    if (!Number.isSafeInteger(report.layout.scrollHeight) || report.layout.scrollHeight < 0 || !Number.isSafeInteger(report.layout.scrollWidth) || report.layout.scrollWidth < 0) {
      fail("runtime.report_malformed", "invalid layout");
    }
    const expected = new Map(descriptor.tasks.map((task) => [task.id, task]));
    const seen = new Set();
    let failedTask = false;
    for (const task of report.tasks) {
      exactKeys(task, ["errorCode", "id", "inputSHA256", "kind", "outputBytes", "outputSHA256", "status"], "runtime.report_malformed");
      if (seen.has(task.id)) fail("runtime.task_duplicate", "duplicate task report");
      seen.add(task.id);
      const expectedTask = expected.get(task.id);
      if (!expectedTask) fail("runtime.task_unknown", "unknown task report");
      if (task.kind !== expectedTask.kind || task.inputSHA256 !== expectedTask.inputSHA256) fail("runtime.report_forged", "task identity mismatch");
      if (!Number.isSafeInteger(task.outputBytes) || task.outputBytes < 0) fail("runtime.report_malformed", "invalid task output bytes");
      if (task.status === "succeeded") {
        if (!digestPattern.test(task.outputSHA256) || task.errorCode !== "") fail("runtime.report_malformed", "invalid succeeded task");
      } else if (task.status === "failed") {
        failedTask = true;
        if (task.outputSHA256 !== "" || task.outputBytes !== 0 || !validDiagnosticCode(task.errorCode)) fail("runtime.report_malformed", "invalid failed task");
      } else {
        fail("runtime.report_malformed", "task report is not terminal");
      }
    }
    if (seen.size !== expected.size) fail("runtime.task_missing", "missing task report");
    const fontKeys = new Set();
    for (const font of report.fontChecks) {
      exactKeys(font, ["family", "loaded", "query"], "runtime.report_malformed");
      const key = `${font.family}\u0000${font.query}`;
      if (!font.family || !font.query || fontKeys.has(key) || typeof font.loaded !== "boolean") fail("runtime.report_malformed", "invalid font check");
      fontKeys.add(key);
      if (report.status === "ready" && !font.loaded) fail("runtime.report_malformed", "failed font in ready report");
    }
    for (const request of report.blockedRequests) {
      exactKeys(request, ["resourceType", "url"], "runtime.report_malformed");
      if (typeof request.resourceType !== "string" || typeof request.url !== "string") fail("runtime.report_malformed", "invalid blocked request");
    }
    if (report.status === "ready") {
      if (failedTask || report.diagnostic !== null || report.blockedRequests.length !== 0) fail("runtime.report_malformed", "failure evidence in ready report");
    } else if (!report.diagnostic || !validDiagnosticCode(report.diagnostic.code) || report.diagnostic.severity !== "error") {
      fail("runtime.report_malformed", "failed report lacks diagnostic");
    }
    return true;
  }

  function createState(descriptor, executionID) {
    const taskStates = new Map(descriptor.tasks.map((task) => [task.id, { descriptor: task, status: "pending", outputSHA256: "", outputBytes: 0, errorCode: "" }]));
    let status = "pending";
    let diagnostic = null;
    function requireRunning() {
      if (status !== "running") fail("runtime.transition_invalid", "runtime is not running");
    }
    return Object.freeze({
      start() {
        if (status !== "pending") fail("runtime.transition_invalid", "runtime can start only from pending");
        status = "running";
      },
      startTask(id) {
        requireRunning();
        const task = taskStates.get(id);
        if (!task) fail("runtime.task_unknown", "unknown task");
        if (task.status !== "pending") fail("runtime.transition_invalid", "task can start only from pending");
        if (task.descriptor.dependsOn.some((dependency) => taskStates.get(dependency).status !== "succeeded")) fail("runtime.dependency_pending", "dependency has not succeeded");
        task.status = "running";
      },
      succeedTask(id, outputSHA256, outputBytes) {
        requireRunning();
        const task = taskStates.get(id);
        if (!task) fail("runtime.task_unknown", "unknown task");
        if (task.status !== "running" || !digestPattern.test(outputSHA256) || !Number.isSafeInteger(outputBytes) || outputBytes < 0) fail("runtime.transition_invalid", "invalid task success transition");
        Object.assign(task, { status: "succeeded", outputSHA256, outputBytes });
      },
      failTask(id, errorCode) {
        requireRunning();
        const task = taskStates.get(id);
        if (!task) fail("runtime.task_unknown", "unknown task");
        if (task.status !== "running" || !validDiagnosticCode(errorCode)) fail("runtime.transition_invalid", "invalid task failure transition");
        Object.assign(task, { status: "failed", errorCode });
        status = "failed";
        diagnostic = { code: errorCode, severity: "error" };
      },
      ready() {
        requireRunning();
        if ([...taskStates.values()].some((task) => task.status !== "succeeded")) fail("runtime.transition_invalid", "tasks are not complete");
        status = "ready";
      },
      fail(errorCode) {
        requireRunning();
        if (!validDiagnosticCode(errorCode)) fail("runtime.transition_invalid", "invalid runtime failure");
        status = "failed";
        diagnostic = { code: errorCode, severity: "error" };
      },
      report() {
        return {
          protocol,
          documentFingerprint: descriptor.documentFingerprint,
          renderInstanceID: descriptor.renderInstanceID,
          executionID,
          status,
          tasks: [...taskStates.values()].map((task) => ({
            id: task.descriptor.id,
            kind: task.descriptor.kind,
            inputSHA256: task.descriptor.inputSHA256,
            outputSHA256: task.outputSHA256,
            outputBytes: task.outputBytes,
            status: task.status,
            errorCode: task.errorCode,
          })),
          fontChecks: [],
          blockedRequests: [],
          layout: { scrollWidth: 0, scrollHeight: 0 },
          diagnostic,
        };
      },
    });
  }

  function createRegistry() {
    const instances = new Set();
    const executions = new Set();
    return Object.freeze({
      register(descriptor, executionID) {
        validateDescriptor(descriptor);
        if (!executionID || typeof executionID !== "string") fail("runtime.execution_invalid", "execution ID is required");
        if (instances.has(descriptor.renderInstanceID)) fail("runtime.instance_duplicate", "render instance is already registered");
        if (executions.has(executionID)) fail("runtime.execution_duplicate", "execution ID is already registered");
        instances.add(descriptor.renderInstanceID);
        executions.add(executionID);
        return createState(structuredClone(descriptor), executionID);
      },
    });
  }

  function createInstanceAllocator() {
    let next = 0;
    return Object.freeze({
      next() {
        const ordinal = next.toString(36).padStart(8, "0");
        next += 1;
        if (ordinal.length > 32) fail("runtime.instance_exhausted", "instance allocator exhausted");
        return `ri-${ordinal}`;
      },
    });
  }

  function canonicalProjection(report) {
    object(report, "runtime.report_malformed");
    if (!Array.isArray(report.tasks)) fail("runtime.report_malformed", "report tasks must be an array");
    const projectionDescriptor = {
      protocol: report.protocol,
      documentFingerprint: report.documentFingerprint,
      renderInstanceID: report.renderInstanceID,
      tasks: report.tasks.map((task) => ({
        id: task.id,
        kind: task.kind,
        inputSHA256: task.inputSHA256,
        dependsOn: [],
      })),
    };
    validateReport(projectionDescriptor, report.executionID, report);
    const tasks = report.tasks.map((task) => ({
      errorCode: task.errorCode,
      id: task.id,
      inputSHA256: task.inputSHA256,
      kind: task.kind,
      outputBytes: task.outputBytes,
      outputSHA256: task.outputSHA256,
      status: task.status,
    })).sort((left, right) => compareASCII(left.kind, right.kind) || compareASCII(left.id, right.id));
    const fontChecks = report.fontChecks.map((font) => ({ family: font.family, loaded: font.loaded, query: font.query }))
      .sort((left, right) => compareASCII(left.family, right.family) || compareASCII(left.query, right.query));
    const blockedRequests = report.blockedRequests.map((request) => ({ resourceType: request.resourceType, url: request.url }))
      .sort((left, right) => compareASCII(left.url, right.url) || compareASCII(left.resourceType, right.resourceType));
    return JSON.stringify({
      blockedRequests,
      diagnosticCode: report.diagnostic?.code ?? "",
      documentFingerprint: report.documentFingerprint,
      fontChecks,
      layout: { scrollHeight: report.layout.scrollHeight, scrollWidth: report.layout.scrollWidth },
      protocol: report.protocol,
      renderInstanceID: report.renderInstanceID,
      status: report.status,
      tasks,
    });
  }

  globalThis.MargoRuntime = Object.freeze({
    canonicalProjection,
    createInstanceAllocator,
    createRegistry,
    protocol,
    validateDescriptor,
    validateReport,
  });
})();
