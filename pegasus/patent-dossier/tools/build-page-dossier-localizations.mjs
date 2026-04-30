import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const dossierDir = path.resolve(__dirname, "..");
const pageDossiersDir = path.join(dossierDir, "page-dossiers");
const i18nDir = path.join(dossierDir, "i18n");

const LANGUAGE_CONFIG = {
  ru: {
    label: "Русский",
    notes: [
      "This file is a localized overlay for detailed page dossiers.",
      "Source English JSON remains the canonical evidence record; localized sentences preserve direct source anchors where exact UI labels should remain unchanged.",
      "Routes, endpoints, file paths, page IDs, and icon identifiers are intentionally preserved as technical anchors."
    ],
    pageSummary: (entry) => `Локализованный обзор поверхности \"${entry.pageId || entry.bundleId}\" для роли ${roleLabel(entry.role, "ru")} на платформе ${platformLabel(entry.platform, "ru")}.`,
    bundleSummary: (entry, count) => `Локализованный пакет \"${entry.bundleId}\" охватывает ${count} поверхностей приложения \"${entry.appId}\".`,
    purposeLabel: "Назначение",
    sourceAnchorLabel: "Source anchor",
    layoutLabel: "Структура макета",
    controlsLabel: "Элементы управления",
    iconsLabel: "Иконография",
    flowsLabel: "Интерактивные потоки",
    dependenciesLabel: "Зависимости данных",
    statesLabel: "Варианты состояния",
    figuresLabel: "Figure blueprints",
    minifeaturesLabel: "Минифункции",
    zoneSentence: (zoneId, position, contents, visibilityRule) => {
      const parts = [`Зона \"${zoneId}\" расположена в области \"${position}\".`];
      if (visibilityRule) {
        parts.push(`Правило видимости: \"${visibilityRule}\".`);
      }
      if (contents.length > 0) {
        parts.push(`Содержимое: ${contents.map(quote).join("; ")}.`);
      }
      return parts.join(" ");
    },
    simpleZoneSentence: (value) => `Зона макета: ${quote(value)}.`,
    buttonSentence: (button, zone, style, visibilityRule, icon) => {
      const parts = [`Кнопка ${quote(button)} расположена в \"${zone || "unspecified zone"}\".`];
      if (style) {
        parts.push(`Стиль: ${quote(style)}.`);
      }
      if (icon) {
        parts.push(`Иконка: ${quote(icon)}.`);
      }
      if (visibilityRule) {
        parts.push(`Правило видимости: ${quote(visibilityRule)}.`);
      }
      return parts.join(" ");
    },
    simpleButtonSentence: (value) => `Элемент управления: ${quote(value)}.`,
    iconSentence: (icon, zone) => `Иконка ${quote(icon)} используется в зоне ${quote(zone || "unspecified zone")}.`,
    simpleIconSentence: (value) => `Иконографическая привязка: ${quote(value)}.`,
    flowSummary: (flowId, steps) => ({
      flowId,
      summary: `Поток \"${flowId}\" содержит ${steps.length} шаг(а/ов).`,
      steps: steps.map((step, index) => `Шаг ${index + 1}: ${quote(step)}.`)
    }),
    simpleFlowSummary: (value) => ({
      summary: `Поток фиксируется как: ${quote(value)}.`
    }),
    dependencyReads: (reads) => `Чтение: ${reads.length ? reads.map(quote).join(", ") : "нет"}.`,
    dependencyWrites: (writes) => `Запись: ${writes.length ? writes.map(quote).join(", ") : "нет"}.`,
    dependencyNote: (key, value) => `${translateFieldName(key, "ru")}: ${quote(value)}.`,
    stateSentence: (value) => `${translateFieldName("state", "ru")}: ${quote(value)}.`,
    figureSentence: (value) => `${translateFieldName("figure", "ru")}: ${quote(value)}.`,
    minifeatureSentence: (value) => `${translateFieldName("minifeature", "ru")}: ${quote(value)}.`
  },
  uz: {
    label: "O'zbekcha",
    notes: [
      "This file is a localized overlay for detailed page dossiers.",
      "Asosiy inglizcha JSON fayllar kanonik dalil manbai bo'lib qoladi; lokalizatsiya qilingan gaplar aniq UI label va texnik identifikatorlarni source anchor sifatida saqlaydi.",
      "Route, endpoint, file path, page ID va icon nomlari texnik anchor sifatida o'zgartirilmaydi."
    ],
    pageSummary: (entry) => `\"${entry.pageId || entry.bundleId}\" yuzasi uchun ${roleLabel(entry.role, "uz")} roli va ${platformLabel(entry.platform, "uz")} platformasidagi lokalizatsiya qilingan ko'rinish.`,
    bundleSummary: (entry, count) => `\"${entry.bundleId}\" paketi \"${entry.appId}\" ilovasi uchun ${count} ta yuzani qamrab oladi.`,
    purposeLabel: "Maqsad",
    sourceAnchorLabel: "Source anchor",
    layoutLabel: "Layout tuzilmasi",
    controlsLabel: "Boshqaruv elementlari",
    iconsLabel: "Ikonografiya",
    flowsLabel: "Interaktiv oqimlar",
    dependenciesLabel: "Ma'lumot bog'liqliklari",
    statesLabel: "Holat variantlari",
    figuresLabel: "Figure blueprint'lar",
    minifeaturesLabel: "Mini-feature'lar",
    zoneSentence: (zoneId, position, contents, visibilityRule) => {
      const parts = [`\"${zoneId}\" zonasi \"${position}\" hududida joylashgan.`];
      if (visibilityRule) {
        parts.push(`Ko'rinish qoidasi: ${quote(visibilityRule)}.`);
      }
      if (contents.length > 0) {
        parts.push(`Tarkibi: ${contents.map(quote).join("; ")}.`);
      }
      return parts.join(" ");
    },
    simpleZoneSentence: (value) => `Layout zonasi: ${quote(value)}.`,
    buttonSentence: (button, zone, style, visibilityRule, icon) => {
      const parts = [`${quote(button)} tugmasi \"${zone || "unspecified zone"}\" hududida joylashgan.`];
      if (style) {
        parts.push(`Uslub: ${quote(style)}.`);
      }
      if (icon) {
        parts.push(`Ikona: ${quote(icon)}.`);
      }
      if (visibilityRule) {
        parts.push(`Ko'rinish qoidasi: ${quote(visibilityRule)}.`);
      }
      return parts.join(" ");
    },
    simpleButtonSentence: (value) => `Boshqaruv elementi: ${quote(value)}.`,
    iconSentence: (icon, zone) => `${quote(icon)} ikonasi ${quote(zone || "unspecified zone")} zonasida ishlatiladi.`,
    simpleIconSentence: (value) => `Ikona joylashuvi: ${quote(value)}.`,
    flowSummary: (flowId, steps) => ({
      flowId,
      summary: `\"${flowId}\" oqimi ${steps.length} ta qadamdan iborat.`,
      steps: steps.map((step, index) => `${index + 1}-qadam: ${quote(step)}.`)
    }),
    simpleFlowSummary: (value) => ({
      summary: `Oqim quyidagicha qayd etilgan: ${quote(value)}.`
    }),
    dependencyReads: (reads) => `O'qish: ${reads.length ? reads.map(quote).join(", ") : "yo'q"}.`,
    dependencyWrites: (writes) => `Yozish: ${writes.length ? writes.map(quote).join(", ") : "yo'q"}.`,
    dependencyNote: (key, value) => `${translateFieldName(key, "uz")}: ${quote(value)}.`,
    stateSentence: (value) => `${translateFieldName("state", "uz")}: ${quote(value)}.`,
    figureSentence: (value) => `${translateFieldName("figure", "uz")}: ${quote(value)}.`,
    minifeatureSentence: (value) => `${translateFieldName("minifeature", "uz")}: ${quote(value)}.`
  }
};

