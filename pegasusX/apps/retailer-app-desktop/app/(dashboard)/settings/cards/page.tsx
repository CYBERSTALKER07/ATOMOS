"use client";

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ArrowLeft, CreditCard, Loader2, Plus } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "../../../../lib/auth";
import { deactivateCard, setDefaultCard } from "../../../../lib/api";

type SavedCard = {
  id: string;
  pan?: string;
  pan_mask?: string;
  type?: string;
  is_default?: boolean;
};

export default function SavedCardsPage() {
  return (
    <Suspense
      fallback={
        <div className="max-w-2xl mx-auto p-6 flex items-center justify-center py-20">
          <Loader2 className="animate-spin text-[var(--desk-accent)]" size={28} />
        </div>
      }
    >
      <SavedCardsPageContent />
    </Suspense>
  );
}

function SavedCardsPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const returnTo = searchParams.get("return_to");
  const orderId = searchParams.get("order_id") ?? "";
  const sessionId = searchParams.get("session_id") ?? "";

  const [cards, setCards] = useState<SavedCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pendingToken, setPendingToken] = useState<string | null>(null);
  const [otpCode, setOtpCode] = useState("");
  const [adding, setAdding] = useState(false);
  const [cardActionId, setCardActionId] = useState<string | null>(null);

  const isDeliveryPaymentReturn = returnTo === "delivery_payment";

  const returnPath = useMemo(() => {
    if (!isDeliveryPaymentReturn) return "/settings";
    const params = new URLSearchParams({ resume_delivery_payment: "1" });
    if (orderId) params.set("order_id", orderId);
    if (sessionId) params.set("session_id", sessionId);
    return `/dock?${params.toString()}`;
  }, [isDeliveryPaymentReturn, orderId, sessionId]);

  const loadCards = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/cards");
      if (!res.ok) {
        throw new Error("Could not load saved cards");
      }
      const data = (await res.json()) as { cards?: SavedCard[] };
      setCards(Array.isArray(data.cards) ? data.cards : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load saved cards");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadCards();
  }, [loadCards]);

  const returnToPayment = useCallback(() => {
    router.push(returnPath);
  }, [router, returnPath]);

  const initiateCard = async () => {
    setAdding(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/card/initiate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (!res.ok) {
        throw new Error("Could not start card tokenization");
      }
      const data = (await res.json()) as { card_token?: string };
      if (!data.card_token) {
        throw new Error("Card tokenization session missing");
      }
      setPendingToken(data.card_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start card tokenization");
    } finally {
      setAdding(false);
    }
  };

  const confirmCard = async () => {
    if (!pendingToken || !otpCode.trim()) return;
    setAdding(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/card/confirm", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          card_token: pendingToken,
          otp_code: otpCode.trim(),
        }),
      });
      if (!res.ok) {
        throw new Error("Could not confirm card");
      }
      setPendingToken(null);
      setOtpCode("");
      await loadCards();
      if (isDeliveryPaymentReturn) {
        returnToPayment();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not confirm card");
    } finally {
      setAdding(false);
    }
  };

  const handleDeactivate = async (tokenId: string) => {
    setCardActionId(tokenId);
    setError(null);
    try {
      const res = await deactivateCard(tokenId);
      if (!res.ok) {
        throw new Error("Could not deactivate card");
      }
      await loadCards();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not deactivate card");
    } finally {
      setCardActionId(null);
    }
  };

  const handleSetDefault = async (tokenId: string) => {
    setCardActionId(tokenId);
    setError(null);
    try {
      const res = await setDefaultCard(tokenId);
      if (!res.ok) {
        throw new Error("Could not set default card");
      }
      await loadCards();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not set default card");
    } finally {
      setCardActionId(null);
    }
  };

  return (
    <PageChrome
      icon="settings"
      title="Saved Cards"
      description={
        isDeliveryPaymentReturn
          ? `Add a card, then return to complete delivery payment for order #${orderId.slice(-8) || "pending"}.`
          : "Manage tokenized payment methods for checkout and delivery."
      }
      loading={loading}
      skeletonVariant="form"
      actions={
        <button
          type="button"
          onClick={() => router.push(isDeliveryPaymentReturn ? returnPath : "/settings")}
          className="portal-btn portal-btn--ghost desk-icon-btn"
          aria-label="Back"
        >
          <ArrowLeft size={18} />
        </button>
      }
    >
    <div className="max-w-2xl mx-auto space-y-6">

      {isDeliveryPaymentReturn && (
        <div className="rounded-2xl border border-[var(--desk-accent)]/30 bg-[var(--desk-accent-soft)] p-4 flex items-center justify-between gap-4">
          <p className="text-sm font-semibold text-[var(--desk-text-primary)]">
            Delivery payment is paused while you add a card.
          </p>
          <button
            type="button"
            onClick={returnToPayment}
            className="px-4 h-10 rounded-xl bg-[var(--desk-accent)] text-white text-sm font-bold"
          >
            Return to payment
          </button>
        </div>
      )}

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-semibold text-red-700">
          {error}
        </div>
      )}

      <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-3">
        {loading ? (
          <div className="flex items-center justify-center py-10">
            <Loader2 className="animate-spin text-[var(--desk-accent)]" size={28} />
          </div>
        ) : cards.length === 0 ? (
          <div className="flex flex-col items-center gap-3 py-10 text-center">
            <CreditCard size={32} className="text-[var(--desk-text-tertiary)]" />
            <p className="text-sm font-semibold text-[var(--desk-text-secondary)]">
              No saved cards yet.
            </p>
          </div>
        ) : (
          cards.map((card) => (
            <div
              key={card.id}
              className="flex items-center justify-between rounded-xl border border-[var(--desk-border)] px-4 py-3 gap-3"
            >
              <div>
                <p className="font-bold text-[var(--desk-text-primary)]">
                  {card.pan || card.pan_mask || "Card"}
                </p>
                <p className="text-xs text-[var(--desk-text-tertiary)] uppercase">
                  {card.type || "CARD"}
                  {card.is_default ? " · Default" : ""}
                </p>
              </div>
              <div className="flex items-center gap-2">
                {!card.is_default && (
                  <button
                    type="button"
                    disabled={cardActionId === card.id}
                    onClick={() => void handleSetDefault(card.id)}
                    className="px-3 h-9 rounded-lg border border-[var(--desk-border)] text-xs font-bold"
                  >
                    Set default
                  </button>
                )}
                <button
                  type="button"
                  disabled={cardActionId === card.id}
                  onClick={() => void handleDeactivate(card.id)}
                  className="px-3 h-9 rounded-lg border border-red-200 text-xs font-bold text-red-700"
                >
                  Remove
                </button>
              </div>
            </div>
          ))
        )}
      </div>

      {!pendingToken ? (
        <button
          type="button"
          onClick={() => void initiateCard()}
          disabled={adding}
          className="flex items-center justify-center gap-2 w-full h-12 rounded-2xl bg-[var(--desk-accent)] text-white font-bold disabled:opacity-60"
        >
          {adding ? <Loader2 size={18} className="animate-spin" /> : <Plus size={18} />}
          Add payment method
        </button>
      ) : (
        <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-4 space-y-3">
          <p className="text-sm font-semibold text-[var(--desk-text-primary)]">
            Enter OTP to confirm card tokenization
          </p>
          <input
            value={otpCode}
            onChange={(e) => setOtpCode(e.target.value)}
            inputMode="numeric"
            className="w-full h-11 rounded-xl border border-[var(--desk-border)] px-3"
            placeholder="OTP code"
          />
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => {
                setPendingToken(null);
                setOtpCode("");
              }}
              className="flex-1 h-11 rounded-xl border border-[var(--desk-border)] font-bold"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void confirmCard()}
              disabled={adding || !otpCode.trim()}
              className="flex-1 h-11 rounded-xl bg-[var(--desk-accent)] text-white font-bold disabled:opacity-60"
            >
              {adding ? "Confirming..." : "Confirm"}
            </button>
          </div>
        </div>
      )}
    </div>
    </PageChrome>
  );
}
