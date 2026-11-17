import type { Locale } from "@pegasusx/i18n";
import { formatWholeSom } from "./retailer-w2";
import type {
  Order,
  RetailerAIPrediction,
} from "./types";
import type { CreditRelationship } from "./retailer-w3";

export interface FastHomeCreditProfile {
  retailer_id?: string;
  supplier_id?: string;
  credit_limit_minor?: number;
  current_balance_minor?: number;
  available_credit_minor?: number;
  risk_tier?: string;
  status?: string;
  delinquency_count?: number;
}

export type WorkspaceMode = "fast" | "full";

export const WORKSPACE_MODE_STORAGE_KEY = "pegasus_retailer_workspace_mode";
export const WORKSPACE_MODE_CHANGE_EVENT = "pegasus:retailer-workspace-mode-change";

export interface FastHomeTomorrowDelivery {
  order_id: string;
  supplier_id?: string;
  supplier_name?: string;
  state: string;
  item_count: number;
  amount_minor: number;
  currency: string;
  delivery_date?: string;
  window_label?: string;
}

export interface FastHomeProposalItem {
  product_id: string;
  sku?: string;
  product_name: string;
  quantity: number;
  unit_price_minor: number;
  line_total_minor: number;
}

export interface FastHomePendingProposal {
  order_id: string;
  source: string;
  supplier_id?: string;
  supplier_name?: string;
  state: string;
  confirmation_status?: string;
  proposed_delivery_date?: string;
  auto_confirm_at?: string;
  item_count: number;
  amount_minor: number;
  currency: string;
  items?: FastHomeProposalItem[];
}

export interface FastHomeActivePromise {
  promise_id?: string;
  promised_until?: string;
  promised_minor?: number;
  status?: string;
}

export interface RetailerFastHomePayload {
  server_now: string;
  cutoff_at?: string | null;
  next_delivery_date?: string | null;
  tomorrow_delivery?: FastHomeTomorrowDelivery | null;
  balance_owed_minor: number;
  overdue_minor: number;
  days_overdue: number;
  active_promise?: FastHomeActivePromise | null;
  credit_headroom_minor: number;
  pending_proposal?: FastHomePendingProposal | null;
  pending_approvals_count: number;
  locale?: string;
  currency?: string;
  degraded_fallback?: boolean;
}

export interface TelegramLinkStatusPayload {
  linked: boolean;
  telegram_chat_id?: string;
  username?: string;
  bot_username?: string;
  bot_deep_link?: string;
  deep_link?: string;
  linked_at?: string;
  locale?: string;
}

export interface TelegramLinkRequestPayload {
  ok?: boolean;
  otp_code: string;
  expires_at: string;
  bot_deep_link?: string;
  deep_link?: string;
}

export interface CutoffCountdownState {
  valid: boolean;
  expired: boolean;
  remainingMs: number;
  formatted: string;
}

export function computeServerClockOffsetMs(
  serverNowIso: string | null | undefined,
  clientNowMs: number = Date.now(),
): number {
  if (!serverNowIso) return 0;
  const parsed = Date.parse(serverNowIso);
  if (!Number.isFinite(parsed)) return 0;
  return parsed - clientNowMs;
}

