"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Loader2,
  ShoppingCart,
  Trash2,
  CreditCard,
  Banknote,
  AlertTriangle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

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
};

function formatMoney(minor: number, currency = "UZS") {
  return `${(minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })} ${currency}`;
}

export default function POSPage() {
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

  const totalMinor = useMemo(
    () => cart.reduce((s, l) => s + l.qty * l.unit_price_minor, 0),
    [cart],
  );

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

  useEffect(() => {
    void loadRegisters();
  }, [loadRegisters]);

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
      setBanner(e instanceof Error ? e.message : "Create register failed");
    } finally {
      setBusy(false);
    }
  };

  const openSession = async () => {
    if (!registerId) return;
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
          currency: "UZS",
        }),
      });
      const json = await res.json();
      if (!res.ok) throw new Error((json as { error?: string }).error || "open_failed");
      setSession(json as Session);
      setBanner("Session open — ready to sell");
      setCart([]);
      setLastSale(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Open failed");
    } finally {
      setBusy(false);
    }
  };

  const closeSession = async () => {
    if (!session) return;
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
      setBanner(
        `Session closed. Variance: ${formatMoney((json as Session & { variance_minor?: number }).variance_minor ?? 0)}`,
      );
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Close failed");
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
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/pos/sales", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `sale-${Date.now()}`,
        },
        body: JSON.stringify({
          session_id: session.session_id,
          stock_bin: "FLOOR",
          lines: cart.map((l) => ({
            sku: l.sku,
            name: l.name,
            qty: l.qty,
            unit_price_minor: l.unit_price_minor,
          })),
          tenders: [{ method: tender, amount_minor: totalMinor }],
        }),
      });
      const json = await res.json();
      if (!res.ok) {
        throw new Error(
          (json as { error?: string; sku?: string }).error
            ? `${(json as { error: string }).error}${(json as { sku?: string }).sku ? ` (${(json as { sku: string }).sku})` : ""}`
            : `sale_failed_${res.status}`,
        );
      }
      setLastSale(json as Sale);
      setCart([]);
      setBanner(`Sale ${(json as Sale).receipt_number} · ${formatMoney((json as Sale).total_minor)}`);
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Sale failed");
    } finally {
      setBusy(false);
    }
  };

  const voidLast = async () => {
    if (!lastSale || lastSale.status === "VOIDED") return;
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
      setBanner(e instanceof Error ? e.message : "Void failed (manager role may be required)");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title="Point of sale"
      description="Cashier mode: open till, scan/add lines, cash or card, void with manager rights. Sells from FLOOR stock."
    >
      <div className="mx-auto grid max-w-5xl gap-4 px-4 pb-16 pt-2 lg:grid-cols-2">
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
          <h2 className="font-semibold">Till session</h2>
          {registers.length === 0 ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">No registers yet.</p>
              <input
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
                value={newRegLabel}
                onChange={(e) => setNewRegLabel(e.target.value)}
              />
              <button
                type="button"
                disabled={busy}
                onClick={() => void createRegister()}
                className="rounded-lg bg-foreground px-3 py-2 text-sm text-background"
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
                    Opening float (major units)
                    <input
                      className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                      value={floatMinor}
                      onChange={(e) => setFloatMinor(e.target.value)}
                      placeholder="0.00 → enter as tiyin/100 e.g. 100000 for 1000.00"
                    />
                  </label>
                  <p className="text-xs text-muted-foreground">
                    Enter float in minor units (integer tiyin/cents). Example: 100000 = 1000.00
                  </p>
                  <button
                    type="button"
                    disabled={busy || !registerId}
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
                    disabled={busy}
                    onClick={() => void closeSession()}
                    className="rounded-lg border border-border px-3 py-2 text-sm"
                  >
                    Close session
                  </button>
                </>
              )}
            </>
          )}
        </section>

        {/* Cart */}
        <section className="rounded-xl border border-border bg-card p-4 space-y-3">
          <h2 className="font-semibold flex items-center gap-2">
            <ShoppingCart className="h-4 w-4" /> Cart
          </h2>
          {!session && (
            <p className="text-sm text-muted-foreground">Open a session to sell.</p>
          )}
          {session && (
            <>
              <div className="grid grid-cols-2 gap-2">
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder="SKU / barcode"
                  value={sku}
                  onChange={(e) => setSku(e.target.value)}
                />
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder="Name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder="Qty"
                  value={qty}
                  onChange={(e) => setQty(e.target.value)}
                />
                <input
                  className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                  placeholder="Price (major e.g. 150.00)"
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
                <span>Total</span>
                <span>{formatMoney(totalMinor)}</span>
              </div>
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
                  className={`flex-1 rounded-lg border px-3 py-2 text-sm flex items-center justify-center gap-1 ${
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
                ) : (
                  `Complete sale (${tender})`
                )}
              </button>
              {lastSale && lastSale.status === "COMPLETED" && (
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
      </div>
    </PageChrome>
  );
}
