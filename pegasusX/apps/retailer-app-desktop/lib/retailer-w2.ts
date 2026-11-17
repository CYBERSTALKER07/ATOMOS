import { formatUZS, type Locale } from "@pegasusx/i18n";
import { minorToWholeRound } from "./money";

export interface OwnerApprovalRequiredDetail {
  code: "owner_approval_required";
  request_id: string;
  client_order_id?: string;
  status: string;
  tripped_rule: string;
  amount_minor: number;
  threshold_minor?: number;
  remaining_cap_minor?: number;
  message: string;
  fallback_payment_methods: string[];
}

export interface RetailerApprovalItem {
  request_id: string;
  retailer_id: string;
  supplier_id?: string;
  requested_by_user: string;
  requested_by_name?: string;
  client_order_id?: string;
  amount_minor: number;
  tripped_rule: string;
  status: string;
  payload_kind?: string;
  payment_method?: string;
  approved_by_user?: string;
  decision_reason?: string;
  expires_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface RetailerMandate {
  can_receive_goods: boolean;
  can_pay_cash: boolean;
  can_use_credit: boolean;
  daily_credit_cap_minor: number;
  daily_cash_cap_minor: number;
  max_single_order_minor: number;
  updated_at?: string;
  updated_by?: string;
}

export interface RetailerDailyUsage {
  usage_date: string;
  spent_credit_minor: number;
  spent_cash_minor: number;
  order_count: number;
}

export interface RetailerMandateEnvelope {
  user_id: string;
  retailer_id: string;
  retailer_role: string;
  is_owner: boolean;
  mandate: RetailerMandate;
  daily_usage?: RetailerDailyUsage;
}

/**
 * Formats integer minor units (tiyin) into rounded whole `so'm` using `@pegasusx/i18n`'s `formatUZS`.
 * Never renders raw minor integers.
 */
export function formatWholeSom(
  minor: number | bigint | null | undefined,
  currency?: string | null,
): string {
  const whole = minorToWholeRound(minor);
  const code = String(currency ?? "").trim().toUpperCase();
  if (!code || code === "UZS" || code === "SO'M" || code === "SOM") {
    return formatUZS(BigInt(whole) * BigInt(100));
  }
  const grouped = formatUZS(BigInt(whole) * BigInt(100)).replace(/\s*so'm$/i, "");
  return `${grouped} ${code}`;
}

export function somInputToMinor(somText: string): number | null {
  const normalized = somText.replace(/[\s,_'\u00A0]/g, "").trim();
  if (!normalized) return null;
  if (!/^\d+$/.test(normalized)) return null;
  const som = Number(normalized);
  if (!Number.isSafeInteger(som) || som < 0) return null;
  const minor = som * 100;
  return Number.isSafeInteger(minor) ? minor : null;
}

export function minorToSomInput(minor: number | null | undefined): string {
  return String(Math.max(0, minorToWholeRound(minor ?? 0)));
}

export function parseOwnerApprovalRequired(
  status: number,
  body: unknown,
): OwnerApprovalRequiredDetail | null {
  if (status !== 409 || !body || typeof body !== "object") {
    return null;
  }
  const obj = body as Record<string, unknown>;
  const code = String(obj.code ?? obj.error ?? "").trim().toLowerCase();
  if (code !== "owner_approval_required") {
    return null;
  }
  const methodsRaw = Array.isArray(obj.fallback_payment_methods)
    ? obj.fallback_payment_methods
    : [];
  const methods = methodsRaw
    .map((m) => String(m ?? "").trim().toUpperCase())
    .filter(Boolean);

  return {
    code: "owner_approval_required",
    request_id: String(obj.request_id ?? ""),
    client_order_id:
      typeof obj.client_order_id === "string" ? obj.client_order_id : undefined,
    status: String(obj.status ?? "PENDING_OWNER"),
    tripped_rule: String(obj.tripped_rule ?? "SELLER_MANDATE"),
    amount_minor:
      typeof obj.amount_minor === "number" && Number.isFinite(obj.amount_minor)
        ? Math.round(obj.amount_minor)
        : 0,
    threshold_minor:
      typeof obj.threshold_minor === "number" && Number.isFinite(obj.threshold_minor)
        ? Math.round(obj.threshold_minor)
        : undefined,
    remaining_cap_minor:
      typeof obj.remaining_cap_minor === "number" && Number.isFinite(obj.remaining_cap_minor)
        ? Math.round(obj.remaining_cap_minor)
        : undefined,
    message:
      typeof obj.message === "string" && obj.message.trim()
        ? obj.message.trim()
        : typeof obj.detail === "string" && obj.detail.trim()
          ? obj.detail.trim()
          : "Order exceeds seller mandate and requires owner approval.",
    fallback_payment_methods: methods,
  };
}

export function parseClaimErrorMessage(
  errBody: unknown,
  status?: number,
  locale: Locale = "en",
): string {
  const c = W2_COPY[locale] ?? W2_COPY.en;
  const obj =
    errBody && typeof errBody === "object"
      ? (errBody as Record<string, unknown>)
      : null;
  const rawCode = String(obj?.code ?? obj?.error ?? "").trim().toLowerCase();
  const rawMsg = String(obj?.message ?? obj?.detail ?? "").trim();
  const combined = `${rawCode} ${rawMsg}`.toLowerCase();

  if (combined.includes("reused_photo_evidence_refused")) {
    return c.claimReusedPhoto;
  }
  if (combined.includes("photo_evidence_required")) {
    return c.claimPhotoRequired;
  }
  if (combined.includes("invalid_evidence_uri")) {
    return c.claimInvalidPhoto;
  }
  if (combined.includes("claim_window_expired")) {
    return c.claimWindowExpired;
  }
  if (rawMsg) return rawMsg;
  if (rawCode && rawCode !== "claim_failed") return rawCode;
  return status ? `Claim failed (${status})` : "Claim failed";
}

export const W2_COPY: Record<
  Locale,
  {
    claimReusedPhoto: string;
    claimPhotoRequired: string;
    claimInvalidPhoto: string;
    claimWindowExpired: string;
    sentToOwnerTitle: string;
    payCashInstead: string;
    payCardInstead: string;
    inboxTitle: string;
    inboxSubtitle: string;
    approve: string;
    reject: string;
    noPendingApprovals: string;
    rejectReasonPlaceholder: string;
    mandateButton: string;
    mandateTitle: string;
    mandateSave: string;
    mandateSaved: string;
    canReceiveGoods: string;
    canPayCash: string;
    canUseCredit: string;
    dailyCreditCapSom: string;
    dailyCashCapSom: string;
    maxSingleOrderSom: string;
  }
> = {
  en: {
    claimReusedPhoto:
      "This photo was already used for another claim. Please take or choose a new photo.",
    claimPhotoRequired: "Photo proof is required for this claim type.",
    claimInvalidPhoto: "Invalid photo evidence link. Please re-upload your photo.",
    claimWindowExpired: "Claim window has expired (48h post-delivery).",
    sentToOwnerTitle: "Sent to Owner for approval",
    payCashInstead: "Pay Cash instead",
    payCardInstead: "Pay Card instead",
    inboxTitle: "Owner Approval Inbox",
    inboxSubtitle: "Review seller orders that exceeded mandate caps",
    approve: "Approve",
    reject: "Reject",
    noPendingApprovals: "No pending approval requests.",
    rejectReasonPlaceholder: "Rejection reason (optional)",
    mandateButton: "Mandate",
    mandateTitle: "Seller Mandate Caps (so'm)",
    mandateSave: "Save mandate",
    mandateSaved: "Mandate saved",
    canReceiveGoods: "Can receive goods",
    canPayCash: "Can pay cash",
    canUseCredit: "Can use credit",
    dailyCreditCapSom: "Daily credit cap (so'm, 0 = unlimited)",
    dailyCashCapSom: "Daily cash cap (so'm, 0 = unlimited)",
    maxSingleOrderSom: "Max single order (so'm, 0 = unlimited)",
  },
  ru: {
    claimReusedPhoto:
      "Это фото уже использовалось в другой претензии. Сделайте или выберите новое фото.",
    claimPhotoRequired: "Для этого типа претензии требуется фото.",
    claimInvalidPhoto: "Недействительная ссылка на фото. Загрузите фото повторно.",
    claimWindowExpired: "Срок подачи претензии истёк (48 часов после доставки).",
    sentToOwnerTitle: "Отправлено владельцу на одобрение",
    payCashInstead: "Оплатить наличными",
    payCardInstead: "Оплатить картой",
    inboxTitle: "Запросы на одобрение",
    inboxSubtitle: "Заказы продавцов, превысившие лимиты мандата",
    approve: "Одобрить",
    reject: "Отклонить",
    noPendingApprovals: "Нет ожидающих запросов на одобрение.",
    rejectReasonPlaceholder: "Причина отклонения (необязательно)",
    mandateButton: "Мандат",
    mandateTitle: "Лимиты мандата продавца (so'm)",
    mandateSave: "Сохранить мандат",
    mandateSaved: "Мандат сохранён",
    canReceiveGoods: "Приёмка товара",
    canPayCash: "Оплата наличными",
    canUseCredit: "Покупка в кредит",
    dailyCreditCapSom: "Дневной лимит кредита (so'm, 0 = без лимита)",
    dailyCashCapSom: "Дневной лимит наличных (so'm, 0 = без лимита)",
    maxSingleOrderSom: "Макс. сумма одного заказа (so'm, 0 = без лимита)",
  },
  uz: {
    claimReusedPhoto:
      "Bu rasm boshqa da'voda ishlatilgan. Iltimos, yangi rasm oling yoki tanlang.",
    claimPhotoRequired: "Ushbu da'vo turi uchun foto tasdiq majburiy.",
    claimInvalidPhoto: "Rasm havolasi noto'g'ri. Iltimos, rasmni qayta yuklang.",
    claimWindowExpired: "Da'vo topshirish muddati tugagan (yetkazilgandan keyin 48 soat).",
    sentToOwnerTitle: "Tasdiqlash uchun egasiga yuborildi",
    payCashInstead: "Naqd to'lash",
    payCardInstead: "Karta bilan to'lash",
    inboxTitle: "Ega tasdiqlash qutisi",
    inboxSubtitle: "Vakolat limitidan oshgan buyurtmalarni tasdiqlang",
    approve: "Tasdiqlash",
    reject: "Rad etish",
    noPendingApprovals: "Kutilayotgan tasdiqlash so'rovlari yo'q.",
    rejectReasonPlaceholder: "Rad etish sababi (ixtiyoriy)",
    mandateButton: "Vakolat",
    mandateTitle: "Sotuvchi vakolati limitlari (so'm)",
    mandateSave: "Vakolatni saqlash",
    mandateSaved: "Vakolat saqlandi",
    canReceiveGoods: "Tovarni qabul qilish",
    canPayCash: "Naqd to'lash huquqi",
    canUseCredit: "Nasiya (kredit) huquqi",
    dailyCreditCapSom: "Kunlik kredit limiti (so'm, 0 = cheklanmagan)",
    dailyCashCapSom: "Kunlik naqd limiti (so'm, 0 = cheklanmagan)",
    maxSingleOrderSom: "Bitta buyurtma limiti (so'm, 0 = cheklanmagan)",
  },
};

export function formatTrippedRuleLabel(rule: string): string {
  switch (String(rule || "").trim().toUpperCase()) {
    case "DAILY_CREDIT_CAP_EXCEEDED":
      return "Daily credit cap exceeded";
    case "CREDIT_MANDATE_DISABLED":
      return "Credit purchases disabled by seller mandate";
    case "CASH_MANDATE_DISABLED":
      return "Cash purchases disabled by seller mandate";
    case "DAILY_CASH_CAP_EXCEEDED":
      return "Daily cash cap exceeded";
    case "MAX_SINGLE_ORDER_EXCEEDED":
      return "Single-order mandate limit exceeded";
    default:
      return rule ? rule.replace(/_/g, " ") : "Seller mandate";
  }
}
