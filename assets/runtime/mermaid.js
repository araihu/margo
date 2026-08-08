(() => {
  "use strict";

  const INSTANCE_PATTERN = /^ri-[0-9a-z]{8,32}$/;
  const DIGEST_PATTERN = /^[0-9a-f]{64}$/;
  const TASK_KIND = "mermaid";
  const SOURCE_ROOT_DOMAIN = "margo/mermaid-source-root/v1";
  let processTail = Promise.resolve();

  class MermaidQueueError extends Error {
    constructor(code, message) {
      super(message);
      this.name = "MargoMermaidQueueError";
      this.code = code;
    }
  }

  function fail(code, message) {
    throw new MermaidQueueError(code, message);
  }

  function isPlainObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
  }

  function cloneAndFreeze(value, seen = new WeakSet()) {
    if (value === null || typeof value !== "object") return value;
    if (seen.has(value)) return value;
    seen.add(value);
    for (const child of Object.values(value)) cloneAndFreeze(child, seen);
    return Object.freeze(value);
  }

  function copyAndFreeze(value) {
    let copy;
    try {
      copy = structuredClone(value);
    } catch {
      fail("mermaid.configuration_invalid", "base configuration is not cloneable");
    }
    return cloneAndFreeze(copy);
  }

  function decimal8(value) {
    if (!Number.isSafeInteger(value) || value < 0 || value > 99999999) {
      fail("mermaid.block_ordinal_invalid", "block ordinal must be an eight-digit decimal ordinal");
    }
    return value.toString(10).padStart(8, "0");
  }

  function requireInstanceID(value) {
    if (typeof value !== "string" || !INSTANCE_PATTERN.test(value)) {
      fail("mermaid.instance_invalid", "render instance ID is invalid");
    }
    return value;
  }

  function requireSource(value) {
    if (typeof value !== "string" || value.length === 0) {
      fail("mermaid.source_invalid", "Mermaid source must be a non-empty string");
    }
    return value;
  }

  function requireFamily(value) {
    if (typeof value !== "string" || !/^[a-z][a-z0-9-]{0,31}$/.test(value)) {
      fail("mermaid.family_invalid", "Mermaid family is invalid");
    }
    return value;
  }

  function sha256Hex(value) {
    return crypto.subtle.digest("SHA-256", new TextEncoder().encode(value)).then((bytes) => {
      return [...new Uint8Array(bytes)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
    });
  }

  function errorCode(error, fallback = "mermaid.task_failed") {
    if (error && typeof error.code === "string" && /^[a-z0-9_]+(?:\.[a-z0-9_]+)*$/.test(error.code)) return error.code;
    return fallback;
  }

  function targetElement(value) {
    if (value === undefined || value === null) return null;
    if (!(value instanceof Element) || typeof value.replaceChildren !== "function") {
      fail("mermaid.target_invalid", "render target must be a DOM element");
    }
    return value;
  }

  function serializeSVG(svg) {
    if (typeof svg !== "string" || svg === "") fail("mermaid.svg_invalid", "normalized SVG is empty");
    const parsed = new DOMParser().parseFromString(svg, "image/svg+xml");
    if (!parsed.documentElement || parsed.documentElement.localName !== "svg" || parsed.querySelector("parsererror")) {
      fail("mermaid.svg_invalid", "normalized SVG is not an SVG document");
    }
    const serialized = new XMLSerializer().serializeToString(parsed.documentElement);
    if (serialized === "") fail("mermaid.svg_invalid", "serialized SVG is empty");
    return { parsed, serialized };
  }

  function insertSVG(target, parsed) {
    if (!target) return false;
    const ownerDocument = target.ownerDocument ?? document;
    target.replaceChildren(ownerDocument.importNode(parsed.documentElement, true));
    return true;
  }

  function create(options) {
    if (!isPlainObject(options)) fail("mermaid.queue_options_invalid", "queue options must be an object");
    if (!options.mermaid || typeof options.mermaid.initialize !== "function" || typeof options.mermaid.render !== "function") {
      fail("mermaid.engine_invalid", "Mermaid engine must expose initialize and render");
    }
    if (!isPlainObject(options.baseConfig)) fail("mermaid.configuration_invalid", "base configuration must be an object");
    if (typeof options.normalizeSVG !== "function" || typeof options.validateSVG !== "function") {
      fail("mermaid.pipeline_invalid", "normalizer and validator are required");
    }

    const engine = options.mermaid;
    const baseConfig = copyAndFreeze(options.baseConfig);

    async function execute(input) {
      if (!isPlainObject(input)) fail("mermaid.task_invalid", "task must be an object");
      const instanceID = requireInstanceID(input.instanceID ?? input.renderInstanceID);
      const ordinalText = decimal8(input.blockOrdinal);
      const source = requireSource(input.source);
      const family = requireFamily(input.family);
      const target = targetElement(input.target);
      const sourceSHA256 = await sha256Hex(source);
      const taskID = `${instanceID}:${TASK_KIND}:${ordinalText}:${sourceSHA256}`;
      if (input.id !== undefined && input.id !== taskID) fail("runtime.task_identity_mismatch", "task ID does not match source identity");
      if (input.inputSHA256 !== undefined && input.inputSHA256 !== sourceSHA256) fail("runtime.task_identity_mismatch", "task input digest does not match source");

      const sourceRootID = `msrc-${await sha256Hex(`${SOURCE_ROOT_DOMAIN}\n${instanceID}\n${ordinalText}`)}`;
      const failed = (error) => {
        const report = Object.freeze({
          id: taskID,
          kind: TASK_KIND,
          inputSHA256: sourceSHA256,
          outputSHA256: "",
          outputBytes: 0,
          status: "failed",
          errorCode: errorCode(error),
        });
        return { ...report, report, sourceRootID, blockOrdinal: input.blockOrdinal, inserted: false, svg: "" };
      };

      try {
        const config = copyAndFreeze({ ...baseConfig, deterministicIDSeed: sourceRootID });
        engine.initialize(config);
        const rendered = await engine.render(sourceRootID, source);
        if (!rendered || typeof rendered.svg !== "string") fail("mermaid.render_malformed", "Mermaid render returned no SVG");
        const context = {
          sourceRootID,
          renderInstanceID: instanceID,
          blockOrdinal: input.blockOrdinal,
          family,
          profile: options.profile,
          profileFingerprint: options.profileFingerprint,
        };
        const normalized = await options.normalizeSVG(rendered.svg, context);
        const normalizedSVG = typeof normalized === "string" ? normalized : normalized?.svg;
        if (typeof normalizedSVG !== "string") fail("mermaid.normalize_malformed", "normalizer returned no SVG");
        await options.validateSVG(normalizedSVG, context);
        const serialized = serializeSVG(normalizedSVG);
        const outputSHA256 = await sha256Hex(serialized.serialized);
        const outputBytes = new TextEncoder().encode(serialized.serialized).byteLength;
        const inserted = insertSVG(target, serialized.parsed);
        const report = Object.freeze({
          id: taskID,
          kind: TASK_KIND,
          inputSHA256: sourceSHA256,
          outputSHA256,
          outputBytes,
          status: "succeeded",
          errorCode: "",
        });
        return {
          ...report,
          report,
          sourceRootID,
          blockOrdinal: input.blockOrdinal,
          inserted,
          diagramType: typeof rendered.diagramType === "string" ? rendered.diagramType : "",
          svg: serialized.serialized,
        };
      } catch (error) {
        return failed(error);
      }
    }

    function run(input) {
      const job = processTail.then(() => execute(input));
      processTail = job.then(() => undefined, () => undefined);
      return job;
    }

    return Object.freeze({
      run,
      enqueue: run,
    });
  }

  globalThis.MargoMermaidQueue = Object.freeze({ create });
})();
