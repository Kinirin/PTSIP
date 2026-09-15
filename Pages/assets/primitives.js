import { supabaseClientState } from "./supabase-client.js";

(function () {
  "use strict";

  const I18N_URL = "./i18n/catalog.json";
  const DATA_URL = "./data/primitive-catalog.json";
  const STORAGE_KEY = "ptsip.pages.locale";

  const GROUPS = [
    {
      key: "official_core",
      listId: "official-core-list",
      countId: "official-core-count",
      emptyKey: "primitive.groups.official.empty"
    },
    {
      key: "community",
      listId: "community-list",
      countId: "community-count",
      emptyKey: "primitive.groups.community.empty"
    },
    {
      key: "core_candidates",
      listId: "candidate-list",
      countId: "candidate-count",
      emptyKey: "primitive.groups.candidates.empty"
    }
  ];

  let catalog = null;
  let data = null;
  let currentLocale = "en";
  let selectedIdentity = null;

  function message(key) {
    const entry = catalog && catalog.messages ? catalog.messages[key] : null;
    if (!entry) return key;
    return entry[currentLocale] || entry[catalog.default_locale] || key;
  }

  function localeCandidates() {
    const candidates = [];
    const params = new URLSearchParams(window.location.search);
    const queryLocale = params.get("lang");
    const stored = window.localStorage.getItem(STORAGE_KEY);

    if (queryLocale) candidates.push(queryLocale);
    if (stored) candidates.push(stored);

    for (const locale of (navigator.languages || [navigator.language || ""])) {
      if (locale) candidates.push(locale);
    }

    candidates.push(catalog.default_locale);
    return candidates.map(function (value) {
      return String(value).trim().toLowerCase();
    });
  }

  function resolveLocale() {
    const supported = new Set(catalog.locales.map(function (item) {
      return item.code.toLowerCase();
    }));

    for (const candidate of localeCandidates()) {
      if (supported.has(candidate)) return candidate;

      const base = candidate.split("-")[0];
      if (supported.has(base)) return base;

      const alias = catalog.aliases[candidate];
      if (alias && supported.has(alias.toLowerCase())) return alias.toLowerCase();
    }

    return catalog.default_locale;
  }

  function applyTranslations() {
    document.documentElement.lang = currentLocale;
    document.title = message("primitive.meta.title");

    document.querySelectorAll("[data-i18n]").forEach(function (node) {
      node.textContent = message(node.dataset.i18n);
    });

    document.querySelectorAll("[data-i18n-aria]").forEach(function (node) {
      node.setAttribute("aria-label", message(node.dataset.i18nAria));
    });

    const clientStatus = document.getElementById("supabase-client-status");
    if (clientStatus) {
      clientStatus.textContent = supabaseClientState.configured
        ? message("primitive.data.supabase_configured")
        : message("primitive.data.supabase_unavailable");
    }
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

    applyTranslations();
    renderAll();
  }

  function presentation(item) {
    return item && item.presentation && typeof item.presentation === "object"
      ? item.presentation
      : {};
  }

  function displayName(item) {
    return presentation(item).name || item.identity || "Primitive";
  }

  function displaySummary(item) {
    return presentation(item).description
      || message("primitive.detail.unavailable");
  }

  function allItems() {
    const groups = (data && data.groups) || {};
    return GROUPS.flatMap(function (group) {
      return Array.isArray(groups[group.key]) ? groups[group.key] : [];
    });
  }

  function safeHttpUrl(value) {
    if (!value) return null;

    try {
      const url = new URL(value, window.location.href);
      return url.protocol === "https:" || url.protocol === "http:" ? url.href : null;
    } catch (_) {
      return null;
    }
  }

  function selectPrimitive(identity, updateUrl) {
    selectedIdentity = identity;

    if (updateUrl) {
      const url = new URL(window.location.href);
      if (identity) {
        url.searchParams.set("primitive", identity);
      } else {
        url.searchParams.delete("primitive");
      }
      window.history.replaceState({}, "", url);
    }

    renderDetail();
  }

  function primitiveCard(item) {
    const wrapper = document.createElement("article");
    wrapper.className = "primitive-card";

    const button = document.createElement("button");
    button.type = "button";

    const heading = document.createElement("h3");
    heading.textContent = displayName(item);

    const summary = document.createElement("p");
    summary.textContent = displaySummary(item);

    const identity = document.createElement("span");
    identity.className = "primitive-identity";
    identity.textContent = item.identity || message("primitive.detail.unavailable");

    button.append(heading, summary, identity);
    button.addEventListener("click", function () {
      selectPrimitive(item.identity || null, true);
    });

    wrapper.appendChild(button);
    return wrapper;
  }

  function renderGroup(group) {
    const groups = (data && data.groups) || {};
    const items = Array.isArray(groups[group.key]) ? groups[group.key] : [];
    const list = document.getElementById(group.listId);
    const count = document.getElementById(group.countId);

    count.textContent = String(items.length);
    list.textContent = "";

    if (!items.length) {
      const empty = document.createElement("div");
      empty.className = "empty-state";
      empty.textContent = message(group.emptyKey);
      list.appendChild(empty);
      return;
    }

    for (const item of items) {
      list.appendChild(primitiveCard(item));
    }
  }

  function renderValue(value) {
    if (value === null || value === undefined || value === "") {
      return document.createTextNode(message("primitive.detail.unavailable"));
    }

    if (typeof value === "object") {
      const pre = document.createElement("pre");
      pre.textContent = JSON.stringify(value, null, 2);
      return pre;
    }

    return document.createTextNode(String(value));
  }

  function detailRow(labelKey, value) {
    const row = document.createElement("div");
    row.className = "detail-row";

    const term = document.createElement("dt");
    term.textContent = message(labelKey);

    const description = document.createElement("dd");
    description.appendChild(renderValue(value));

    row.append(term, description);
    return row;
  }

  function renderDetail() {
    const target = document.getElementById("primitive-detail");
    target.textContent = "";

    const item = allItems().find(function (candidate) {
      return candidate.identity === selectedIdentity;
    });

    if (!item) {
      const heading = document.createElement("h2");
      heading.id = "primitive-detail-title";
      heading.textContent = message("primitive.detail.title");

      const placeholder = document.createElement("p");
      placeholder.className = "detail-placeholder";
      placeholder.textContent = message("primitive.detail.empty");

      target.append(heading, placeholder);
      return;
    }

    const view = presentation(item);
    const heading = document.createElement("h2");
    heading.id = "primitive-detail-title";
    heading.textContent = displayName(item);

    const summary = document.createElement("p");
    summary.textContent = displaySummary(item);

    const dl = document.createElement("dl");
    dl.className = "detail-grid";

    dl.append(
      detailRow("primitive.detail.identity", item.identity),
      detailRow("primitive.detail.status", item.status),
      detailRow("primitive.detail.author", view.author_or_registrant),
      detailRow("primitive.detail.source_language", view.source_language),
      detailRow("primitive.detail.description", view.description),
      detailRow("primitive.detail.input", view.input_contract),
      detailRow("primitive.detail.output", view.output_contract),
      detailRow("primitive.detail.parameters", view.parameter_contract),
      detailRow("primitive.detail.determinism", view.determinism_conformance),
      detailRow("primitive.detail.adoption", view.adoption),
      detailRow("primitive.detail.candidate", view.core_candidate),
      detailRow("primitive.detail.relationships", view.official_core_relationship)
    );

    const actions = document.createElement("div");
    actions.className = "detail-actions";

    const discussionUrl = safeHttpUrl(view.discussion_url);
    if (discussionUrl) {
      const discussion = document.createElement("a");
      discussion.className = "button secondary";
      discussion.href = discussionUrl;
      discussion.rel = "noopener noreferrer";
      discussion.textContent = message("primitive.detail.open_discussion");
      actions.appendChild(discussion);
    }

    const copy = document.createElement("button");
    copy.type = "button";
    copy.className = "button secondary";
    copy.textContent = message("primitive.detail.copy_link");
    copy.addEventListener("click", async function () {
      try {
        await navigator.clipboard.writeText(window.location.href);
      } catch (_) {
        // Clipboard support is optional; no fallback data collection is performed.
      }
    });
    actions.appendChild(copy);

    target.append(heading, summary, dl, actions);
  }

  function renderAll() {
    if (!catalog || !data) return;

    for (const group of GROUPS) {
      renderGroup(group);
    }

    renderDetail();
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
        loadJson(I18N_URL),
        loadJson(DATA_URL)
      ]);

      catalog = loaded[0];
      data = loaded[1];
      currentLocale = resolveLocale();
      selectedIdentity = new URLSearchParams(window.location.search).get("primitive");

      populateLanguageSelector();
      applyTranslations();
      renderAll();
    } catch (error) {
      console.error(error);
      document.documentElement.lang = "en";
    }
  }

  start();
}());
