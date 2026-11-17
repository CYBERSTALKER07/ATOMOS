import type { Locale } from "@pegasusx/i18n";

export type ArPaymentMethod = "WALLET" | "CARD" | "BANK_TRANSFER" | "CASH";

export interface CreditRelationshipTerms {
  terms_days?: number;
  grace_period_days?: number;
}

export interface CreditRelationship {
  supplier_id: string;
  supplier_name?: string;
  credit_enabled: boolean;
  terms_days: number;
  grace_period_days: number;
  credit_limit_minor: number;
  available_credit_minor: number;
  current_balance_minor: number;
  profile_status?: string;
  on_hold: boolean;
  prepay_only_locked?: boolean;
  resolved_terms?: CreditRelationshipTerms;
}

export interface ArInvoice {
  invoice_id: string;
  supplier_id?: string;
  order_id?: string;
  status: string;
  principal_minor?: number;
  balance_minor: number;
  disputed_minor?: number;
  undisputed_minor?: number;
  currency?: string;
  due_at?: string;
  created_at?: string;
  terms_days?: number;
  grace_period_days?: number;
  aging_bucket?: string;
  dunning_step?: number;
}

export interface ArPaymentPromise {
  promise_id: string;
  supplier_id: string;
  invoice_id?: string;
  quarter_key: string;
  due_at: string;
  promised_minor: number;
  status: string;
  approved_by?: string;
  created_at?: string;
}

export interface ArLedgerEntry {
  entry_id?: string;
  id?: string;
  invoice_id?: string;
  supplier_id?: string;
  entry_type?: string;
  type?: string;
  amount_minor: number;
  balance_after_minor?: number;
  reference?: string;
  note?: string;
  created_at?: string;
}

export interface ArStatement {
  supplier_id: string;
  currency: string;
  total_open_minor: number;
  total_disputed_minor: number;
  total_undisputed_minor: number;
  advance_balance_minor: number;
  current_quarter_key: string;
  self_serve_promise_used: boolean;
  max_extension_days: number;
  invoices: ArInvoice[];
  promises: ArPaymentPromise[];
  ledger_entries: ArLedgerEntry[];
}

export interface PayArFifoRequest {
  supplier_id: string;
  amount_minor: number;
  currency: string;
  payment_method: ArPaymentMethod;
  payment_reference?: string;
  earmark_invoice_id?: string;
}

export interface PayArFifoAllocation {
  invoice_id: string;
  order_id?: string;
  applied_minor: number;
  remaining_balance_minor: number;
  status: string;
}

export interface PayArFifoResponse {
  paid_minor: number;
  advance_added_minor: number;
  advance_balance_minor: number;
  allocations: PayArFifoAllocation[];
}

export interface PayArInvoiceRequest {
  amount_minor: number;
  payment_method: ArPaymentMethod;
  payment_reference?: string;
}

export interface CreateArPromiseRequest {
  supplier_id: string;
  invoice_id?: string;
  extension_days: number;
}

export interface ArPromisesResponse {
  promises: ArPaymentPromise[];
  current_quarter_key?: string;
  self_serve_promise_used?: boolean;
  max_extension_days?: number;
}

export interface QuarterlyPromiseLimitError {
  code: string;
  quarterKey?: string;
  message?: string;
}

export class ArApiError extends Error {
  status: number;
  code?: string;
  quarterKey?: string;
  body?: unknown;

  constructor(
    message: string,
    options: {
      status: number;
      code?: string;
      quarterKey?: string;
      body?: unknown;
    },
  ) {
    super(message);
    this.name = "ArApiError";
    this.status = options.status;
    this.code = options.code;
    this.quarterKey = options.quarterKey;
    this.body = options.body;
  }
}

export function generateIdempotencyUuid(): string {
  const cryptoApi = globalThis.crypto;
  if (cryptoApi && typeof cryptoApi.randomUUID === "function") {
    return cryptoApi.randomUUID();
  }
  const bytes = new Uint8Array(16);
  if (cryptoApi && typeof cryptoApi.getRandomValues === "function") {
    cryptoApi.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i += 1) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
  }
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0"));
  return `${hex.slice(0, 4).join("")}-${hex.slice(4, 6).join("")}-${hex.slice(6, 8).join("")}-${hex.slice(8, 10).join("")}-${hex.slice(10, 16).join("")}`;
}