export function computeCutoffCountdown(
  cutoffIso: string | null | undefined,
  serverOffsetMs: number,
  clientNowMs: number = Date.now(),
): CutoffCountdownState {
  if (!cutoffIso) {
    return { valid: false, expired: false, remainingMs: 0, formatted: "--:--:--" };
  }
  const cutoffMs = Date.parse(cutoffIso);
  if (!Number.isFinite(cutoffMs)) {
    return { valid: false, expired: false, remainingMs: 0, formatted: "--:--:--" };
  }
  const syncedNowMs = clientNowMs + serverOffsetMs;
  const diffMs = cutoffMs - syncedNowMs;
  if (diffMs <= 0) {
    return { valid: true, expired: true, remainingMs: 0, formatted: "00:00:00" };
  }
  const totalSec = Math.floor(diffMs / 1000);
  const hours = Math.floor(totalSec / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;
  return {
    valid: true,
    expired: false,
    remainingMs: diffMs,
    formatted: `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`,
  };
}

export function resolveDefaultWorkspaceMode(
  retailerRole?: string | null,
  isOwner?: boolean | null,
): WorkspaceMode {
  const normalized = (retailerRole ?? "").trim().toUpperCase();
  if (isOwner === true || normalized === "OWNER") {
    return "full";
  }
  if (normalized === "BUYER" || normalized === "CASHIER" || normalized === "MANAGER") {
    return "fast";
  }
  return "fast";
}

export function parseStoredWorkspaceMode(raw: string | null | undefined): WorkspaceMode | null {
  if (raw === "fast" || raw === "full") return raw;
  return null;
}

export function readStoredWorkspaceMode(): WorkspaceMode | null {
  if (typeof window === "undefined" || typeof window.localStorage?.getItem !== "function") {
    return null;
  }
  return parseStoredWorkspaceMode(window.localStorage.getItem(WORKSPACE_MODE_STORAGE_KEY));
}

export function writeStoredWorkspaceMode(mode: WorkspaceMode): void {
  if (typeof window === "undefined") return;
  try {
    if (typeof window.localStorage?.setItem === "function") {
      window.localStorage.setItem(WORKSPACE_MODE_STORAGE_KEY, mode);
    }
    window.dispatchEvent(new CustomEvent(WORKSPACE_MODE_CHANGE_EVENT, { detail: mode }));
  } catch {
    // ignore storage quota errors
  }
}

export function resolveEffectiveWorkspaceMode(
  persistedMode: WorkspaceMode | null,
  retailerRole?: string | null,
  isOwner?: boolean | null,
): WorkspaceMode {
  if (persistedMode === "fast" || persistedMode === "full") {
    return persistedMode;
  }
  return resolveDefaultWorkspaceMode(retailerRole, isOwner);
}

const ACTIVE_DELIVERY_STATES = new Set([
  "PENDING",
  "PENDING_REVIEW",
  "SCHEDULED",
  "AUTO_ACCEPTED",
  "LOADED",
  "DISPATCHED",
  "IN_TRANSIT",
  "ARRIVING",
  "ARRIVED",
  "ARRIVED_SHOP_CLOSED",
  "SHOP_CLOSED_PENDING",
]);

function nextCutoffUtcIso(now: Date): string {
  // Default Tashkent 18:00 (13:00 UTC) cutoff when no explicit proposal auto_confirm_at exists
  const cutoff = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), 13, 0, 0, 0),
  );
  if (cutoff.getTime() <= now.getTime()) {
    cutoff.setUTCDate(cutoff.getUTCDate() + 1);
  }
  return cutoff.toISOString();
}

