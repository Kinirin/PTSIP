(function () {
  "use strict";

  const CATALOG_URL = "./i18n/catalog.json";
  const MODES_URL = "./data/modes.json";
  const STORAGE_KEY = "ptsip.pages.locale";

  let catalog = null;
  let modes = [];
  let currentLocale = "en";

  function normalizedCandidates() {
    const candidates = [];
    const params = new URLSearchParams(window.location.search);
    const queryLocale = params.get("lang");
    const storedLocale = window.localStorage.getItem(STORAGE_KEY);

    if (queryLocale) candidates.push(queryLocale);
    if (storedLocale) candidates.push(storedLocale);

    for (const locale of (navigator.languages || [navigator.language || ""])) {
      if (locale) candidates.push(locale);
    }

    candidates.push(catalog.default_locale);

    return candidates.map(function (locale) {
      return String(locale).trim().toLowerCase();
    });
  }

  function resolveLocale() {
    const supported = new Set(catalog.locales.map(function (locale) {
      return locale.code.toLowerCase();
    }));

    for (const candidate of normalizedCandidates()) {
      if (supported.has(candidate)) return candidate;

      const base = candidate.split("-")[0];
      if (supported.has(base)) return base;

      const alias = catalog.aliases[candidate];
      if (alias && supported.has(alias.toLowerCase())) return alias.toLowerCase();
    }

    return catalog.default_locale;
  }

  function message(key, locale) {
    const entry = catalog.messages[key];
    if (!entry) return key;

    return entry[locale] || entry[catalog.default_locale] || key;
  }

  function localizedRecord(record, locale) {
    const labels = record.labels || {};
    return labels[locale] || labels[catalog.default_locale] || {};
  }

  function applyTranslations(locale) {
    document.documentElement.lang = locale;
    document.title = message("meta.title", locale);

    document.querySelectorAll("[data-i18n]").forEach(function (node) {
      node.textContent = message(node.dataset.i18n, locale);
    });

    document.querySelectorAll("[data-i18n-aria]").forEach(function (node) {
      node.setAttribute("aria-label", message(node.dataset.i18nAria, locale));
    });
  }

  function populateLanguageSelector() {
    const selector = document.getElementById("language-selector");
    selector.textContent = "";

    for (const locale of catalog.locales) {
      const option = document.createElement("option");
      option.value = locale.code;
      option.textContent = locale.label;
      selector.appendChild(option);
    }

    selector.value = currentLocale;

    selector.addEventListener("change", function () {
      setLocale(selector.value, true);
    });
  }

  function setLocale(locale, persist) {
    const supported = catalog.locales.some(function (item) {
      return item.code === locale;
    });

    currentLocale = supported ? locale : catalog.default_locale;

    if (persist) {
      window.localStorage.setItem(STORAGE_KEY, currentLocale);
      const url = new URL(window.location.href);
      url.searchParams.set("lang", currentLocale);
      window.history.replaceState({}, "", url);
    }

    const selector = document.getElementById("language-selector");
    if (selector) selector.value = currentLocale;

    applyTranslations(currentLocale);
    renderModes();
  }

  function createModeCard(mode) {
    const localized = localizedRecord(mode, currentLocale);
    const article = document.createElement("article");
    article.className = "mode-card";

    const header = document.createElement("div");
    header.className = "mode-card-header";

    const title = document.createElement("h3");
    title.textContent = localized.name || mode.id;

    const status = document.createElement("span");
    status.className = "mode-status";
    status.textContent = mode.status;

    header.append(title, status);

    const summary = document.createElement("p");
    summary.textContent = localized.summary || "";

    const meta = document.createElement("div");
    meta.className = "mode-meta";

    if (mode.compatibility) {
      const compatibility = document.createElement("span");
      compatibility.textContent = "PTSIP " + mode.compatibility;
      meta.appendChild(compatibility);
    }

    for (const tag of (mode.tags || [])) {
      const badge = document.createElement("span");
      badge.textContent = tag;
      meta.appendChild(badge);
    }

    article.append(header, summary, meta);

    if (mode.source_url) {
      const link = document.createElement("a");
      link.className = "text-link";
      link.href = mode.source_url;
      link.textContent = message("modes.source", currentLocale);
      link.rel = "noopener noreferrer";
      article.appendChild(link);
    }

    return article;
  }

  function renderModes() {
    const list = document.getElementById("mode-list");
    if (!list || !catalog) return;

    list.textContent = "";

    if (!modes.length) {
      const empty = document.createElement("div");
      empty.className = "empty-state";
      empty.textContent = message("modes.empty", currentLocale);
      list.appendChild(empty);
      return;
    }

    for (const mode of modes) {
      list.appendChild(createModeCard(mode));
    }
  }

  async function loadJson(url) {
    const response = await fetch(url, { cache: "no-cache" });
    if (!response.ok) {
      throw new Error("Failed to load " + url + ": " + response.status);
    }
    return response.json();
  }

  async function start() {
    try {
      const loaded = await Promise.all([
        loadJson(CATALOG_URL),
        loadJson(MODES_URL)
      ]);

      catalog = loaded[0];
      modes = Array.isArray(loaded[1].modes) ? loaded[1].modes : [];
      currentLocale = resolveLocale();

      populateLanguageSelector();
      setLocale(currentLocale, false);
    } catch (error) {
      console.error(error);
      document.documentElement.lang = "en";
    }
  }

  start();
}());
