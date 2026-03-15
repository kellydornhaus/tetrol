const loaderEl = document.getElementById("loader");

function updateStatus(message, isError = false) {
  if (!loaderEl) {
    return;
  }
  loaderEl.textContent = message;
  if (isError) {
    loaderEl.classList.add("error");
  }
}

function removeLoader() {
  if (loaderEl && loaderEl.parentNode) {
    loaderEl.parentNode.removeChild(loaderEl);
  }
}

// Polyfill instantiateStreaming for browsers that lack it.
if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}

function parseBool(value, defaultValue) {
  if (value == null) {
    return defaultValue;
  }
  const normalized = value.toLowerCase();
  if (["1", "true", "yes", "on"].includes(normalized)) {
    return true;
  }
  if (["0", "false", "no", "off"].includes(normalized)) {
    return false;
  }
  return defaultValue;
}

function defaultFlags() {
  const mode = (document.body?.dataset?.mode || "").toLowerCase();
  if (mode === "full") {
    return { embed: false, showFPS: true };
  }
  if (mode === "embed") {
    return { embed: true, showFPS: false };
  }
  const path = (window.location.pathname || "").toLowerCase();
  if (path.includes("main")) {
    return { embed: false, showFPS: true };
  }
  return { embed: true, showFPS: false };
}

function buildArgsFromQuery() {
  const params = new URLSearchParams(window.location.search);
  const defaults = defaultFlags();
  const embed = parseBool(params.get("embed"), defaults.embed);
  let showFPS = parseBool(params.get("fps"), defaults.showFPS);
  if (params.has("no-fps")) {
    showFPS = false;
  }
  const demo = params.get("demo") || params.get("screen") || "";
  const args = [];
  if (embed) {
    args.push("-embed");
  }
  if (!showFPS) {
    args.push("-no-fps");
  }
  if (demo) {
    args.push(`-demo=${demo}`);
  }
  return args;
}

async function start() {
  if (typeof Go === "undefined") {
    updateStatus("wasm_exec.js failed to load", true);
    return;
  }

  const go = new Go();
  go.argv = ["layouter.wasm", ...buildArgsFromQuery()];

  updateStatus("Downloading Layouter…");

  try {
    const result = await WebAssembly.instantiateStreaming(fetch("layouter.wasm"), go.importObject);
    removeLoader();
    await go.run(result.instance);
  } catch (err) {
    console.error(err);
    updateStatus(`Failed to load: ${err.message}`, true);
  }
}

start();