export function buildFastHomeFallback(params: {
  orders?: Array<Order & { supplier_name?: string; currency?: string; estimated_delivery?: string }>;
  predictions?: RetailerAIPrediction[];
  creditProfile?: FastHomeCreditProfile | null;
  relationships?: CreditRelationship[];
  pendingApprovalsCount?: number;
  now?: Date;
  currency?: string;
}): RetailerFastHomePayload {
  const now = params.now ?? new Date();
  const orders = Array.isArray(params.orders) ? params.orders : [];
  const predictions = Array.isArray(params.predictions) ? params.predictions : [];
  const relationships = Array.isArray(params.relationships) ? params.relationships : [];
  const currency =
    params.currency ||
    orders.find((o) => Boolean(o.currency))?.currency ||
    predictions[0]?.currency ||
    "UZS";

  const activeDeliveries = orders
    .filter((o) => ACTIVE_DELIVERY_STATES.has(String(o.state || "").toUpperCase()))
    .sort((a, b) => {
      const aTime = Date.parse(
        a.proposed_delivery_date || a.estimated_delivery || a.deliver_before || a.created_at || "",
      );
      const bTime = Date.parse(
        b.proposed_delivery_date || b.estimated_delivery || b.deliver_before || b.created_at || "",
      );
      if (Number.isFinite(aTime) && Number.isFinite(bTime)) return aTime - bTime;
      if (Number.isFinite(aTime)) return -1;
      if (Number.isFinite(bTime)) return 1;
      return 0;
    });

  const tomorrowOrder = activeDeliveries[0] ?? null;
  const tomorrowDelivery: FastHomeTomorrowDelivery | null = tomorrowOrder
    ? {
        order_id: tomorrowOrder.order_id,
        supplier_id: tomorrowOrder.supplier_id,
        supplier_name: tomorrowOrder.supplier_name,
        state: tomorrowOrder.state,
        item_count: (tomorrowOrder.items ?? []).reduce((sum, item) => sum + (item.quantity || 0), 0),
        amount_minor: tomorrowOrder.amount ?? 0,
        currency: tomorrowOrder.currency || currency,
        delivery_date:
          tomorrowOrder.proposed_delivery_date ||
          tomorrowOrder.estimated_delivery ||
          tomorrowOrder.deliver_before ||
          undefined,
      }
    : null;

  const proposalOrder = orders.find((o) => {
    const state = String(o.state || "").toUpperCase();
    const conf = String(o.confirmation_status || "").toUpperCase();
    return (
      state === "PENDING_REVIEW" ||
      (o.order_source === "MANUAL_PREORDER" && state === "SCHEDULED" && (!conf || conf === "DRAFT")) ||
      conf === "PENDING_WAREHOUSE"
    );
  });

  let pendingProposal: FastHomePendingProposal | null = null;
  if (proposalOrder) {
    pendingProposal = {
      order_id: proposalOrder.order_id,
      source: proposalOrder.order_source || "MANUAL_PREORDER",
      supplier_id: proposalOrder.supplier_id,
      supplier_name: proposalOrder.supplier_name,
      state: proposalOrder.state,
      confirmation_status: proposalOrder.confirmation_status ?? undefined,
      proposed_delivery_date: proposalOrder.proposed_delivery_date ?? proposalOrder.deliver_before ?? undefined,
      auto_confirm_at: proposalOrder.auto_confirm_at ?? undefined,
      item_count: (proposalOrder.items ?? []).reduce((sum, item) => sum + (item.quantity || 0), 0),
      amount_minor: proposalOrder.amount ?? 0,
      currency: proposalOrder.currency || currency,
      items: (proposalOrder.items ?? []).map((item) => ({
        product_id: item.sku_id,
        sku: item.sku_id,
        product_name: item.sku_name || item.sku_id,
        quantity: item.quantity,
        unit_price_minor: item.unit_price,
        line_total_minor: item.unit_price * item.quantity,
      })),
    };
  } else if (predictions.length > 0) {
    const pred = predictions[0];
    const lineItems = pred.line_items ?? [];
    pendingProposal = {
      order_id: pred.order_id,
      source: pred.order_source || "AI_PREDICTED",
      state: pred.confirmation_status || "PENDING_REVIEW",
      confirmation_status: pred.confirmation_status,
      proposed_delivery_date: pred.requested_delivery_date,
      auto_confirm_at: pred.auto_confirm_at,
      item_count: lineItems.reduce((sum, item) => sum + (item.quantity || 0), 0),
      amount_minor: pred.total_minor ?? 0,
      currency: pred.currency || currency,
      items: lineItems.map((item) => ({
        product_id: item.sku,
        sku: item.sku,
        product_name: item.name || item.sku,
        quantity: item.quantity,
        unit_price_minor: item.unit_price_minor,
        line_total_minor: item.unit_price_minor * item.quantity,
      })),
    };
  }

  const relationshipBalance = relationships.reduce(
    (sum, rel) => sum + (rel.current_balance_minor || 0),
    0,
  );
  const relationshipHeadroom = relationships.reduce(
    (sum, rel) => sum + (rel.available_credit_minor || 0),
    0,
  );
  const balanceOwedMinor =
    params.creditProfile?.current_balance_minor ?? relationshipBalance;
  const creditHeadroomMinor =
    params.creditProfile?.available_credit_minor ?? relationshipHeadroom;
  const maxDelinquency = params.creditProfile?.delinquency_count || 0;
  const hasHold = relationships.some(
    (rel) => rel.on_hold || String(rel.profile_status || "").toUpperCase() === "ON_HOLD",
  );

  return {
    server_now: now.toISOString(),
    cutoff_at: pendingProposal?.auto_confirm_at || nextCutoffUtcIso(now),
    next_delivery_date: tomorrowDelivery?.delivery_date ?? pendingProposal?.proposed_delivery_date ?? null,
    tomorrow_delivery: tomorrowDelivery,
    balance_owed_minor: balanceOwedMinor,
    overdue_minor: hasHold ? balanceOwedMinor : 0,
    days_overdue: maxDelinquency > 0 ? maxDelinquency : 0,
    credit_headroom_minor: creditHeadroomMinor,
    pending_proposal: pendingProposal,
    pending_approvals_count: params.pendingApprovalsCount ?? 0,
    currency,
    degraded_fallback: true,
  };
}

