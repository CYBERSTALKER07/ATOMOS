/**
 * Merge desktop-extract.json into catalogs/{en,ru,uz}.json with draft
 * ru/uz translations (reuse exact EN→locale map; else glossary + heuristics).
 *
 * Usage: node packages/i18n/scripts/merge-desktop-catalog.mjs
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
    if (!cursor[part] || typeof cursor[part] === "string") {
      cursor[part] = {};
    }
    cursor = cursor[part];
  }
  const leaf = parts[parts.length - 1];
  if (typeof cursor[leaf] === "undefined") {
    cursor[leaf] = value;
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

/** Phrase-level glossary seeded from existing catalogs + common UI. */
function buildPhraseMaps(enFlat, localeFlat) {
  const map = new Map();
  for (const [key, en] of Object.entries(enFlat)) {
    const loc = localeFlat[key];
    if (typeof en === "string" && typeof loc === "string" && en !== loc) {
      map.set(en, loc);
    }
  }
  return map;
}

const WORD_RU = {
  add: "Добавить",
  all: "Все",
  amount: "Сумма",
  apply: "Применить",
  back: "Назад",
  cancel: "Отмена",
  close: "Закрыть",
  create: "Создать",
  delete: "Удалить",
  description: "Описание",
  details: "Детали",
  edit: "Изменить",
  error: "Ошибка",
  filter: "Фильтр",
  loading: "Загрузка",
  name: "Название",
  next: "Далее",
  open: "Открыть",
  order: "Заказ",
  orders: "Заказы",
  previous: "Назад",
  refresh: "Обновить",
  remove: "Удалить",
  retry: "Повторить",
  save: "Сохранить",
  search: "Поиск",
  status: "Статус",
  submit: "Отправить",
  sync: "Синхронизировать",
  total: "Итого",
  update: "Обновить",
  view: "Просмотр",
  warehouse: "Склад",
  factory: "Завод",
  supplier: "Поставщик",
  retailer: "Ритейлер",
  driver: "Водитель",
  fleet: "Автопарк",
  inventory: "Инвентарь",
  payment: "Оплата",
  payments: "Платежи",
  settings: "Настройки",
  actions: "Действия",
  active: "Активные",
  inactive: "Неактивные",
  pending: "Ожидание",
  completed: "Завершено",
  failed: "Ошибка",
  today: "Сегодня",
  yesterday: "Вчера",
  revenue: "Выручка",
  products: "Товары",
  product: "Товар",
  quantity: "Количество",
  price: "Цена",
  date: "Дата",
  time: "Время",
  type: "Тип",
  notes: "Заметки",
  confirm: "Подтвердить",
  continue: "Продолжить",
  download: "Скачать",
  upload: "Загрузить",
  export: "Экспорт",
  import: "Импорт",
  yes: "Да",
  no: "Нет",
  none: "Нет",
  required: "Обязательно",
  optional: "Необязательно",
  enabled: "Включено",
  disabled: "Отключено",
  address: "Адрес",
  phone: "Телефон",
  email: "Эл. почта",
  password: "Пароль",
  account: "Аккаунт",
  dashboard: "Панель",
  overview: "Обзор",
  analytics: "Аналитика",
  billing: "Биллинг",
  catalog: "Каталог",
  map: "Карта",
  route: "Маршрут",
  routes: "Маршруты",
  delivery: "Доставка",
  deliveries: "Доставки",
  credit: "Кредит",
  claim: "Претензия",
  claims: "Претензии",
  alert: "Оповещение",
  alerts: "Оповещения",
  empty: "Пусто",
  unknown: "Неизвестно",
  configure: "Настроить",
  configuration: "Конфигурация",
  manage: "Управление",
  select: "Выбрать",
  selected: "Выбрано",
  clear: "Очистить",
  reset: "Сбросить",
  show: "Показать",
  hide: "Скрыть",
  more: "Ещё",
  less: "Меньше",
  of: "из",
  to: "до",
  from: "от",
  for: "для",
  with: "с",
  without: "без",
  and: "и",
  or: "или",
  new: "Новый",
  no_results: "Нет результатов",
};