function parseSortableTimestamp(value?: string): number {
  if (!value) {
    return Number.POSITIVE_INFINITY;
  }
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? ms : Number.POSITIVE_INFINITY;
}

export function sortArInvoicesOldestFirst(invoices: ArInvoice[]): ArInvoice[] {
  return [...invoices].sort((a, b) => {
    const dueDiff = parseSortableTimestamp(a.due_at) - parseSortableTimestamp(b.due_at);
    if (dueDiff !== 0) {
      return dueDiff;
    }
    const createdDiff =
      parseSortableTimestamp(a.created_at) - parseSortableTimestamp(b.created_at);
    if (createdDiff !== 0) {
      return createdDiff;
    }
    return a.invoice_id.localeCompare(b.invoice_id);
  });
}

export function resolveRelationshipTermsDays(rel: CreditRelationship): number {
  if (typeof rel.terms_days === "number" && rel.terms_days > 0) {
    return rel.terms_days;
  }
  return rel.resolved_terms?.terms_days ?? rel.terms_days ?? 0;
}

export function resolveRelationshipGraceDays(rel: CreditRelationship): number {
  if (typeof rel.grace_period_days === "number" && rel.grace_period_days > 0) {
    return rel.grace_period_days;
  }
  return rel.resolved_terms?.grace_period_days ?? rel.grace_period_days ?? 0;
}

export function extractArErrorPayload(
  body: unknown,
  status: number,
  fallback: string,
): { code?: string; quarterKey?: string; message: string } {
  if (body && typeof body === "object") {
    const obj = body as Record<string, unknown>;
    const code =
      typeof obj.code === "string"
        ? obj.code
        : typeof obj.error === "string"
          ? obj.error
          : undefined;
    const quarterKey =
      typeof obj.quarter_key === "string" ? obj.quarter_key : undefined;
    const rawMsg =
      typeof obj.message === "string" && obj.message.trim()
        ? obj.message
        : typeof obj.detail === "string" && obj.detail.trim()
          ? obj.detail
          : typeof obj.error === "string" && obj.error.trim()
            ? obj.error
            : `${fallback}_${status}`;
    return { code, quarterKey, message: rawMsg };
  }
  return { message: `${fallback}_${status}` };
}

export function parseQuarterlyPromiseLimitError(
  err: unknown,
): QuarterlyPromiseLimitError | null {
  if (err instanceof ArApiError) {
    const bodyObj =
      err.body && typeof err.body === "object"
        ? (err.body as Record<string, unknown>)
        : null;
    const bodyCode =
      typeof bodyObj?.code === "string"
        ? bodyObj.code
        : typeof bodyObj?.error === "string"
          ? bodyObj.error
          : undefined;
    const code = err.code || bodyCode;
    if (
      code === "quarterly_promise_limit_reached" ||
      err.message.includes("quarterly_promise_limit_reached") ||
      (err.status === 409 && !code)
    ) {
      return {
        code: code || "quarterly_promise_limit_reached",
        quarterKey:
          err.quarterKey ||
          (typeof bodyObj?.quarter_key === "string"
            ? bodyObj.quarter_key
            : undefined),
        message: err.message,
      };
    }
    return null;
  }
  if (err instanceof Error && err.message.includes("quarterly_promise_limit_reached")) {
    return {
      code: "quarterly_promise_limit_reached",
      message: err.message,
    };
  }
  return null;
}

export function parseArApiErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof ArApiError) {
    return err.message || fallback;
  }
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return fallback;
}

export interface W3CreditCopy {
  eyebrow: string;
  title: string;
  subtitle: string;
  refresh: string;
  loading: string;
  emptySuppliers: string;
  suppliersHeading: string;
  available: string;
  balanceDue: string;
  limit: string;
  terms: string;
  graceDays: string;
  daysUnit: string;
  onHold: string;
  prepayOnly: string;
  activeBadge: string;
  statementHeading: string;
  quarterLabel: string;
  totalOpen: string;
  undisputed: string;
  disputed: string;
  advanceBalance: string;
  payHeading: string;
  payModeFifo: string;
  payModeEarmark: string;
  payModeFifoHint: string;
  payModeEarmarkHint: string;
  selectInvoiceLabel: string;
  amountLabel: string;
  amountPlaceholder: string;
  invalidAmount: string;
  methodLabel: string;
  referenceLabel: string;
  referencePlaceholder: string;
  submitPayFifo: string;
  submitPayEarmark: string;
  paying: string;
  paymentSuccessTitle: string;
  paidSummary: string;
  advanceAddedSummary: string;
  promisesHeading: string;
  promiseUsedBadge: string;
  promiseAvailableBadge: string;
  promisePrepayBlocked: string;
  extensionDaysLabel: string;
  promiseInvoiceOptional: string;
  requestExtension: string;
  requestingExtension: string;
  promiseCreated: string;
  quarterlyLimitTitle: string;
  quarterlyLimitDetail: string;
  noPromises: string;
  invoicesHeading: string;
  oldestFirstNote: string;
  noOpenInvoices: string;
  invoiceOrderPrefix: string;
  dueLabel: string;
  agingLabel: string;
  disputedTag: string;
  undisputedTag: string;
  dunningLabel: string;
  payThisInvoice: string;
  useUndisputedFill: string;
  methods: Record<ArPaymentMethod, string>;
  loadError: string;
  paymentError: string;
  promiseError: string;
}