export function formatFastWholeSom(minor: number, currency = "UZS"): string {
  return formatWholeSom(minor, currency);
}

export interface FastHomeCopy {
  fastModeLabel: string;
  fullErpLabel: string;
  fastModeTitle: string;
  fastModeSubtitle: string;
  switchToFull: string;
  switchToFast: string;
  pendingApprovalsCta: (count: number) => string;
  tomorrowDeliveryTitle: string;
  tomorrowDeliveryEmpty: string;
  tomorrowDeliveryBrowse: string;
  tomorrowDeliveryItems: (count: number) => string;
  trackDelivery: string;
  whatYouOweTitle: string;
  whatYouOweClean: string;
  whatYouOweOverdue: (amount: string, days: number) => string;
  creditHeadroom: (amount: string) => string;
  payNowCta: string;
  proposedOrderTitle: string;
  proposedOrderEmpty: string;
  proposedOrderNewOrder: string;
  cutoffLabel: string;
  cutoffExpired: string;
  confirmCta: string;
  editCta: string;
  holdCta: string;
  holdReasonPrompt: string;
  editTitle: string;
  editDeliveryDate: string;
  saveEditCta: string;
  cancelCta: string;
  actionConfirmed: string;
  actionHeld: string;
  actionUpdated: string;
  degradedNotice: string;
  telegramTitle: string;
  telegramSubtitle: string;
  telegramLinked: string;
  telegramNotLinked: string;
  telegramGenerateCode: string;
  telegramRefreshStatus: string;
  telegramCopyCode: string;
  telegramCopyLink: string;
  telegramOpenBot: string;
  telegramCopied: string;
  telegramExpiresAt: (time: string) => string;
}