const WORD_UZ = {
  add: "Qo'shish",
  all: "Barchasi",
  amount: "Summa",
  apply: "Qo'llash",
  back: "Orqaga",
  cancel: "Bekor qilish",
  close: "Yopish",
  create: "Yaratish",
  delete: "O'chirish",
  description: "Tavsif",
  details: "Tafsilotlar",
  edit: "Tahrirlash",
  error: "Xato",
  filter: "Filtr",
  loading: "Yuklanmoqda",
  name: "Nomi",
  next: "Keyingi",
  open: "Ochish",
  order: "Buyurtma",
  orders: "Buyurtmalar",
  previous: "Oldingi",
  refresh: "Yangilash",
  remove: "Olib tashlash",
  retry: "Qayta urinish",
  save: "Saqlash",
  search: "Qidiruv",
  status: "Holat",
  submit: "Yuborish",
  sync: "Sinxronlash",
  total: "Jami",
  update: "Yangilash",
  view: "Ko'rish",
  warehouse: "Ombor",
  factory: "Zavod",
  supplier: "Yetkazib beruvchi",
  retailer: "Chakana",
  driver: "Haydovchi",
  fleet: "Avtopark",
  inventory: "Inventar",
  payment: "To'lov",
  payments: "To'lovlar",
  settings: "Sozlamalar",
  actions: "Amallar",
  active: "Faol",
  inactive: "Nofaol",
  pending: "Kutilmoqda",
  completed: "Tugallangan",
  failed: "Muvaffaqiyatsiz",
  today: "Bugun",
  yesterday: "Kecha",
  revenue: "Daromad",
  products: "Mahsulotlar",
  product: "Mahsulot",
  quantity: "Miqdor",
  price: "Narx",
  date: "Sana",
  time: "Vaqt",
  type: "Tur",
  notes: "Izohlar",
  confirm: "Tasdiqlash",
  continue: "Davom etish",
  download: "Yuklab olish",
  upload: "Yuklash",
  export: "Eksport",
  import: "Import",
  yes: "Ha",
  no: "Yo'q",
  none: "Yo'q",
  required: "Majburiy",
  optional: "Ixtiyoriy",
  enabled: "Yoqilgan",
  disabled: "O'chirilgan",
  address: "Manzil",
  phone: "Telefon",
  email: "Email",
  password: "Parol",
  account: "Hisob",
  dashboard: "Boshqaruv paneli",
  overview: "Umumiy ko'rinish",
  analytics: "Analitika",
  billing: "Billing",
  catalog: "Katalog",
  map: "Xarita",
  route: "Marshrut",
  routes: "Marshrutlar",
  delivery: "Yetkazib berish",
  deliveries: "Yetkazib berishlar",
  credit: "Kredit",
  claim: "Da'vo",
  claims: "Da'volar",
  alert: "Ogohlantirish",
  alerts: "Ogohlantirishlar",
  empty: "Bo'sh",
  unknown: "Noma'lum",
  configure: "Sozlash",
  configuration: "Konfiguratsiya",
  manage: "Boshqarish",
  select: "Tanlash",
  selected: "Tanlangan",
  clear: "Tozalash",
  reset: "Qayta o'rnatish",
  show: "Ko'rsatish",
  hide: "Yashirish",
  more: "Yana",
  less: "Kamroq",
  of: "dan",
  to: "gacha",
  from: "dan",
  for: "uchun",
  with: "bilan",
  without: "siz",
  and: "va",
  or: "yoki",
  new: "Yangi",
};

function draftTranslate(en, locale, phraseMap) {
  if (phraseMap.has(en)) return phraseMap.get(en);

  const words = locale === "ru" ? WORD_RU : WORD_UZ;
  // Exact case-insensitive word
  const lower = en.toLowerCase();
  if (words[lower]) {
    return words[lower];
  }

  // Title-case multi-word: translate known tokens, keep unknowns
  const parts = en.split(/(\s+|[,/:|—–-]+)/);
  let translatedAny = false;
  const out = parts.map((part) => {
    if (/^\s+$/.test(part) || /^[,/:|—–-]+$/.test(part)) return part;
    const key = part.toLowerCase().replace(/[^a-z0-9']/gi, "");
    const hit = words[key];
    if (!hit) return part;
    translatedAny = true;
    // Preserve trailing punctuation on token
    const lead = part.match(/^[^a-zA-Z0-9]*/)?.[0] ?? "";
    const trail = part.match(/[^a-zA-Z0-9]*$/)?.[0] ?? "";
    if (part[0] && part[0] === part[0].toUpperCase() && /[a-z]/i.test(part[0])) {
      return lead + hit.charAt(0).toUpperCase() + hit.slice(1) + trail;
    }
    return lead + hit + trail;
  });

  if (translatedAny) return out.join("");

  // Fallback draft marker style matching "same language family" expectation:
  // keep English — honest draft residual for linguistic QA.
  // Prefer a light locale hint for short labels only.
  if (en.length <= 28 && /^[A-Za-z0-9 .,&'’/+()-]+$/.test(en)) {
    return en; // short unknown label; linguistic QA residual
  }
  return en;
}

const extractPath = path.resolve(generatedDir, "desktop-extract.json");
const extract = JSON.parse(fs.readFileSync(extractPath, "utf8"));

const catalogs = Object.fromEntries(
  supportedLocales.map((locale) => [locale, readCatalog(locale)]),
);
const flats = Object.fromEntries(
  supportedLocales.map((locale) => [locale, flattenCatalog(catalogs[locale])]),
);

const phraseRu = buildPhraseMaps(flats.en, flats.ru);
const phraseUz = buildPhraseMaps(flats.en, flats.uz);

deepMerge(catalogs.en, extract.en_additions);

const flatProposals = extract.proposals;
let translatedRu = 0;
let translatedUz = 0;
let passthrough = 0;

for (const p of flatProposals) {
  const en = p.text;
  const ru = draftTranslate(en, "ru", phraseRu);
  const uz = draftTranslate(en, "uz", phraseUz);
  if (ru !== en) translatedRu += 1;
  else passthrough += 1;
  if (uz !== en) translatedUz += 1;
  setNested(catalogs.ru, p.key, ru);
  setNested(catalogs.uz, p.key, uz);
}

for (const locale of supportedLocales) {
  const filePath = path.resolve(catalogsDir, `${locale}.json`);
  writeFile(filePath, `${JSON.stringify(catalogs[locale], null, 2)}\n`);
}

console.log(
  `Merged ${flatProposals.length} new keys. ru drafted≠en: ${translatedRu}, uz drafted≠en: ${translatedUz}, en-passthrough residuals: ${passthrough}`,
);
console.log(`Updated catalogs in ${catalogsDir}`);
