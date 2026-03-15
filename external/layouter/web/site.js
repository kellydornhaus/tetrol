import { DEMOS } from "./demos.js";

const frame = document.getElementById("demo-frame");
const frameLoader = document.getElementById("frame-loader");
const titleEl = document.getElementById("demo-title");
const summaryEl = document.getElementById("demo-summary");
const tagsEl = document.getElementById("demo-tags");
const kindEl = document.getElementById("demo-kind");
const screenNameEl = document.getElementById("demo-screen-name");
const heroCountEl = document.getElementById("demo-count");
const pillIndexEl = document.getElementById("demo-index");
const prevBtn = document.getElementById("prev-demo");
const nextBtn = document.getElementById("next-demo");
const copyLinkBtn = document.getElementById("copy-link");
const codeTabsEl = document.getElementById("code-tabs");
const codeBlockEl = document.getElementById("code-block");
const codeBlockWrapEl = document.getElementById("code-block-wrap");
const codePathEl = document.getElementById("code-path");
const demoViewEl = document.getElementById("demo-view");

let currentIndex = 0;
const codeCache = new Map();
const SOURCE_LANGS = new Set(["xml", "css", "go"]);
const TAB_KIND_DEMO = "demo";
const DEMO_COMPLEXITY_ORDER = [
  "align-self-xml",
  "flowstack-xml",
  "rounded-xml",
  "theme-xml",
  "visibility-fx",
  "positioning",
  "assets-xml",
  "auto-font-xml",
  "viewport-column-xml",
  "xml-demo",
  "templates-xml",
  "scoreboard-xml",
];
const DEMO_ORDER_LOOKUP = new Map(DEMO_COMPLEXITY_ORDER.map((id, idx) => [id, idx]));

function prepareShowcaseDemos(entries) {
  const filtered = (entries || [])
    .map((demo) => {
      const sources = (demo.sources || []).filter((src) => SOURCE_LANGS.has((src.lang || "").toLowerCase()));
      return { ...demo, sources };
    })
    .filter((demo) => demo.sources.length > 0);
  filtered.sort((a, b) => {
    const ai = DEMO_ORDER_LOOKUP.has(a.id) ? DEMO_ORDER_LOOKUP.get(a.id) : DEMO_COMPLEXITY_ORDER.length;
    const bi = DEMO_ORDER_LOOKUP.has(b.id) ? DEMO_ORDER_LOOKUP.get(b.id) : DEMO_COMPLEXITY_ORDER.length;
    if (ai !== bi) {
      return ai - bi;
    }
    return a.title.localeCompare(b.title);
  });
  return filtered;
}

const SHOWCASE_DEMOS = prepareShowcaseDemos(DEMOS);

function showCodeView() {
  if (codeBlockWrapEl) {
    codeBlockWrapEl.classList.remove("tab-hidden");
  }
  if (demoViewEl) {
    demoViewEl.classList.add("tab-hidden");
  }
}

function showDemoView() {
  if (codeBlockWrapEl) {
    codeBlockWrapEl.classList.add("tab-hidden");
  }
  if (demoViewEl) {
    demoViewEl.classList.remove("tab-hidden");
  }
  if (codePathEl) {
    codePathEl.textContent = "Live Demo";
  }
}

function activateTab(entry, button) {
  if (!entry || !codeTabsEl) {
    return;
  }
  Array.from(codeTabsEl.children).forEach((child) => child.classList.remove("active"));
  if (button) {
    button.classList.add("active");
  }
  if (entry.kind === TAB_KIND_DEMO) {
    showDemoView();
  } else {
    loadCode(entry);
  }
}

function initialIndex() {
  const params = new URLSearchParams(window.location.search);
  const slug = params.get("demo");
  if (!slug) {
    return 0;
  }
  const idx = SHOWCASE_DEMOS.findIndex((d) => d.id === slug);
  return idx >= 0 ? idx : 0;
}

function setCurrentIndex(nextIndex, updateUrl = false) {
  const total = SHOWCASE_DEMOS.length;
  if (total === 0) {
    return;
  }
  currentIndex = ((nextIndex % total) + total) % total;
  renderCurrent(updateUrl);
}

function renderCurrent(updateUrl = false) {
  const demo = SHOWCASE_DEMOS[currentIndex];
  if (!demo) {
    return;
  }
  const url = `demo.html?demo=${encodeURIComponent(demo.id)}`;
  if (frame) {
    if (frameLoader) {
      frameLoader.classList.remove("hidden");
      frameLoader.textContent = "Preparing demo…";
    }
    frame.src = url;
  }
  if (titleEl) {
    titleEl.textContent = demo.title;
  }
  if (kindEl) {
    kindEl.textContent = demo.kind || (demo.usesXml ? "XML / CSS" : "Go Layout");
  }
  if (summaryEl) {
    summaryEl.textContent = demo.summary || "";
  }
  if (screenNameEl) {
    screenNameEl.textContent = demo.screen || demo.title;
  }
  const countLabel = `${currentIndex + 1}/${SHOWCASE_DEMOS.length}`;
  if (heroCountEl) {
    heroCountEl.textContent = countLabel;
  }
  if (pillIndexEl) {
    pillIndexEl.textContent = countLabel;
  }
  renderTags(demo.tags || []);
  renderCodeTabs(demo);
  if (updateUrl) {
    syncUrl(demo.id);
  }
}

