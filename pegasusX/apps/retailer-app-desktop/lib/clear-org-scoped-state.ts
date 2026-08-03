/**
 * Wave C1.3 hard client contract: on select-org / switch-org, clear all
 * org-scoped local state — not cart alone.
 *
 * Must clear:
 * 1. Cart / hold draft
 * 2. Open POS session (local)
 * 3. Offline count drafts
 * 4. In-memory assist context
 *
 * Listeners: window event `pegasusx:org-switched` so React providers reset.
 */
import { clearParkedPosCart } from "./pending-pos-sales";

export const ORG_SWITCHED_EVENT = "pegasusx:org-switched";

/** localStorage / sessionStorage keys known to be org-scoped. */
const LOCAL_KEYS = [
  "retailer_cart",
  "retailer_pos_parked_cart_v1",
  "retailer_pending_pos_sales_v1",
  "retailer_stock_count_draft_v1",
  "retailer_assist_context_v1",
  "retailer_pos_session_v1",
] as const;

const SESSION_KEYS = [
  "retailer_assist_context_v1",
  "retailer_pos_session_v1",
  "retailer_stock_count_draft_v1",
  "dock_pending_patches_v1",
] as const;

export type ClearOrgScopedStateResult = {
  clearedLocalKeys: string[];
  clearedSessionKeys: string[];
  parkedCartCleared: boolean;
};

export async function clearOrgScopedState(): Promise<ClearOrgScopedStateResult> {
  const clearedLocalKeys: string[] = [];
  const clearedSessionKeys: string[] = [];

  if (typeof window !== "undefined") {
    for (const key of LOCAL_KEYS) {
      try {
        if (window.localStorage.getItem(key) != null) {
          window.localStorage.removeItem(key);
          clearedLocalKeys.push(key);
        }
      } catch {
        /* private mode */
      }
    }
    for (const key of SESSION_KEYS) {
      try {
        if (window.sessionStorage.getItem(key) != null) {
          window.sessionStorage.removeItem(key);
          clearedSessionKeys.push(key);
        }
      } catch {
        /* private mode */
      }
    }
    // Prefix sweep for future count drafts / assist drafts
    try {
      const prefixes = ["retailer_count_draft_", "retailer_assist_", "retailer_pos_"];
      for (let i = window.localStorage.length - 1; i >= 0; i--) {
        const k = window.localStorage.key(i);
        if (!k) continue;
        if (prefixes.some((p) => k.startsWith(p))) {
          window.localStorage.removeItem(k);
          if (!clearedLocalKeys.includes(k)) clearedLocalKeys.push(k);
        }
      }
    } catch {
      /* ignore */
    }
  }

  let parkedCartCleared = false;
  try {
    await clearParkedPosCart();
    parkedCartCleared = true;
  } catch {
    parkedCartCleared = false;
  }

  if (typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent(ORG_SWITCHED_EVENT, {
        detail: { at: Date.now(), clearedLocalKeys, clearedSessionKeys },
      }),
    );
  }

  return { clearedLocalKeys, clearedSessionKeys, parkedCartCleared };
}
