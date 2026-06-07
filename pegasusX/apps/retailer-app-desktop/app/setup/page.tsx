"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@heroui/react";
import { Building2, MapPin, Loader2, ChevronRight, AlertTriangle, FileText } from "lucide-react";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

type SetupStep = "tax" | "address";

interface SetupState {
  taxId: string;
  billingAddress: string;
  shippingAddress: string;
  city: string;
  postalCode: string;
}

const INITIAL_STATE: SetupState = {
  taxId: "",
  billingAddress: "",
  shippingAddress: "",
  city: "",
  postalCode: "",
};

export default function RetailerSetupPage() {
  const router = useRouter();
  const [step, setStep] = useState<SetupStep>("tax");
  const [state, setState] = useState<SetupState>(INITIAL_STATE);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validateTax(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.taxId.trim().length < 5) e.taxId = "Tax ID is required";
    return e;
  }

  function validateAddress(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.billingAddress.trim().length < 5) e.billingAddress = "Billing address is required";
    if (state.shippingAddress.trim().length < 5) e.shippingAddress = "Shipping address is required";
    if (state.city.trim().length < 2) e.city = "City is required";
    return e;
  }

  function handleNext() {
    const errs = validateTax();
    if (Object.keys(errs).length > 0) {
      setErrors(errs);
      return;
    }
    setErrors({});
    setStep("address");
  }

  async function handleSubmit(e?: React.FormEvent) {
    if (e) e.preventDefault();
    const errs = validateAddress();
    if (Object.keys(errs).length > 0) {
      setErrors(errs);
      return;
    }
    setErrors({});
    setSubmitting(true);
    setSubmitError(null);

    try {
      const token = getCookie("pegasus_retailer_jwt");
      const res = await fetch(`${API}/v1/retailer/setup`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(state),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => null);
        throw new Error(errorData?.message || "Setup failed");
      }
      
      // Force page reload to ensure middleware gets new token with is_configured=true
      window.location.href = "/dashboard";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Setup failed");
      setSubmitting(false);
    }
  }

  function getCookie(name: string) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift();
  }

  return (
    <div className="min-h-dvh w-full flex items-center justify-center p-6 bg-[var(--desk-canvas)]">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full max-w-md bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-[32px] shadow-2xl overflow-hidden flex flex-col"
      >
        <div className="p-10 flex-1">
          <div className="flex flex-col items-center text-center mb-8">
            <div className="w-16 h-16 rounded-2xl bg-[var(--desk-accent)] flex items-center justify-center text-white shadow-xl mb-6 rotate-3 hover:rotate-0 transition-transform cursor-default">
              <Building2 size={32} />
            </div>
            <h1 className="md-typescale-display-small font-bold text-[var(--desk-text-primary)] tracking-tight">
              Company Profile
            </h1>
            <p className="mt-2 md-typescale-body-large text-[var(--desk-text-secondary)]">
              {step === "tax" ? "Provide your business identity" : "Where should we deliver?"}
            </p>
          </div>

          <div className="mb-8 flex gap-2 justify-center">
            <div className="h-1.5 rounded-full flex-1 transition-colors duration-300 bg-[var(--desk-accent)]" />
            <div className={`h-1.5 rounded-full flex-1 transition-colors duration-300 ${step === "address" ? "bg-[var(--desk-accent)]" : "bg-[var(--desk-border)]"}`} />
          </div>

          <AnimatePresence mode="wait">
            <motion.div
              key={step}
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              transition={{ duration: 0.2 }}
            >
              {step === "tax" ? (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Tax ID / VAT
                    </label>
                    <div className="relative group">
                      <FileText className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="text"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.taxId}
                        onChange={(e) => setState({ ...state, taxId: e.target.value })}
                        placeholder="e.g. 123456789"
                      />
                    </div>
                    {errors.taxId && <p className="text-red-500 text-xs mt-1">{errors.taxId}</p>}
                  </div>
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Billing Address
                    </label>
                    <div className="relative group">
                      <MapPin className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="text"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.billingAddress}
                        onChange={(e) => setState({ ...state, billingAddress: e.target.value })}
                      />
                    </div>
                    {errors.billingAddress && <p className="text-red-500 text-xs mt-1">{errors.billingAddress}</p>}
                  </div>
                  
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Shipping Address (Default)
                    </label>
                    <div className="relative group">
                      <MapPin className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="text"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.shippingAddress}
                        onChange={(e) => setState({ ...state, shippingAddress: e.target.value })}
                      />
                    </div>
                    {errors.shippingAddress && <p className="text-red-500 text-xs mt-1">{errors.shippingAddress}</p>}
                  </div>

                  <div className="flex gap-4">
                    <div className="space-y-1.5 flex-[2]">
                      <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                        City
                      </label>
                      <input
                        type="text"
                        className="w-full h-12 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.city}
                        onChange={(e) => setState({ ...state, city: e.target.value })}
                      />
                      {errors.city && <p className="text-red-500 text-xs mt-1">{errors.city}</p>}
                    </div>
                    
                    <div className="space-y-1.5 flex-1">
                      <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                        Postal
                      </label>
                      <input
                        type="text"
                        className="w-full h-12 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.postalCode}
                        onChange={(e) => setState({ ...state, postalCode: e.target.value })}
                      />
                    </div>
                  </div>
                </div>
              )}
            </motion.div>
          </AnimatePresence>

          <AnimatePresence>
            {submitError && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                className="mt-6 p-3 rounded-xl bg-red-50 border border-red-100 text-red-600 text-xs font-bold text-center flex items-center justify-center gap-2"
              >
                <AlertTriangle size={14} />
                {submitError}
              </motion.div>
            )}
          </AnimatePresence>

        </div>
        
        <div className="p-6 bg-[var(--desk-surface-subtle)] border-t border-[var(--desk-border)] flex gap-3">
          {step === "address" && (
            <Button
              type="button"
              onPress={() => setStep("tax")}
              isDisabled={submitting}
              className="flex-1 h-14 bg-[var(--desk-canvas)] text-[var(--desk-text-primary)] font-bold rounded-2xl border border-[var(--desk-border)] transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30"
            >
              Back
            </Button>
          )}
          {step === "tax" ? (
            <Button
              type="button"
              onPress={handleNext}
              className="flex-[2] h-14 bg-[var(--desk-text-primary)] text-white font-bold rounded-2xl shadow-xl transition-all hover:scale-[1.02] active:scale-95 flex items-center justify-center gap-2"
            >
              Continue <ChevronRight size={20} />
            </Button>
          ) : (
            <Button
              type="button"
              onPress={() => handleSubmit()}
              isDisabled={submitting}
              className="flex-[2] h-14 bg-[var(--desk-text-primary)] text-white font-bold rounded-2xl shadow-xl transition-all hover:scale-[1.02] active:scale-95 flex items-center justify-center gap-2 disabled:opacity-30"
            >
              {submitting ? <Loader2 size={20} className="animate-spin" /> : "Complete Setup"}
            </Button>
          )}
        </div>
      </motion.div>
    </div>
  );
}
