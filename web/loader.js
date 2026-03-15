const loaderEl = document.getElementById("loader");

function updateStatus(message, isError) {
  if (!loaderEl) return;
  loaderEl.textContent = message;
  if (isError) loaderEl.classList.add("error");
}

function removeLoader() {
  if (loaderEl && loaderEl.parentNode) {
    loaderEl.parentNode.removeChild(loaderEl);
  }
}

// Polyfill for browsers that lack instantiateStreaming.
if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}

async function start() {
  if (typeof Go === "undefined") {
    updateStatus("wasm_exec.js failed to load", true);
    return;
  }

  const go = new Go();

  updateStatus("Downloading Tetrol…");

  try {
    const result = await WebAssembly.instantiateStreaming(
      fetch("tetrol.wasm"),
      go.importObject,
    );
    removeLoader();
    await go.run(result.instance);
  } catch (err) {
    console.error(err);
    updateStatus("Failed to load: " + err.message, true);
  }
}

start();
