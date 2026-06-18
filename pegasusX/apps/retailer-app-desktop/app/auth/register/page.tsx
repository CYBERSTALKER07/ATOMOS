"use client";

import { useState, useMemo } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@heroui/react";
import { ShieldCheck, Phone, KeyRound, Loader2, ChevronRight, AlertTriangle, User, Mail, MapPin } from "lucide-react";
import Link from "next/link";
import { storeToken } from "../../../lib/bridge";
import { LocationPicker } from "../../../components/LocationPicker";
import {
  COUNTRIES,
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type StepId,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
  normalizeReceivingWindow,
} from "./wizard-state";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

export default function RetailerRegisterPage() {
  const router = useRouter();
  const [state, setState] = useState<WizardState>(INITIAL_STATE);
  const [submitting, setSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState<string | null>(null);

  const stepIndex = STEP_ORDER.indexOf(state.step);
  const dialCode = useMemo(
    () => COUNTRIES.find((c) => c.code === state.identity.countryCode)?.dialCode ?? "",
    [state.identity.countryCode],
  );

  function validateCurrent(): Record<string, string> {
    switch (state.step) {
      case "identity":    return validateIdentity(state.identity);
      case "verification":return validateVerification(state.verification);
      case "profile":     return validateProfile(state.profile);
    }
  }

  function next() {
    const e = validateCurrent();
    if (Object.keys(e).length > 0) {
      setErrors(e);
      return;
    }
    setErrors({});
    
    // TODO: If step === 'identity', trigger Firebase Recaptcha / OTP before moving to next step
    
    if (stepIndex < STEP_ORDER.length - 1) {
      setState((s) => ({ ...s, step: STEP_ORDER[stepIndex + 1] as StepId }));
    }
  }

  function back() {
    setErrors({});
    if (stepIndex > 0) {
      setState((s) => ({ ...s, step: STEP_ORDER[stepIndex - 1] as StepId }));
    }
  }

  async function submit(e?: React.FormEvent) {
    if (e) e.preventDefault();
    const errs = validateCurrent();
    if (Object.keys(errs).length > 0) {
      setErrors(errs);
      return;
    }
    setErrors({});
    setSubmitting(true);
    setSubmitError(null);
    try {
      const phone = `${dialCode}${state.identity.phoneLocal}`;
      const lat = Number.parseFloat(state.profile.latitude.trim());
      const lng = Number.parseFloat(state.profile.longitude.trim());
      const payload = {
        phone_number: phone,
        password: state.verification.otpCode,
        store_name: state.profile.legalName.trim(),
        owner_name: state.profile.contactName.trim(),
        delivery_address: state.profile.deliveryAddress.trim(),
        place_id: state.profile.placeId.trim() || undefined,
        latitude: lat,
        longitude: lng,
        receiving_window_open: normalizeReceivingWindow(state.profile.receivingWindowOpen),
        receiving_window_close: normalizeReceivingWindow(state.profile.receivingWindowClose),
      };
      
      const res = await fetch(`${API}/v1/auth/retailer/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => null);
        throw new Error(errorData?.message || "Registration failed");
      }
      
      const data = await res.json();
      if (!data?.token) throw new Error("Registration response is missing token");

      document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      if (data.refresh_token) {
        document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
      }
      await storeToken(data.token, data.refresh_token || "");
      if (data.user)
        localStorage.setItem("retailer_profile", JSON.stringify(data.user));

      router.replace(data.is_configured ? "/dashboard" : "/setup");
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setSubmitting(false);
    }
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
              <ShieldCheck size={32} />
            </div>
            <h1 className="md-typescale-display-small font-bold text-[var(--desk-text-primary)] tracking-tight">
              Join Pegasus
            </h1>
            <p className="mt-2 md-typescale-body-large text-[var(--desk-text-secondary)]">
              {STEP_LABELS[state.step]}
            </p>
          </div>

          <div className="mb-8 flex gap-2 justify-center">
            {STEP_ORDER.map((stepId, idx) => (
              <div 
                key={stepId} 
                className={`h-1.5 rounded-full flex-1 transition-colors duration-300 ${
                  idx <= stepIndex ? "bg-[var(--desk-accent)]" : "bg-[var(--desk-border)]"
                }`}
              />
            ))}
          </div>

          <AnimatePresence mode="wait">
            <motion.div
              key={state.step}
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              transition={{ duration: 0.2 }}
            >
              {state.step === "identity" && (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Country
                    </label>
                    <div className="relative group">
                      <MapPin className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <select
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold appearance-none"
                        value={state.identity.countryCode}
                        onChange={(e) => setState((s) => ({ ...s, identity: { ...s.identity, countryCode: e.target.value } }))}
                      >
                        {COUNTRIES.map((c) => (
                          <option key={c.code} value={c.code}>{c.name}</option>
                        ))}
                      </select>
                    </div>
                    {errors.countryCode && <p className="text-red-500 text-xs mt-1">{errors.countryCode}</p>}
                  </div>
                  
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Protocol Handle
                    </label>
                    <div className="relative group flex bg-[var(--desk-canvas)] rounded-xl border border-transparent focus-within:border-[var(--desk-accent)] focus-within:ring-2 focus-within:ring-[var(--desk-accent-soft)] transition-all overflow-hidden">
                      <div className="flex items-center pl-4 pr-3 bg-[var(--desk-surface-subtle)] border-r border-[var(--desk-border)]">
                        <Phone className="text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors mr-2" size={18} />
                        <span className="font-bold text-[var(--desk-text-secondary)]">{dialCode}</span>
                      </div>
                      <input
                        type="tel"
                        inputMode="numeric"
                        className="flex-1 h-12 px-4 bg-transparent outline-none md-typescale-body-medium font-bold"
                        value={state.identity.phoneLocal}
                        onChange={(e) => setState((s) => ({ ...s, identity: { ...s.identity, phoneLocal: e.target.value.replace(/\D/g, "") } }))}
                      />
                    </div>
                    {errors.phoneLocal && <p className="text-red-500 text-xs mt-1">{errors.phoneLocal}</p>}
                  </div>
                  <div id="recaptcha-container"></div>
                </div>
              )}

              {state.step === "verification" && (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Verification Code
                    </label>
                    <div className="relative group">
                      <KeyRound className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="text"
                        inputMode="numeric"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold tracking-[0.5em] text-center"
                        maxLength={6}
                        value={state.verification.otpCode}
                        onChange={(e) => setState((s) => ({ ...s, verification: { ...s.verification, otpCode: e.target.value.replace(/\D/g, "") } }))}
                      />
                    </div>
                    {errors.otpCode && <p className="text-red-500 text-xs mt-1">{errors.otpCode}</p>}
                  </div>
                </div>
              )}

              {state.step === "profile" && (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Legal Company Name
                    </label>
                    <div className="relative group">
                      <ShieldCheck className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="text"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.profile.legalName}
                        onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, legalName: e.target.value } }))}
                      />
                    </div>
                    {errors.legalName && <p className="text-red-500 text-xs mt-1">{errors.legalName}</p>}
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Primary Contact Name
                    </label>
                    <div className="relative group">
                      <User className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="text"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.profile.contactName}
                        onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, contactName: e.target.value } }))}
                      />
                    </div>
                    {errors.contactName && <p className="text-red-500 text-xs mt-1">{errors.contactName}</p>}
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                      Work Email
                    </label>
                    <div className="relative group">
                      <Mail className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors" size={18} />
                      <input
                        type="email"
                        className="w-full h-12 pl-12 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.profile.email}
                        onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, email: e.target.value } }))}
                      />
                    </div>
                    {errors.email && <p className="text-red-500 text-xs mt-1">{errors.email}</p>}
                  </div>

                  <div className="space-y-1.5">
                    <LocationPicker
                      label="Store address"
                      value={{
                        address: state.profile.deliveryAddress,
                        lat: state.profile.latitude,
                        lng: state.profile.longitude,
                        place_id: state.profile.placeId || undefined,
                      }}
                      onChange={(next) =>
                        setState((s) => ({
                          ...s,
                          profile: {
                            ...s.profile,
                            deliveryAddress: next.address,
                            placeId: next.place_id ?? "",
                            latitude: next.lat,
                            longitude: next.lng,
                          },
                        }))
                      }
                    />
                    {errors.deliveryAddress && <p className="text-red-500 text-xs mt-1">{errors.deliveryAddress}</p>}
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                        Receiving window open
                      </label>
                      <input
                        type="text"
                        placeholder="09:00"
                        className="w-full h-12 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.profile.receivingWindowOpen}
                        onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, receivingWindowOpen: e.target.value } }))}
                      />
                      {errors.receivingWindowOpen && <p className="text-red-500 text-xs mt-1">{errors.receivingWindowOpen}</p>}
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-[10px] font-black uppercase tracking-[0.2em] text-[var(--desk-text-tertiary)] pl-1">
                        Receiving window close
                      </label>
                      <input
                        type="text"
                        placeholder="18:00"
                        className="w-full h-12 px-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] border border-transparent focus:border-[var(--desk-accent)] transition-all md-typescale-body-medium font-bold"
                        value={state.profile.receivingWindowClose}
                        onChange={(e) => setState((s) => ({ ...s, profile: { ...s.profile, receivingWindowClose: e.target.value } }))}
                      />
                      {errors.receivingWindowClose && <p className="text-red-500 text-xs mt-1">{errors.receivingWindowClose}</p>}
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
          {stepIndex > 0 && (
            <Button
              type="button"
              onPress={back}
              isDisabled={submitting}
              className="flex-1 h-14 bg-[var(--desk-canvas)] text-[var(--desk-text-primary)] font-bold rounded-2xl border border-[var(--desk-border)] transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30"
            >
              Back
            </Button>
          )}
          {stepIndex < STEP_ORDER.length - 1 ? (
            <Button
              type="button"
              onPress={next}
              className="flex-[2] h-14 bg-[var(--desk-text-primary)] text-white font-bold rounded-2xl shadow-xl transition-all hover:scale-[1.02] active:scale-95 flex items-center justify-center gap-2"
            >
              Continue <ChevronRight size={20} />
            </Button>
          ) : (
            <Button
              type="button"
              onPress={() => submit()}
              isDisabled={submitting}
              className="flex-[2] h-14 bg-[var(--desk-text-primary)] text-white font-bold rounded-2xl shadow-xl transition-all hover:scale-[1.02] active:scale-95 flex items-center justify-center gap-2 disabled:opacity-30"
            >
              {submitting ? <Loader2 size={20} className="animate-spin" /> : "Create Account"}
            </Button>
          )}
        </div>
        {stepIndex === 0 && (
          <div className="pb-6 bg-[var(--desk-surface-subtle)] text-center">
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Already a retailer?{" "}
              <Link href="/auth/login" className="font-bold underline hover:text-[var(--desk-text-primary)]">
                Sign in
              </Link>
            </p>
          </div>
        )}
      </motion.div>
    </div>
  );
}