const ROLE_LABELS = {
  ru: {
    SUPPLIER: "поставщик",
    RETAILER: "ритейлер",
    DRIVER: "водитель",
    PAYLOAD: "payload-оператор"
  },
  uz: {
    SUPPLIER: "ta'minotchi",
    RETAILER: "chakana savdogar",
    DRIVER: "haydovchi",
    PAYLOAD: "payload operatori"
  }
};

const PLATFORM_LABELS = {
  ru: {
    web: "web",
    android: "android",
    ios: "iOS",
    "react-native-tablet": "React Native tablet"
  },
  uz: {
    web: "web",
    android: "android",
    ios: "iOS",
    "react-native-tablet": "React Native tablet"
  }
};

function main() {
  const files = fs.readdirSync(pageDossiersDir)
    .filter((name) => name.endsWith(".json"))
    .sort();

  for (const [lang, config] of Object.entries(LANGUAGE_CONFIG)) {
    const output = {
      generatedAt: new Date().toISOString(),
      language: lang,
      label: config.label,
      sourceFolder: "patent-dossier/page-dossiers",
      localizationMode: "overlay-with-source-anchors",
      notes: config.notes,
      fileCount: files.length,
      entries: files.map((name) => buildEntry(name, lang))
    };

    const targetFile = path.join(i18nDir, `page-dossiers.${lang}.json`);
    fs.writeFileSync(targetFile, `${JSON.stringify(output, null, 2)}\n`, "utf8");
  }

  console.log(`Generated page dossier localization overlays for ${Object.keys(LANGUAGE_CONFIG).join(", ")}.`);
}