function renderTags(tags) {
  if (!tagsEl) {
    return;
  }
  tagsEl.innerHTML = "";
  (tags || []).forEach((tag) => {
    const span = document.createElement("span");
    span.className = "tag";
    span.textContent = tag;
    tagsEl.appendChild(span);
  });
}

function renderCodeTabs(demo) {
  if (!codeTabsEl || !codeBlockEl) {
    return;
  }
  codeTabsEl.innerHTML = "";
  const sources = (demo.sources || []).map((src) => ({ ...src, kind: "code" }));
  const entries = [...sources, { label: "Live Demo", kind: TAB_KIND_DEMO }];
  entries.forEach((entry) => {
    const btn = document.createElement("button");
    btn.type = "button";
    const label = entry.kind === TAB_KIND_DEMO ? "Demo" : entry.label || entry.path;
    btn.textContent = label;
    btn.addEventListener("click", () => activateTab(entry, btn));
    codeTabsEl.appendChild(btn);
  });
  const defaultIdx = sources.length > 0 ? 0 : entries.length - 1;
  const defaultBtn = codeTabsEl.children[defaultIdx];
  activateTab(entries[defaultIdx], defaultBtn);
}

async function loadCode(entry) {
  if (!entry || !codeBlockEl) {
    return;
  }
  showCodeView();
  const cacheKey = `${entry.path}#${entry.focus || ""}`;
  if (codeCache.has(cacheKey)) {
    renderCode(entry, codeCache.get(cacheKey));
    return;
  }
  try {
    const res = await fetch(`code/${entry.path}`);
    if (!res.ok) {
      throw new Error(`fetch failed (${res.status})`);
    }
    const text = await res.text();
    const snippet = extractSnippet(text, entry);
    codeCache.set(cacheKey, snippet);
    renderCode(entry, snippet);
  } catch (err) {
    codeBlockEl.textContent = `Unable to load ${entry.path}: ${err.message}`;
    if (codePathEl) {
      codePathEl.textContent = entry.path;
    }
  }
}

function extractSnippet(text, entry) {
  if (!entry.focus) {
    return text.trim();
  }
  const lang = (entry.lang || "").toLowerCase();
  const needle = entry.focus.startsWith("func ") ? entry.focus : entry.focus;
  const idx = text.indexOf(needle);
  if (idx === -1 && lang === "go") {
    const fnIdx = text.indexOf(`func ${entry.focus}`);
    if (fnIdx === -1) {
      return text.trim();
    }
    return sliceGoFunction(text, fnIdx).trim();
  }
  if (idx === -1) {
    return text.trim();
  }
  if (lang === "go") {
    return sliceGoFunction(text, idx).trim();
  }
  return text.slice(idx).trim();
}

function sliceGoFunction(text, startIdx) {
  const tail = text.slice(startIdx);
  const nextIdx = tail.indexOf("\nfunc ");
  if (nextIdx === -1) {
    return tail;
  }
  return tail.slice(0, nextIdx);
}

function renderCode(entry, snippet) {
  if (!codeBlockEl) {
    return;
  }
  codeBlockEl.textContent = snippet;
  if (codePathEl) {
    codePathEl.textContent = entry.path;
  }
}

function syncUrl(id) {
  const url = new URL(window.location.href);
  url.searchParams.set("demo", id);
  window.history.replaceState({}, "", url.toString());
}

function copyLink() {
  const demo = SHOWCASE_DEMOS[currentIndex];
  if (!demo) {
    return;
  }
  const url = new URL(window.location.href);
  url.searchParams.set("demo", demo.id);
  const value = url.toString();
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(value);
  } else {
    const textarea = document.createElement("textarea");
    textarea.value = value;
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand("copy");
    document.body.removeChild(textarea);
  }
  if (copyLinkBtn) {
    copyLinkBtn.textContent = "Copied";
    setTimeout(() => (copyLinkBtn.textContent = "Copy link"), 900);
  }
}

function bindEvents() {
  if (prevBtn) {
    prevBtn.addEventListener("click", () => setCurrentIndex(currentIndex - 1, true));
  }
  if (nextBtn) {
    nextBtn.addEventListener("click", () => setCurrentIndex(currentIndex + 1, true));
  }
  if (copyLinkBtn) {
    copyLinkBtn.addEventListener("click", copyLink);
  }
  if (frame) {
    frame.addEventListener("load", () => {
      if (frameLoader) {
        frameLoader.textContent = "";
        frameLoader.classList.add("hidden");
      }
    });
  }
}

function init() {
  bindEvents();
  if (SHOWCASE_DEMOS.length === 0) {
    if (frameLoader) {
      frameLoader.textContent = "No demos available.";
    }
    if (titleEl) {
      titleEl.textContent = "Coming soon";
    }
    if (summaryEl) {
      summaryEl.textContent = "Try rebuilding with code snippets included.";
    }
    return;
  }
  if (heroCountEl) {
    heroCountEl.textContent = `1/${SHOWCASE_DEMOS.length}`;
  }
  setCurrentIndex(initialIndex(), true);
}

init();
