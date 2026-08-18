"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Loader2,
  ShoppingCart,
  Trash2,
  CreditCard,
  Banknote,
  AlertTriangle,
  WifiOff,
  CloudUpload,
  PauseCircle,
  PlayCircle,
  XCircle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
import {
  clearParkedPosCart,
  countPendingForSession,
  enqueueOfflinePosSale,
  flushPendingPosSales,
  listPendingPosSales,
  loadParkedPosCart,
  newClientSaleId,
  provisionalReceipt,
  saveParkedPosCart,
  type PendingPosSale,
} from "@/lib/pending-pos-sales";
import { retailerPosSaleKey } from "@pegasusx/api-client";
import { moneyCurrency, sessionPackCurrency } from "@/lib/payment-catalog";

type Register = {
  register_id: string;
  label: string;
  location_id: string;
  status: string;
};

type Session = {
  session_id: string;
  register_id: string;
  status: string;
  opening_float_minor: number;
  currency: string;
};

type CartLine = {
  sku: string;
  name: string;
  qty: number;
  unit_price_minor: number;
};

type Sale = {
  sale_id: string;
  receipt_number: string;
  total_minor: number;
  status: string;
  lines: CartLine[];
  origin?: string;
  client_sale_id?: string;
};

/** Wave C3.1/C3.2 server parked cart (POS_HOLDS_ENABLED). */
type ServerHold = {
  hold_id: string;
  location_id: string;
  register_id?: string;
  status: string;
  cart: { lines?: CartLine[] } | CartLine[] | null;
  note?: string;
  expires_at?: string;
};

function formatMoney(minor: number, currency?: string) {
  return `${(minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })} ${moneyCurrency(currency)}`;
}