function buildEntry(fileName, lang) {
  const sourcePath = path.join(pageDossiersDir, fileName);
  const record = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
  const base = {
    dossierFile: fileName,
    pageId: record.pageId,
    bundleId: record.bundleId,
    appId: record.appId,
    route: record.route,
    navRoute: record.navRoute,
    state: record.state,
    viewName: record.viewName,
    platform: record.platform,
    role: record.role,
    status: record.status,
    shell: record.shell,
    sourceFile: record.sourceFile,
    sourceFiles: record.sourceFiles
  };

  if (Array.isArray(record.surfaces)) {
    return {
      ...base,
      entryType: "bundle",
      localizedSummary: LANGUAGE_CONFIG[lang].bundleSummary(record, record.surfaces.length),
      surfaces: record.surfaces.map((surface) => buildSurface(surface, lang))
    };
  }

  return {
    ...base,
    entryType: "page",
    localizedSummary: LANGUAGE_CONFIG[lang].pageSummary(record),
    localized: buildLocalizedFields(record, lang)
  };
}

function buildSurface(surface, lang) {
  return {
    pageId: surface.pageId,
    navRoute: surface.navRoute,
    state: surface.state,
    viewName: surface.viewName,
    surfaceType: surface.surfaceType,
    sourceFile: surface.sourceFile,
    localizedSummary: LANGUAGE_CONFIG[lang].pageSummary(surface),
    localized: buildLocalizedFields(surface, lang)
  };
}

function buildLocalizedFields(record, lang) {
  const config = LANGUAGE_CONFIG[lang];
  const localized = {
    purpose: buildPurposeSummary(record, lang),
    purposeSourceAnchor: record.purpose || null
  };

  if (Array.isArray(record.layoutZones)) {
    localized.layoutOverview = record.layoutZones.map((zone) => describeZone(zone, lang));
  }

  if (Array.isArray(record.buttonPlacements)) {
    localized.controlOverview = record.buttonPlacements.map((button) => describeButton(button, lang));
  }

  if (Array.isArray(record.iconPlacements)) {
    localized.iconOverview = record.iconPlacements.map((icon) => describeIcon(icon, lang));
  }

  if (Array.isArray(record.interactiveFlows)) {
    localized.flowOverview = record.interactiveFlows.map((flow) => describeFlow(flow, lang));
  }

  if (record.dataDependencies) {
    localized.dependencyOverview = describeDependencies(record.dataDependencies, lang);
  }

  if (Array.isArray(record.stateVariants)) {
    localized.stateOverview = record.stateVariants.map((state) => config.stateSentence(state));
  }

  if (Array.isArray(record.figureBlueprints)) {
    localized.figureOverview = record.figureBlueprints.map((figure) => config.figureSentence(figure));
  }

  if (Array.isArray(record.minifeatures)) {
    localized.minifeatureOverview = record.minifeatures.map((feature) => config.minifeatureSentence(feature));
  }

  if (typeof record.minifeatureCount === "number") {
    localized.minifeatureCount = record.minifeatureCount;
  }

  return localized;
}

function describeZone(zone, lang) {
  const config = LANGUAGE_CONFIG[lang];

  if (typeof zone === "string") {
    return config.simpleZoneSentence(zone);
  }

  const contents = extractContents(zone.contents);
  return config.zoneSentence(zone.zoneId || "unlabeled-zone", zone.position || "unspecified position", contents, zone.visibilityRule);
}