export const FAST_HOME_COPY: Record<Locale, FastHomeCopy> = {
  uz: {
    fastModeLabel: "Tezkor rejim",
    fullErpLabel: "To'liq ERP",
    fastModeTitle: "Kunlik tezkor boshqaruv",
    fastModeSubtitle: "Ertaga keladigan yuk, qarz holati va bugungi buyurtma bir ekranda.",
    switchToFull: "To'liq ERPga o'tish",
    switchToFast: "Tezkor rejimga o'tish",
    pendingApprovalsCta: (count) => `${count} ta egasi tasdig'ini kutmoqda`,
    tomorrowDeliveryTitle: "Ertaga yetkazib berish",
    tomorrowDeliveryEmpty: "Ertaga rejalashtirilgan faol yetkazib berish yo'q.",
    tomorrowDeliveryBrowse: "Katalogdan buyurtma berish",
    tomorrowDeliveryItems: (count) => `${count} xil pozitsiya`,
    trackDelivery: "Kuzatish",
    whatYouOweTitle: "Joriy qarzingiz",
    whatYouOweClean: "Muddati o'tgan qarz yo'q",
    whatYouOweOverdue: (amount, days) => `Muddati o'tgan: ${amount} (${days} kun)`,
    creditHeadroom: (amount) => `Mavjud kredit limiti: ${amount}`,
    payNowCta: "To'lash / Kredit",
    proposedOrderTitle: "Bugungi taklif buyurtmasi",
    proposedOrderEmpty: "Tasdiqlashni kutayotgan buyurtma taklifi yo'q.",
    proposedOrderNewOrder: "Katalogga o'tish",
    cutoffLabel: "Kesim taymeri (server vaqti)",
    cutoffExpired: "Kesim muddati tugadi",
    confirmCta: "Tasdiqlash",
    editCta: "Tahrirlash",
    holdCta: "To'xtatish",
    holdReasonPrompt: "Bugungi taklif vaqtincha to'xtatildi",
    editTitle: "Buyurtma taklifini tahrirlash",
    editDeliveryDate: "Yetkazish sanasi (YYYY-MM-DD)",
    saveEditCta: "Saqlash",
    cancelCta: "Bekor qilish",
    actionConfirmed: "Buyurtma tasdiqlandi.",
    actionHeld: "Taklif to'xtatildi.",
    actionUpdated: "Taklif yangilandi.",
    degradedNotice: "Tezkor BFF vaqtincha mavjud emas — zaxira oqimdan ma'lumot olindi.",
    telegramTitle: "Telegram tezkor ulanish",
    telegramSubtitle: "Buyurtmalarni tasdiqlash, qarz va yetkazish xabarlarini Telegramda oling.",
    telegramLinked: "Telegram ulangan",
    telegramNotLinked: "Hali ulanmagan",
    telegramGenerateCode: "6 xonali kod olish",
    telegramRefreshStatus: "Holatni yangilash",
    telegramCopyCode: "Kodni nusxalash",
    telegramCopyLink: "Havolani nusxalash",
    telegramOpenBot: "Botni ochish",
    telegramCopied: "Nusxalandi",
    telegramExpiresAt: (time) => `Amal qilish muddati: ${time}`,
  },
  ru: {
    fastModeLabel: "Быстрый режим",
    fullErpLabel: "Полный ERP",
    fastModeTitle: "Быстрый рабочий стол",
    fastModeSubtitle: "Завтрашняя доставка, текущая задолженность и заказ дня на одном экране.",
    switchToFull: "Переключить на полный ERP",
    switchToFast: "Переключить на быстрый режим",
    pendingApprovalsCta: (count) => `Ожидает одобрения владельца: ${count}`,
    tomorrowDeliveryTitle: "Доставка на завтра",
    tomorrowDeliveryEmpty: "Активных доставок на завтра пока нет.",
    tomorrowDeliveryBrowse: "Открыть каталог",
    tomorrowDeliveryItems: (count) => `${count} поз.`,
    trackDelivery: "Отследить",
    whatYouOweTitle: "К оплате",
    whatYouOweClean: "Просроченной задолженности нет",
    whatYouOweOverdue: (amount, days) => `Просрочено: ${amount} (${days} дн.)`,
    creditHeadroom: (amount) => `Доступный лимит: ${amount}`,
    payNowCta: "Оплатить / Кредит",
    proposedOrderTitle: "Предложенный заказ на сегодня",
    proposedOrderEmpty: "Нет ожидающих подтверждения заказов.",
    proposedOrderNewOrder: "Перейти в каталог",
    cutoffLabel: "Таймер отсечки (сервер)",
    cutoffExpired: "Время отсечки истекло",
    confirmCta: "Подтвердить",
    editCta: "Изменить",
    holdCta: "Приостановить",
    holdReasonPrompt: "Отложено пользователем в быстром режиме",
    editTitle: "Изменить предложенный заказ",
    editDeliveryDate: "Дата доставки (YYYY-MM-DD)",
    saveEditCta: "Сохранить",
    cancelCta: "Отмена",
    actionConfirmed: "Заказ подтверждён.",
    actionHeld: "Заказ приостановлен.",
    actionUpdated: "Заказ обновлён.",
    degradedNotice: "Сервис Fast-Home недоступен — показаны резервные данные.",
    telegramTitle: "Быстрая привязка Telegram",
    telegramSubtitle: "Получайте уведомления о заказах и долге и подтверждайте заказы в Telegram.",
    telegramLinked: "Telegram привязан",
    telegramNotLinked: "Не привязан",
    telegramGenerateCode: "Получить 6-значный код",
    telegramRefreshStatus: "Обновить статус",
    telegramCopyCode: "Копировать код",
    telegramCopyLink: "Копировать ссылку",
    telegramOpenBot: "Открыть бота",
    telegramCopied: "Скопировано",
    telegramExpiresAt: (time) => `Действует до: ${time}`,
  },
  en: {
    fastModeLabel: "Fast Mode",
    fullErpLabel: "Full ERP",
    fastModeTitle: "Daily Fast Home",
    fastModeSubtitle: "Tomorrow's delivery, amount owed, and today's proposed order on one screen.",
    switchToFull: "Switch to Full ERP",
    switchToFast: "Switch to Fast Mode",
    pendingApprovalsCta: (count) => `${count} pending owner approvals`,
    tomorrowDeliveryTitle: "Tomorrow's delivery",
    tomorrowDeliveryEmpty: "No scheduled deliveries queued for tomorrow.",
    tomorrowDeliveryBrowse: "Order from catalog",
    tomorrowDeliveryItems: (count) => `${count} units`,
    trackDelivery: "Track",
    whatYouOweTitle: "What you owe",
    whatYouOweClean: "No overdue balance",
    whatYouOweOverdue: (amount, days) => `Overdue: ${amount} (${days}d)`,
    creditHeadroom: (amount) => `Credit headroom: ${amount}`,
    payNowCta: "Pay / Credit",
    proposedOrderTitle: "Today's proposed order",
    proposedOrderEmpty: "No proposed order awaiting confirmation.",
    proposedOrderNewOrder: "Browse catalog",
    cutoffLabel: "Order cutoff (server-synced)",
    cutoffExpired: "Cutoff passed",
    confirmCta: "Confirm",
    editCta: "Edit",
    holdCta: "Hold",
    holdReasonPrompt: "Held from Fast Mode home",
    editTitle: "Edit proposed order",
    editDeliveryDate: "Delivery date (YYYY-MM-DD)",
    saveEditCta: "Save changes",
    cancelCta: "Cancel",
    actionConfirmed: "Order confirmed.",
    actionHeld: "Order placed on hold.",
    actionUpdated: "Order proposal updated.",
    degradedNotice: "Fast-Home endpoint unavailable — showing fallback live feeds.",
    telegramTitle: "Telegram quick-bind",
    telegramSubtitle: "Confirm daily orders, check debt, and receive delivery updates via Telegram.",
    telegramLinked: "Telegram linked",
    telegramNotLinked: "Not linked yet",
    telegramGenerateCode: "Generate 6-digit OTP",
    telegramRefreshStatus: "Refresh status",
    telegramCopyCode: "Copy OTP",
    telegramCopyLink: "Copy link",
    telegramOpenBot: "Open bot",
    telegramCopied: "Copied",
    telegramExpiresAt: (time) => `Expires at: ${time}`,
  },
};