export default function POSPage() {
  const t = usePortalT();
  const [registers, setRegisters] = useState<Register[]>([]);
  const [registerId, setRegisterId] = useState("");
  const [session, setSession] = useState<Session | null>(null);
  const [cart, setCart] = useState<CartLine[]>([]);
  const [sku, setSku] = useState("");
  const [name, setName] = useState("");
  const [qty, setQty] = useState("1");
  const [price, setPrice] = useState("0");
  const [floatMinor, setFloatMinor] = useState("0");
  const [closingCash, setClosingCash] = useState("0");
  const [tender, setTender] = useState<"CASH" | "CARD">("CASH");
  const [banner, setBanner] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [lastSale, setLastSale] = useState<Sale | null>(null);
  const [newRegLabel, setNewRegLabel] = useState("Register 1");
  const [online, setOnline] = useState(
    typeof navigator === "undefined" ? true : navigator.onLine,
  );
  const [pending, setPending] = useState<PendingPosSale[]>([]);
  const [receiptSeq, setReceiptSeq] = useState(1);
  // C3.2 server holds (hidden when feature 404 / disabled)
  const [holdsEnabled, setHoldsEnabled] = useState(false);
  const [serverHolds, setServerHolds] = useState<ServerHold[]>([]);
  const [holdNote, setHoldNote] = useState("");

  const totalMinor = useMemo(
    () => cart.reduce((s, l) => s + l.qty * l.unit_price_minor, 0),
    [cart],
  );

  const pendingForSession = session
    ? countPendingForSession(pending, session.session_id)
    : 0;

  const refreshPending = useCallback(async () => {
    setPending(await listPendingPosSales());
  }, []);

  useEffect(() => {
    const on = () => setOnline(true);
    const off = () => setOnline(false);
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    void refreshPending();
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
    };
  }, [refreshPending]);

  useEffect(() => {
    if (!online) return;
    void flushPendingPosSales().then(async (r) => {
      if (r.flushed > 0) {
        setBanner(`Synced ${r.flushed} offline sale(s)`);
      }
      await refreshPending();
    });
  }, [online, refreshPending]);

  // Force cash when offline
  useEffect(() => {
    if (!online && tender === "CARD") setTender("CASH");
  }, [online, tender]);

  // Persist parked cart
  useEffect(() => {
    if (!session) return;
    void saveParkedPosCart({
      sessionId: session.session_id,
      lines: cart,
      updatedAt: Date.now(),
    });
  }, [cart, session]);

  const loadRegisters = useCallback(async () => {
    try {
      const res = await apiFetch("/v1/retailer/registers");
      if (!res.ok) return;
      const json = (await res.json()) as { items?: Register[] };
      const items = json.items ?? [];
      setRegisters(items);
      if (!registerId && items[0]) setRegisterId(items[0].register_id);
    } catch {
      /* ignore */
    }
  }, [registerId]);

  const activeLocationId = useMemo(() => {
    const reg = registers.find((r) => r.register_id === registerId) || registers[0];
    return reg?.location_id || "";
  }, [registers, registerId]);

  const loadServerHolds = useCallback(async () => {
    if (!online) return;
    try {
      const q = activeLocationId
        ? `?location_id=${encodeURIComponent(activeLocationId)}`
        : "";
      const res = await apiFetch(`/v1/retailer/pos/holds${q}`);
      if (res.status === 404) {
        setHoldsEnabled(false);
        setServerHolds([]);
        return;
      }
      if (!res.ok) return;
      setHoldsEnabled(true);
      const json = (await res.json()) as { items?: ServerHold[] };
      setServerHolds(json.items ?? []);
    } catch {
      /* feature optional */
    }
  }, [online, activeLocationId]);

  useEffect(() => {
    void loadRegisters();
  }, [loadRegisters]);

  useEffect(() => {
    void loadServerHolds();
  }, [loadServerHolds]);

  const parkServerHold = async () => {
    if (!session || cart.length === 0 || !activeLocationId) return;
    if (!online) {
      setError(t("retailer_desktop.residual.text.park_hold_requires_network_use_local_auto_park_offline"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/pos/holds", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `hold-park-${session.session_id}-${Date.now()}`,
        },
        body: JSON.stringify({
          location_id: activeLocationId,
          register_id: session.register_id,
          cart: { lines: cart },
          note: holdNote.trim() || undefined,
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (res.status === 404) {
        setHoldsEnabled(false);
        throw new Error("Server holds disabled (POS_HOLDS_ENABLED)");
      }
      if (!res.ok) {
        throw new Error(
          (json as { error?: string }).error || `park_failed_${res.status}`,
        );
      }
      setCart([]);
      await clearParkedPosCart();
      setHoldNote("");
      setBanner("Cart parked on server (no stock held)");
      await loadServerHolds();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.park_failed"));
    } finally {
      setBusy(false);
    }
  };

  const resumeServerHold = async (hold: ServerHold) => {
    if (!session || !online) {
      setError(t("retailer_desktop.residual.text.resume_requires_open_session_and_network"));
      return;
    }
    if (cart.length > 0) {
      setError(t("retailer_desktop.residual.text.clear_or_park_current_cart_before_resuming_a_hold"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch(
        `/v1/retailer/pos/holds/${hold.hold_id}/resume`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ location_id: activeLocationId || hold.location_id }),
        },
      );
      const json = (await res.json().catch(() => ({}))) as ServerHold & {
        error?: string;
        code?: string;
      };
      if (!res.ok) {
        throw new Error(json.code || json.error || `resume_failed_${res.status}`);
      }
      // Rehydrate cart from hold snapshot — no stock mutation server-side.
      const raw = json.cart ?? hold.cart;
      let lines: CartLine[] = [];
      if (Array.isArray(raw)) {
        lines = raw as CartLine[];
      } else if (raw && typeof raw === "object" && Array.isArray((raw as { lines?: CartLine[] }).lines)) {
        lines = (raw as { lines: CartLine[] }).lines;
      }
      setCart(lines);
      setBanner(`Resumed hold ${hold.hold_id.slice(0, 8)}…`);
      await loadServerHolds();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.resume_failed"));
    } finally {
      setBusy(false);
    }
  };

  const voidServerHold = async (holdId: string) => {
    if (!online) return;
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch(`/v1/retailer/pos/holds/${holdId}/void`, {
        method: "POST",
      });
      if (!res.ok) {
        const json = await res.json().catch(() => ({}));
        throw new Error(
          (json as { error?: string }).error || `void_failed_${res.status}`,
        );
      }
      setBanner("Hold voided (no stock change)");
      await loadServerHolds();
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.void_hold_failed"));
    } finally {
      setBusy(false);
    }
  };

  const createRegister = async () => {
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/registers", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `reg-${Date.now()}`,
        },
        body: JSON.stringify({ label: newRegLabel }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error((json as { error?: string }).error || "create_failed");
      setBanner("Register created (POS + stock packs auto-enabled if needed)");
      await loadRegisters();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : t("retailer_desktop.residual.text.create_register_failed"));
    } finally {
      setBusy(false);
    }
  };

  const openSession = async () => {
    if (!registerId) return;
    if (!online) {
      setError(t("retailer_desktop.residual.text.open_session_requires_network_connect_then_open_till"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/pos/sessions/open", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `open-${registerId}-${Date.now()}`,
        },
        body: JSON.stringify({
          register_id: registerId,
          opening_float_minor: Number(floatMinor) || 0,
          currency: sessionPackCurrency(),
        }),
      });
      const json = await res.json();
      if (!res.ok) throw new Error((json as { error?: string }).error || "open_failed");
      const sess = json as Session;
      setSession(sess);
      const parked = await loadParkedPosCart(sess.session_id);
      setCart(parked?.lines ?? []);
      setBanner("Session open — ready to sell");
      setLastSale(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.open_failed"));
    } finally {
      setBusy(false);
    }
  };

  const closeSession = async () => {
    if (!session) return;
    if (pendingForSession > 0) {
      setError(
        `Sync ${pendingForSession} offline sale(s) before closing this session.`,
      );
      return;
    }
    if (!online) {
      setError(t("retailer_desktop.residual.text.close_session_requires_network"));
      return;
    }
    setBusy(true);
    try {
      const res = await apiFetch(
        `/v1/retailer/pos/sessions/${session.session_id}/close`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            closing_cash_minor: Number(closingCash) || 0,
          }),
        },
      );
      const json = await res.json();
      if (!res.ok) throw new Error((json as { error?: string }).error || "close_failed");
      setSession(null);
      setCart([]);
      await clearParkedPosCart();
      setBanner(
        `Session closed. Variance: ${formatMoney((json as Session & { variance_minor?: number }).variance_minor ?? 0)}`,
      );
    } catch (e) {
      setBanner(e instanceof Error ? e.message : t("retailer_desktop.residual.text.close_failed"));
    } finally {
      setBusy(false);
    }
  };

  const addLine = () => {
    const unit = Math.round(Number(price) * 100);
    const q = Math.max(1, Number(qty) || 1);
    if (!sku.trim() || unit < 0) return;
    setCart((c) => {
      const i = c.findIndex((l) => l.sku === sku.trim());
      if (i >= 0) {
        const next = [...c];
        next[i] = { ...next[i], qty: next[i].qty + q };
        return next;
      }
      return [
        ...c,
        {
          sku: sku.trim(),
          name: name.trim() || sku.trim(),
          qty: q,
          unit_price_minor: unit,
        },
      ];
    });
    setSku("");
    setName("");
    setQty("1");
  };

  const completeSale = async () => {
    if (!session || cart.length === 0) return;
    if (!online && tender === "CARD") {
      setError(t("retailer_desktop.residual.text.card_sales_require_network_use_cash_offline"));
      return;
    }
    setBusy(true);
    setBanner(null);
    setError(null);

    const clientSaleId = newClientSaleId();
    const payload = {
      session_id: session.session_id,
      stock_bin: "FLOOR",
      origin: online ? "online" : "offline",
      client_sale_id: clientSaleId,
      client_created_at: new Date().toISOString(),
      lines: cart.map((l) => ({
        sku: l.sku,
        name: l.name,
        qty: l.qty,
        unit_price_minor: l.unit_price_minor,
      })),
      tenders: [{ method: tender, amount_minor: totalMinor }],
    };

    try {
      if (!online) {
        // Offline cash only
        const receipt = provisionalReceipt(receiptSeq);
        setReceiptSeq((n) => n + 1);
        await enqueueOfflinePosSale({
          clientSaleId,
          clientReceipt: receipt,
          sessionId: session.session_id,
          payload: { ...payload, origin: "offline", tenders: [{ method: "CASH", amount_minor: totalMinor }] },
        });
        setLastSale({
          sale_id: clientSaleId,
          receipt_number: receipt,
          total_minor: totalMinor,
          status: "COMPLETED",
          lines: cart,
          origin: "offline",
          client_sale_id: clientSaleId,
        });
        setCart([]);
        await clearParkedPosCart();
        setBanner(`Offline sale ${receipt} · ${formatMoney(totalMinor)} · will sync`);
        await refreshPending();
        return;
      }

      const res = await apiFetch("/v1/retailer/pos/sales", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": retailerPosSaleKey(clientSaleId),
        },
        body: JSON.stringify(payload),
      });
      const json = await res.json();
      if (!res.ok) {
        // Network-ish: queue as offline cash if cash tender
        if (res.status === 0 || res.status >= 500) {
          throw new Error((json as { error?: string }).error || `sale_failed_${res.status}`);
        }
        throw new Error(
          (json as { error?: string; sku?: string }).error
            ? `${(json as { error: string }).error}${(json as { sku?: string }).sku ? ` (${(json as { sku: string }).sku})` : ""}`
            : `sale_failed_${res.status}`,
        );
      }
      setLastSale(json as Sale);
      setCart([]);
      await clearParkedPosCart();
      setBanner(`Sale ${(json as Sale).receipt_number} · ${formatMoney((json as Sale).total_minor)}`);
    } catch (e) {
      // If online request failed due to network, queue cash sales
      const msg = e instanceof Error ? e.message : t("retailer_desktop.residual.text.sale_failed");
      if (
        tender === "CASH" &&
        (/failed to fetch|network|load failed|offline/i.test(msg) || !navigator.onLine)
      ) {
        try {
          const receipt = provisionalReceipt(receiptSeq);
          setReceiptSeq((n) => n + 1);
          await enqueueOfflinePosSale({
            clientSaleId,
            clientReceipt: receipt,
            sessionId: session.session_id,
            payload: { ...payload, origin: "offline", tenders: [{ method: "CASH", amount_minor: totalMinor }] },
          });
          setLastSale({
            sale_id: clientSaleId,
            receipt_number: receipt,
            total_minor: totalMinor,
            status: "COMPLETED",
            lines: cart,
            origin: "offline",
            client_sale_id: clientSaleId,
          });
          setCart([]);
          await clearParkedPosCart();
          setBanner(`Queued offline ${receipt} · ${formatMoney(totalMinor)}`);
          await refreshPending();
          return;
        } catch {
          /* fall through */
        }
      }
      setBanner(msg);
    } finally {
      setBusy(false);
    }
  };

  const syncNow = async () => {
    setBusy(true);
    try {
      const r = await flushPendingPosSales();
      setBanner(
        r.flushed > 0
          ? `Synced ${r.flushed} sale(s)${r.failed ? `, ${r.failed} failed` : ""}`
          : r.failed
            ? `${r.failed} sale(s) failed to sync`
            : "Nothing to sync",
      );
      await refreshPending();
    } finally {
      setBusy(false);
    }
  };

  const voidLast = async () => {
    if (!lastSale || lastSale.status === "VOIDED") return;
    if (lastSale.origin === "offline" || lastSale.sale_id === lastSale.client_sale_id) {
      setError(t("retailer_desktop.residual.text.void_requires_a_synced_server_sale_sync_first_or_discard_from_pe"));
      return;
    }
    if (!confirm("Void last sale and restock?")) return;
    setBusy(true);
    try {
      const res = await apiFetch(
        `/v1/retailer/pos/sales/${lastSale.sale_id}/void`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason: "cashier_void" }),
        },
      );
      const json = await res.json();
      if (!res.ok) throw new Error((json as { error?: string }).error || "void_failed");
      setLastSale(json as Sale);
      setBanner("Sale voided — stock restored");
    } catch (e) {
      setBanner(e instanceof Error ? e.message : t("retailer_desktop.residual.text.void_failed_manager_role_may_be_required"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title={t("retailer_desktop.pos.text.point_of_sale")}
      description={t("retailer_desktop.residual.text.cashier_mode_open_till_online_sell_cash_offline_when_needed_sync")}
    >
      <div className="mx-auto grid max-w-5xl gap-4 px-4 pb-16 pt-2 lg:grid-cols-2">
        {!online && (
          <div className="lg:col-span-2 flex items-center gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-800 dark:text-amber-200">
            <WifiOff className="h-4 w-4 shrink-0" />
            Offline · cash sales queue until reconnect. Card disabled. Open/close till needs network.
            {pendingForSession > 0 && (
              <span className="ml-auto font-medium">{pendingForSession} pending</span>
            )}
          </div>
        )}
        {banner && (
          <div className="lg:col-span-2 rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        {error && (
          <div className="lg:col-span-2 flex items-center gap-2 text-sm text-red-600">
            <AlertTriangle className="h-4 w-4" />
            {error}
          </div>
        )}

        {/* Session control */}
        <section className="rounded-xl border border-border bg-card p-4 space-y-3">
          <h2 className="font-semibold">{t("retailer_desktop.pos.text.till_session")}</h2>
          {registers.length === 0 ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">{t("retailer_desktop.pos.text.no_registers_yet")}</p>
              <input
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
                value={newRegLabel}
                onChange={(e) => setNewRegLabel(e.target.value)}
              />
              <button
                type="button"
                disabled={busy || !online}
                onClick={() => void createRegister()}
                className="rounded-lg bg-foreground px-3 py-2 text-sm text-background disabled:opacity-50"
              >
                Create register
              </button>
            </div>
          ) : (
            <>
              <label className="block text-sm">
                Register
                <select
                  className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                  value={registerId}
                  onChange={(e) => setRegisterId(e.target.value)}
                  disabled={!!session}
                >
                  {registers.map((r) => (
                    <option key={r.register_id} value={r.register_id}>
                      {r.label}
                    </option>
                  ))}
                </select>
              </label>
              {!session ? (
                <>
                  <label className="block text-sm">
                    Opening float (minor units)
                    <input
                      className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                      value={floatMinor}
                      onChange={(e) => setFloatMinor(e.target.value)}
                    />
                  </label>
                  <button
                    type="button"
                    disabled={busy || !registerId || !online}
                    onClick={() => void openSession()}
                    className="rounded-lg bg-foreground px-3 py-2 text-sm text-background disabled:opacity-50"
                  >
                    Open session
                  </button>
                </>
              ) : (
                <>
                  <p className="text-sm">
                    Open · {session.session_id.slice(0, 12)}… · float{" "}
                    {formatMoney(session.opening_float_minor, session.currency)}
                  </p>
                  <label className="block text-sm">
                    Closing cash counted (minor)
                    <input
                      className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                      value={closingCash}
                      onChange={(e) => setClosingCash(e.target.value)}
                    />
                  </label>
                  <button
                    type="button"
                    disabled={busy || !online || pendingForSession > 0}
                    onClick={() => void closeSession()}
                    className="rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
                  >
                    {pendingForSession > 0
                      ? `Close blocked (${pendingForSession} pending)`
                      : "Close session"}
                  </button>
                </>
              )}
            </>
          )}

          {pending.length > 0 && (
            <div className="mt-3 space-y-2 border-t border-border pt-3">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium">Offline queue ({pending.length})</h3>
                <button
                  type="button"
                  disabled={busy || !online}
                  onClick={() => void syncNow()}
                  className="inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs disabled:opacity-50"
                >
                  <CloudUpload className="h-3 w-3" /> Sync now
                </button>
              </div>
              <ul className="max-h-40 space-y-1 overflow-y-auto text-xs">
                {pending.map((p) => (
                  <li
                    key={p.id}
                    className={`rounded border px-2 py-1 ${
                      p.status === "FAILED"
                        ? "border-red-500/40 text-red-600"
                        : "border-border text-muted-foreground"
                    }`}
                  >
                    {p.clientReceipt} · {p.status}
                    {p.lastError ? ` · ${p.lastError}` : ""}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>

        {/* Cart */}
        <section className="rounded-xl border border-border bg-card p-4 space-y-3">
          <h2 className="font-semibold flex items-center gap-2">
            <ShoppingCart className="h-4 w-4" /> Cart
            {cart.length > 0 && (
              <span className="text-xs font-normal text-muted-foreground">(auto-parked)</span>
            )}
          </h2>
          {!session && (
            <p className="text-sm text-muted-foreground">{t("retailer_desktop.pos.text.open_a_session_to_sell")}</p>
          )}
          {session && (
            <>
              <div className="grid grid-cols-2 gap-2">
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder={t("retailer_desktop.pos.text.sku_barcode")}
                  value={sku}
                  onChange={(e) => setSku(e.target.value)}
                />
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder={t("retailer_desktop.pos.text.name")}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder={t("retailer_desktop.pos.text.qty")}
                  value={qty}
                  onChange={(e) => setQty(e.target.value)}
                />
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder={t("retailer_desktop.pos.text.price_major_e_g_150_00")}
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                />
              </div>
              <button
                type="button"
                onClick={addLine}
                className="rounded-lg border border-border px-3 py-2 text-sm"
              >
                Add line
              </button>
              <ul className="divide-y divide-border text-sm">
                {cart.map((l) => (
                  <li
                    key={l.sku}
                    className="flex items-center justify-between py-2 gap-2"
                  >
                    <span>
                      {l.name} × {l.qty} @ {formatMoney(l.unit_price_minor)}
                    </span>
                    <button
                      type="button"
                      className="text-muted-foreground hover:text-foreground"
                      onClick={() =>
                        setCart((c) => c.filter((x) => x.sku !== l.sku))
                      }
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </li>
                ))}
              </ul>
              <div className="flex items-center justify-between font-semibold text-lg">
                <span>{t("retailer_desktop.pos.text.total")}</span>
                <span>{formatMoney(totalMinor)}</span>
              </div>
              {holdsEnabled && (
                <div className="space-y-2 border-t border-border pt-3">
                  <label className="block text-sm text-muted-foreground">
                    Park note (optional)
                    <input
                      className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
                      value={holdNote}
                      onChange={(e) => setHoldNote(e.target.value)}
                      placeholder={t("retailer_desktop.pos.text.customer_returns")}
                    />
                  </label>
                  <button
                    type="button"
                    disabled={busy || cart.length === 0 || !online}
                    onClick={() => void parkServerHold()}
                    className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
                  >
                    <PauseCircle className="h-4 w-4" />
                    Park cart on server
                  </button>
                  <p className="text-xs text-muted-foreground">
                    Does not reserve stock. Resume only at this location. Expires in 24h.
                  </p>
                </div>
              )}
              <div className="flex gap-2">
                <button
                  type="button"
                  className={`flex-1 rounded-lg border px-3 py-2 text-sm flex items-center justify-center gap-1 ${
                    tender === "CASH" ? "border-foreground" : "border-border"
                  }`}
                  onClick={() => setTender("CASH")}
                >
                  <Banknote className="h-4 w-4" /> Cash
                </button>
                <button
                  type="button"
                  disabled={!online}
                  className={`flex-1 rounded-lg border px-3 py-2 text-sm flex items-center justify-center gap-1 disabled:opacity-40 ${
                    tender === "CARD" ? "border-foreground" : "border-border"
                  }`}
                  onClick={() => setTender("CARD")}
                >
                  <CreditCard className="h-4 w-4" /> Card
                </button>
              </div>
              <button
                type="button"
                disabled={busy || cart.length === 0}
                onClick={() => void completeSale()}
                className="w-full rounded-lg bg-foreground py-3 text-background font-medium disabled:opacity-50"
              >
                {busy ? (
                  <Loader2 className="inline h-4 w-4 animate-spin" />
                ) : online ? (
                  `Complete sale (${tender})`
                ) : (
                  "Complete cash sale offline"
                )}
              </button>
              {lastSale && lastSale.status === "COMPLETED" && lastSale.origin !== "offline" && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void voidLast()}
                  className="w-full rounded-lg border border-red-500/40 py-2 text-sm text-red-600"
                >
                  Void last sale {lastSale.receipt_number}
                </button>
              )}
            </>
          )}
        </section>

        {/* C3.2 server holds list (same location) */}
        {holdsEnabled && session && (
          <section className="lg:col-span-2 rounded-xl border border-border bg-card p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="font-semibold">{t("retailer_desktop.pos.text.parked_carts_server")}</h2>
              <button
                type="button"
                disabled={busy || !online}
                onClick={() => void loadServerHolds()}
                className="text-xs text-muted-foreground underline disabled:opacity-50"
              >
                Refresh
              </button>
            </div>
            {serverHolds.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No held carts at this location.
              </p>
            ) : (
              <ul className="divide-y divide-border text-sm">
                {serverHolds.map((h) => {
                  const cartObj = h.cart;
                  let lineCount = 0;
                  if (Array.isArray(cartObj)) lineCount = cartObj.length;
                  else if (cartObj && Array.isArray(cartObj.lines))
                    lineCount = cartObj.lines.length;
                  return (
                    <li
                      key={h.hold_id}
                      className="flex flex-wrap items-center justify-between gap-2 py-3"
                    >
                      <div>
                        <p className="font-medium">
                          {h.hold_id.slice(0, 10)}… · {lineCount} line(s)
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {h.note ? `${h.note} · ` : ""}
                          {h.expires_at
                            ? `expires ${new Date(h.expires_at).toLocaleString()}`
                            : "HELD"}
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          disabled={busy || !online || cart.length > 0}
                          onClick={() => void resumeServerHold(h)}
                          className="inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs disabled:opacity-50"
                          title={
                            cart.length > 0
                              ? "Clear or park current cart first"
                              : "Resume into current cart"
                          }
                        >
                          <PlayCircle className="h-3.5 w-3.5" /> Resume
                        </button>
                        <button
                          type="button"
                          disabled={busy || !online}
                          onClick={() => void voidServerHold(h.hold_id)}
                          className="inline-flex items-center gap-1 rounded-lg border border-red-500/30 px-2 py-1 text-xs text-red-600 disabled:opacity-50"
                        >
                          <XCircle className="h-3.5 w-3.5" /> Void
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
        )}
      </div>
    </PageChrome>
  );
}
