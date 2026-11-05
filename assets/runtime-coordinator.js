(() => {
  "use strict";
  if (globalThis.margoRuntimeCoordinator) return;
  const jobs = new Map();
  let generation = 0;
  const domReady = document.readyState === "loading"
    ? new Promise(resolve => document.addEventListener("DOMContentLoaded", resolve, {once: true}))
    : Promise.resolve();
  function snapshot() {
    const current = ++generation;
    const ready = domReady.then(async () => {
      document.documentElement.dataset.margoRuntimeStatus = "running";
      const results = await Promise.allSettled([...jobs].map(([name, job]) => {
        if (!job.promise) job.promise = Promise.resolve().then(job.execute);
        return job.promise;
      }));
      const failed = results.find(result => result.status === "rejected");
      if (current === generation) document.documentElement.dataset.margoRuntimeStatus = failed ? "failed" : "ready";
      if (failed) throw failed.reason;
      return results.map(result => result.value);
    });
    // Keep failures available to callers without an unhandled-rejection event.
    ready.catch(() => {});
    globalThis.margoRuntimeReady = ready;
    return ready;
  }
  globalThis.margoRuntimeCoordinator = Object.freeze({
    register(name, execute) {
      if (jobs.has(name)) return;
      jobs.set(name, {execute, promise: null});
      snapshot();
    },
    run() {
      // A new execution retries failures and discovers newly mounted content.
      // Reset before scheduling so simultaneous requests share the same jobs.
      for (const job of jobs.values()) job.promise = null;
      return snapshot();
    },
  });
})();
