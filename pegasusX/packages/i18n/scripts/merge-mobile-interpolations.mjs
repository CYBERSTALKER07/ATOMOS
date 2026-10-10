/**
 * Merge interpolation templates into catalogs; prune bad literal-interp keys;
 * draft ru/uz while preserving {placeholders}.
 *
 * Usage: node packages/i18n/scripts/merge-mobile-interpolations.mjs
 */
import fs from "node:fs";
import path from "node:path";

import {
  catalogsDir,
  flattenCatalog,
  generatedDir,
  readCatalog,
  supportedLocales,
  writeFile,
} from "./shared.mjs";
import { mapPlaceholders } from "./lib/placeholders.mjs";

function setNested(tree, dottedKey, value) {
  const parts = dottedKey.split(".");
  let cursor = tree;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (!cursor[part] || typeof cursor[part] === "string") cursor[part] = {};
    cursor = cursor[part];
  }
  cursor[parts[parts.length - 1]] = value;
}

function deepMerge(target, source) {
  for (const [key, value] of Object.entries(source)) {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      if (!target[key] || typeof target[key] !== "object") target[key] = {};
      deepMerge(target[key], value);
    } else if (typeof target[key] === "undefined") {
      target[key] = value;
    }
  }
  return target;
}

/** Remove leaves whose values look like raw source interpolations. */
function pruneBadLeaves(node) {
  let removed = 0;
  for (const [key, value] of Object.entries(node)) {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      removed += pruneBadLeaves(value);
      if (Object.keys(value).length === 0) delete node[key];
      continue;
    }
    if (typeof value !== "string") continue;
    // Raw source interpolations wrongly stored as catalog values.
    // Do NOT treat `$` + `{name}` currency templates as bad (e.g. `${amount}`).
    if (value.includes("\\(")) {
      delete node[key];
      removed += 1;
      continue;
    }
    if (/\$\{[^}]*[.\s(]/.test(value)) {
      delete node[key];
      removed += 1;
    }
  }
  return removed;
}

const extract = JSON.parse(
  fs.readFileSync(
    path.resolve(generatedDir, "mobile-interpolation-extract.json"),
    "utf8",
  ),
);

const catalogs = Object.fromEntries(
  supportedLocales.map((l) => [l, readCatalog(l)]),
);

let pruned = 0;
for (const locale of supportedLocales) {
  pruned += pruneBadLeaves(catalogs[locale]);
}

const flats = Object.fromEntries(
  supportedLocales.map((l) => [l, flattenCatalog(catalogs[l])]),
);

const phraseRu = new Map();
const phraseUz = new Map();
for (const [key, en] of Object.entries(flats.en)) {
  if (flats.ru[key] && flats.ru[key] !== en) phraseRu.set(en, flats.ru[key]);
  if (flats.uz[key] && flats.uz[key] !== en) phraseUz.set(en, flats.uz[key]);
}

const WORD_RU = {
  back: "Назад", cancel: "Отмена", close: "Закрыть", save: "Сохранить",
  retry: "Повторить", search: "Поиск", loading: "Загрузка", error: "Ошибка",
  orders: "Заказы", order: "Заказ", delivery: "Доставка", settings: "Настройки",
  scan: "Сканировать", confirm: "Подтвердить", reject: "Отклонить",
  accept: "Принять", driver: "Водитель", warehouse: "Склад", supplier: "Поставщик",
  retailer: "Ритейлер", factory: "Завод", online: "Онлайн", offline: "Офлайн",
  refresh: "Обновить", submit: "Отправить", next: "Далее", done: "Готово",
  yes: "Да", no: "Нет", all: "Все", active: "Активные", pending: "Ожидание",
  priority: "Приоритет", transfer: "Трансфер", request: "Заявка",
  manifest: "Манифест", expected: "Ожидается", shortfall: "Недостача",
  overage: "Излишек", eligible: "Доступно", until: "до", total: "Итого",
  units: "ед.", state: "Статус", route: "Маршрут", warehouse_colon: "Склад",
};

const WORD_UZ = {
  back: "Orqaga", cancel: "Bekor qilish", close: "Yopish", save: "Saqlash",
  retry: "Qayta urinish", search: "Qidiruv", loading: "Yuklanmoqda", error: "Xato",
  orders: "Buyurtmalar", order: "Buyurtma", delivery: "Yetkazib berish",
  settings: "Sozlamalar", scan: "Skanerlash", confirm: "Tasdiqlash",
  reject: "Rad etish", accept: "Qabul qilish", driver: "Haydovchi",
  warehouse: "Ombor", supplier: "Yetkazib beruvchi", retailer: "Chakana",
  factory: "Zavod", online: "Onlayn", offline: "Oflayn", refresh: "Yangilash",
  submit: "Yuborish", next: "Keyingi", done: "Tayyor", yes: "Ha", no: "Yo‘q",
  all: "Barchasi", active: "Faol", pending: "Kutilmoqda",
  priority: "Muhimlik", transfer: "Transfer", request: "So‘rov",
  manifest: "Manfest", expected: "Kutilgan", shortfall: "Kamomad",
  overage: "Ortiqcha", eligible: "Mavjud", until: "gacha", total: "Jami",
  units: "dona", state: "Holat", route: "Marshrut",
};

function draftWords(en, words) {
  const lower = en.toLowerCase();
  if (words[lower]) return words[lower];
  const parts = en.split(/(\s+)/);
  let changed = false;
  const out = parts.map((part) => {
    if (/^\s+$/.test(part)) return part;
    const key = part.toLowerCase().replace(/[^a-z0-9']/gi, "");
    const hit = words[key];
    if (!hit) return part;
    changed = true;
    return part[0] === part[0].toUpperCase()
      ? hit.charAt(0).toUpperCase() + hit.slice(1)
      : hit;
  });
  return changed ? out.join("") : en;
}

function draft(en, locale) {
  const words = locale === "ru" ? WORD_RU : WORD_UZ;
  const phraseMap = locale === "ru" ? phraseRu : phraseUz;
  return mapPlaceholders(en, (masked) => {
    if (phraseMap.has(masked)) return phraseMap.get(masked);
    return draftWords(masked, words);
  });
}

deepMerge(catalogs.en, extract.en_additions);

let drafted = 0;
for (const p of extract.proposals) {
  const ru = draft(p.template, "ru");
  const uz = draft(p.template, "uz");
  if (ru !== p.template) drafted += 1;
  setNested(catalogs.ru, p.key, ru);
  setNested(catalogs.uz, p.key, uz);
  // Ensure en has the template (deepMerge only fills missing)
  setNested(catalogs.en, p.key, p.template);
}

for (const locale of supportedLocales) {
  writeFile(
    path.resolve(catalogsDir, `${locale}.json`),
    `${JSON.stringify(catalogs[locale], null, 2)}\n`,
  );
}

console.log(
  `Interpolation merge: +${extract.new_key_count} keys; pruned bad leaves≈${pruned}; ru drafted≠en: ${drafted}`,
);
