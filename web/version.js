// Loaded before the WASM payload so the browser can cache-bust when
// the version changes.  Also stamps the document title for testers.
(async function () {
  try {
    const resp = await fetch("version.json", { cache: "no-store" });
    const data = await resp.json();
    const v = data.version || "unknown";
    document.title = document.title + " v" + v;

    // Expose globally so loader.js can read it if needed.
    window.__TETROL_VERSION = v;
  } catch (_) {
    // Non-fatal — game still loads without version info.
  }
})();
