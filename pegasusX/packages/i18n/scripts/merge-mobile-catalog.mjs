/**
 * Merge mobile-extract.json into catalogs with draft ru/uz.
 * Usage: node packages/i18n/scripts/merge-mobile-catalog.mjs
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

function setNested(tree, dottedKey, value) {
  const parts = dottedKey.split(".");
  let cursor = tree;
  for (let i = 0; i < parts.length - 1; i++) {
    const part = parts[i];
    if (!cursor[part] || typeof cursor[part] === "string") cursor[part] = {};
    cursor = cursor[part];
  }
  if (typeof cursor[parts[parts.length - 1]] === "undefined") {
    cursor[parts[parts.length - 1]] = value;
  }
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

const extract = JSON.parse(
  fs.readFileSync(path.resolve(generatedDir, "mobile-extract.json"), "utf8"),
);

const catalogs = Object.fromEntries(
  supportedLocales.map((l) => [l, readCatalog(l)]),
);
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
};

function draft(en, locale, phraseMap) {
  if (phraseMap.has(en)) return phraseMap.get(en);
  const words = locale === "ru" ? WORD_RU : WORD_UZ;
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

deepMerge(catalogs.en, extract.en_additions);

let translatedRu = 0;
for (const p of extract.proposals) {
  const ru = draft(p.text, "ru", phraseRu);
  const uz = draft(p.text, "uz", phraseUz);
  if (ru !== p.text) translatedRu += 1;
  setNested(catalogs.ru, p.key, ru);
  setNested(catalogs.uz, p.key, uz);
}

for (const locale of supportedLocales) {
  writeFile(
    path.resolve(catalogsDir, `${locale}.json`),
    `${JSON.stringify(catalogs[locale], null, 2)}\n`,
  );
}

console.log(
  `Merged ${extract.proposals.length} mobile keys; ru drafted≠en: ${translatedRu}`,
);