function describeButton(button, lang) {
  const config = LANGUAGE_CONFIG[lang];

  if (typeof button === "string") {
    return config.simpleButtonSentence(button);
  }

  return config.buttonSentence(button.button || "unnamed button", button.zone, button.style, button.visibilityRule, button.icon);
}

function describeIcon(icon, lang) {
  const config = LANGUAGE_CONFIG[lang];

  if (typeof icon === "string") {
    return config.simpleIconSentence(icon);
  }

  return config.iconSentence(icon.icon || "unnamed icon", icon.zone);
}

function describeFlow(flow, lang) {
  const config = LANGUAGE_CONFIG[lang];

  if (typeof flow === "string") {
    return config.simpleFlowSummary(flow);
  }

  const steps = Array.isArray(flow.steps) ? flow.steps : [];
  return config.flowSummary(flow.flowId || "unnamed-flow", steps);
}

function describeDependencies(dataDependencies, lang) {
  const config = LANGUAGE_CONFIG[lang];
  const reads = normalizedArray(dataDependencies.readEndpoints || dataDependencies.read);
  const writes = normalizedArray(dataDependencies.writeEndpoints || dataDependencies.write);
  const notes = [];

  for (const [key, value] of Object.entries(dataDependencies)) {
    if (["readEndpoints", "writeEndpoints", "read", "write"].includes(key)) {
      continue;
    }
    if (value == null) {
      continue;
    }
    notes.push(config.dependencyNote(key, typeof value === "string" ? value : JSON.stringify(value)));
  }

  return {
    reads,
    writes,
    localizedNotes: [config.dependencyReads(reads), config.dependencyWrites(writes), ...notes]
  };
}

function extractContents(contents) {
  if (!contents) {
    return [];
  }

  if (Array.isArray(contents)) {
    return contents.flatMap((item) => extractContents(item));
  }

  if (typeof contents === "object") {
    return Object.entries(contents).flatMap(([key, value]) => {
      const nested = extractContents(value);
      if (nested.length === 0) {
        return [key];
      }
      return [`${key}: ${nested.join("; ")}`];
    });
  }

  return [String(contents)];
}

function normalizedArray(value) {
  if (!value) {
    return [];
  }
  return Array.isArray(value) ? value : [value];
}

function buildPurposeSummary(record, lang) {
  const identifier = record.pageId || record.bundleId || record.viewName || record.navRoute || record.route || "unknown-surface";
  const role = roleLabel(record.role, lang);
  const platform = platformLabel(record.platform, lang);
  const surfaceType = surfaceTypeLabel(record.surfaceType || "page", lang);

  if (lang === "ru") {
    return `Поверхность \"${identifier}\" представляет ${surfaceType} для роли ${role} на платформе ${platform}; исходное английское описание назначения сохранено в поле purposeSourceAnchor.`;
  }

  return `\"${identifier}\" yuzasi ${platform} platformasida ${role} roli uchun ${surfaceType} sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.`;
}

function surfaceTypeLabel(surfaceType, lang) {
  const map = {
    ru: {
      page: "страница",
      screen: "экран",
      overlay: "оверлей",
      "root-shell": "корневая оболочка",
      "root-gate": "корневой шлюз",
      "state-screen": "экран состояния",
      "action-region": "область действия"
    },
    uz: {
      page: "sahifa",
      screen: "ekran",
      overlay: "overlay",
      "root-shell": "ildiz shell",
      "root-gate": "ildiz gate",
      "state-screen": "holat ekrani",
      "action-region": "action hududi"
    }
  };

  return map[lang][surfaceType] || surfaceType;
}

function translateFieldName(key, lang) {
  const map = {
    ru: {
      state: "Состояние",
      figure: "Фигура",
      minifeature: "Минифункция",
      refreshModel: "Модель обновления",
      offlineFallback: "Офлайн-фолбэк"
    },
    uz: {
      state: "Holat",
      figure: "Figura",
      minifeature: "Mini-feature",
      refreshModel: "Yangilash modeli",
      offlineFallback: "Offline fallback"
    }
  };

  return map[lang][key] || key;
}

function roleLabel(role, lang) {
  return ROLE_LABELS[lang][role] || role || "unknown-role";
}

function platformLabel(platform, lang) {
  return PLATFORM_LABELS[lang][platform] || platform || "unknown-platform";
}

function quote(value) {
  return `\"${String(value)}\"`;
}

main();