export const W3_CREDIT_COPY: Record<Locale, W3CreditCopy> = {
  uz: {
    eyebrow: "Kredit va debitor qarzlar",
    title: "Kredit liniyalari va to'lovlar",
    subtitle:
      "Yetkazib beruvchilar bo'yicha kredit limitlari, eng eski hisob-fakturalarni FIFO tartibida yopish, avans balansi va choraklik muddat uzaytirish va'dasi.",
    refresh: "Yangilash",
    loading: "Kredit va qarz ma'lumotlari yuklanmoqda...",
    emptySuppliers: "Faol kredit liniyalari topilmadi.",
    suppliersHeading: "Yetkazib beruvchi kredit liniyalari",
    available: "Mavjud",
    balanceDue: "Joriy qarz",
    limit: "Limit",
    terms: "Muddat",
    graceDays: "Imtiyoz davri",
    daysUnit: "kun",
    onHold: "TO'XTATILGAN",
    prepayOnly: "Faqat oldindan to'lov",
    activeBadge: "Faol",
    statementHeading: "Debitor hisobot va avans",
    quarterLabel: "Chorak",
    totalOpen: "Jami ochiq qarz",
    undisputed: "Bahssiz qarz",
    disputed: "Bahsli qism",
    advanceBalance: "Avans balansi",
    payHeading: "Hisob-fakturalarni to'lash",
    payModeFifo: "Eng eski hisobdan boshlab (FIFO)",
    payModeEarmark: "Aniq hisob-fakturaga bog'lash",
    payModeFifoHint:
      "To'lov avtomatik ravishda eng eski ochiq hisob-fakturalarga taqsimlanadi; ortiqcha summa avans balansiga o'tadi.",
    payModeEarmarkHint:
      "To'lov tanlangan hisob-fakturani yopishga yo'naltiriladi.",
    selectInvoiceLabel: "Hisob-fakturani tanlang",
    amountLabel: "To'lov summasi (so'm)",
    amountPlaceholder: "Masalan, 500000",
    invalidAmount: "Butun so'mda musbat to'lov summasini kiriting.",
    methodLabel: "To'lov usuli",
    referenceLabel: "To'lov ma'lumotnomasi (ixtiyoriy)",
    referencePlaceholder: "Kvitansiya yoki tranzaksiya raqami",
    submitPayFifo: "FIFO bo'yicha to'lash",
    submitPayEarmark: "Tanlangan hisobni to'lash",
    paying: "To'lov yuborilmoqda...",
    paymentSuccessTitle: "To'lov qabul qilindi",
    paidSummary: "Yopilgan summa",
    advanceAddedSummary: "Avansga qo'shildi",
    promisesHeading: "To'lov muddatini uzaytirish (chorakda 1 marta)",
    promiseUsedBadge: "Choraklik limit ishlatilgan",
    promiseAvailableBadge: "1 marta uzaytirish mumkin",
    promisePrepayBlocked:
      "Faqat oldindan to'lov rejimi yoqilgani sababli o'z-o'ziga uzaytirish yopiq.",
    extensionDaysLabel: "Uzaytirish kunlari",
    promiseInvoiceOptional: "Bog'langan hisob-faktura (ixtiyoriy)",
    requestExtension: "Muddatni uzaytirish",
    requestingExtension: "So'rov yuborilmoqda...",
    promiseCreated: "To'lov muddatini uzaytirish va'dasi yaratildi.",
    quarterlyLimitTitle: "Choraklik va'da limiti tugadi",
    quarterlyLimitDetail:
      "Ushbu chorakda o'z-o'ziga to'lov muddatini uzaytirish imkoniyatidan allaqachon foydalanilgan.",
    noPromises: "Faol to'lov va'dalari yo'q.",
    invoicesHeading: "Ochiq va qisman to'langan hisob-fakturalar",
    oldestFirstNote: "Eng eski muddatdan boshlab tartiblangan",
    noOpenInvoices: "Tanlangan yetkazib beruvchi bo'yicha ochiq hisob-fakturalar yo'q.",
    invoiceOrderPrefix: "Buyurtma",
    dueLabel: "To'lov muddati",
    agingLabel: "Kechikish guruhi",
    disputedTag: "Bahsli",
    undisputedTag: "Bahssiz",
    dunningLabel: "Eslatma bosqichi",
    payThisInvoice: "Shu hisobni to'lash",
    useUndisputedFill: "Bahssiz summani to'ldirish",
    methods: {
      WALLET: "Hamyon",
      CARD: "Karta",
      BANK_TRANSFER: "Bank o'tkazmasi",
      CASH: "Naqd pul",
    },
    loadError: "Kredit ma'lumotlarini yuklab bo'lmadi.",
    paymentError: "To'lovni amalga oshirib bo'lmadi.",
    promiseError: "Muddatni uzaytirish so'rovi bajarilmadi.",
  },
  ru: {
    eyebrow: "Кредит и дебиторская задолженность",
    title: "Кредитные линии и погашение",
    subtitle:
      "Условия поставщиков, погашение старейших счетов по FIFO, авансовый баланс и квартальное обещание продления срока.",
    refresh: "Обновить",
    loading: "Загрузка данных по кредиту и счетам...",
    emptySuppliers: "Активные кредитные линии не найдены.",
    suppliersHeading: "Кредитные линии поставщиков",
    available: "Доступно",
    balanceDue: "Задолженность",
    limit: "Лимит",
    terms: "Срок",
    graceDays: "Льготный период",
    daysUnit: "дн.",
    onHold: "НА УДЕРЖАНИИ",
    prepayOnly: "Только предоплата",
    activeBadge: "Активен",
    statementHeading: "Выписка ДЗ и аванс",
    quarterLabel: "Квартал",
    totalOpen: "Всего открыто",
    undisputed: "Бесспорный остаток",
    disputed: "Оспаривается",
    advanceBalance: "Авансовый баланс",
    payHeading: "Оплата счетов",
    payModeFifo: "Сначала старые счета (FIFO)",
    payModeEarmark: "Привязать к конкретному счёту",
    payModeFifoHint:
      "Платёж распределяется по самым старым открытым счетам; излишек зачисляется в авансовый баланс.",
    payModeEarmarkHint:
      "Платёж будет направлен на выбранный открытый счёт.",
    selectInvoiceLabel: "Целевой счёт",
    amountLabel: "Сумма оплаты (so'm)",
    amountPlaceholder: "Например, 500000",
    invalidAmount: "Введите положительную целую сумму в so'm.",
    methodLabel: "Способ оплаты",
    referenceLabel: "Номер платежа (необязательно)",
    referencePlaceholder: "Номер квитанции или транзакции",
    submitPayFifo: "Оплатить по FIFO",
    submitPayEarmark: "Оплатить выбранный счёт",
    paying: "Отправка платежа...",
    paymentSuccessTitle: "Платёж проведён",
    paidSummary: "Зачтено",
    advanceAddedSummary: "Зачислено в аванс",
    promisesHeading: "Продление срока оплаты (1 раз в квартал)",
    promiseUsedBadge: "Квартальный лимит исчерпан",
    promiseAvailableBadge: "Доступно 1 продление",
    promisePrepayBlocked:
      "Самостоятельное продление недоступно в режиме «Только предоплата».",
    extensionDaysLabel: "Дней продления",
    promiseInvoiceOptional: "Связанный счёт (необязательно)",
    requestExtension: "Запросить продление",
    requestingExtension: "Отправка запроса...",
    promiseCreated: "Обещание продления срока оформлено.",
    quarterlyLimitTitle: "Квартальный лимит продлений исчерпан",
    quarterlyLimitDetail:
      "Вы уже использовали самостоятельное продление срока оплаты в этом квартале.",
    noPromises: "Активных обещаний оплаты нет.",
    invoicesHeading: "Открытые и частично оплаченные счета",
    oldestFirstNote: "Отсортировано от самых старых",
    noOpenInvoices: "Нет открытых счетов по выбранному поставщику.",
    invoiceOrderPrefix: "Заказ",
    dueLabel: "Срок",
    agingLabel: "Просрочка",
    disputedTag: "Оспаривается",
    undisputedTag: "К оплате",
    dunningLabel: "Шаг взыскания",
    payThisInvoice: "Оплатить этот счёт",
    useUndisputedFill: "Подставить бесспорную сумму",
    methods: {
      WALLET: "Кошелёк",
      CARD: "Карта",
      BANK_TRANSFER: "Банковский перевод",
      CASH: "Наличные",
    },
    loadError: "Не удалось загрузить данные кредита и ДЗ.",
    paymentError: "Не удалось провести оплату.",
    promiseError: "Не удалось оформить продление.",
  },
  en: {
    eyebrow: "Credit & receivables",
    title: "Credit lines & AR repayment",
    subtitle:
      "Supplier credit terms, oldest-first FIFO vs. earmarked invoice repayment, advance balance, and self-serve quarterly extension promise.",
    refresh: "Refresh",
    loading: "Loading credit lines and AR statement...",
    emptySuppliers: "No credit relationships found.",
    suppliersHeading: "Supplier credit lines",
    available: "Available",
    balanceDue: "Balance due",
    limit: "Limit",
    terms: "Terms",
    graceDays: "Grace days",
    daysUnit: "days",
    onHold: "ON HOLD",
    prepayOnly: "Prepay-Only",
    activeBadge: "Active",
    statementHeading: "AR statement & advance balance",
    quarterLabel: "Quarter",
    totalOpen: "Total open",
    undisputed: "Undisputed due",
    disputed: "Disputed",
    advanceBalance: "Advance balance",
    payHeading: "Pay open invoices",
    payModeFifo: "Oldest-first (FIFO)",
    payModeEarmark: "Earmark single invoice",
    payModeFifoHint:
      "Payment settles your oldest open invoices first; any remainder is credited to Advance balance.",
    payModeEarmarkHint:
      "Payment is directed specifically toward the selected open invoice.",
    selectInvoiceLabel: "Target invoice",
    amountLabel: "Payment amount (so'm)",
    amountPlaceholder: "e.g. 500000",
    invalidAmount: "Enter a positive whole-so'm payment amount.",
    methodLabel: "Payment method",
    referenceLabel: "Payment reference (optional)",
    referencePlaceholder: "Receipt or transfer reference",
    submitPayFifo: "Pay oldest-first (FIFO)",
    submitPayEarmark: "Pay selected invoice",
    paying: "Submitting payment...",
    paymentSuccessTitle: "Payment recorded",
    paidSummary: "Applied to invoices",
    advanceAddedSummary: "Added to advance",
    promisesHeading: "Self-serve payment extension (1x per quarter)",
    promiseUsedBadge: "Quarterly promise used",
    promiseAvailableBadge: "1x extension available",
    promisePrepayBlocked:
      "Self-serve extension is unavailable while Prepay-Only lock is active.",
    extensionDaysLabel: "Extension days",
    promiseInvoiceOptional: "Target invoice (optional)",
    requestExtension: "Request extension",
    requestingExtension: "Requesting...",
    promiseCreated: "Payment extension promise recorded.",
    quarterlyLimitTitle: "Quarterly promise limit reached",
    quarterlyLimitDetail:
      "You have already used your self-serve payment extension for this quarter.",
    noPromises: "No active payment promises.",
    invoicesHeading: "Open & partial invoices",
    oldestFirstNote: "Sorted oldest-first",
    noOpenInvoices: "No open or partial invoices for this supplier.",
    invoiceOrderPrefix: "Order",
    dueLabel: "Due",
    agingLabel: "Aging",
    disputedTag: "Disputed",
    undisputedTag: "Undisputed",
    dunningLabel: "Dunning step",
    payThisInvoice: "Pay invoice",
    useUndisputedFill: "Fill undisputed",
    methods: {
      WALLET: "Wallet",
      CARD: "Card",
      BANK_TRANSFER: "Bank transfer",
      CASH: "Cash",
    },
    loadError: "Failed to load credit and AR data.",
    paymentError: "Failed to submit AR payment.",
    promiseError: "Failed to create extension promise.",
  },
};
