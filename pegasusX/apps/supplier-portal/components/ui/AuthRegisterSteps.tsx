"use client";

import type { Dispatch, SetStateAction } from "react";
import type { INNLookupResponse } from "@pegasusx/types";
import { FormAlert, PortalField } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";

export { dialCodeForCountry };

export type AuthIdentityStep = {
  countryCode: string;
  phoneLocal: string;
};

export type AuthVerificationStep = {
  otpCode: string;
  idToken: string;
};

export type AuthProfileStep = {
  taxId: string;
  legalName: string;
  contactName: string;
  email: string;
  password: string;
};

type Errors = Record<string, string>;

export function AuthRegisterIdentityStep({
  identity,
  setIdentity,
  errors,
  dialCode,
}: {
  identity: AuthIdentityStep;
  setIdentity: Dispatch<SetStateAction<AuthIdentityStep>>;
  errors: Errors;
  dialCode: string;
}) {
  return (
    <div className="grid gap-4">
      <div className="grid grid-cols-[160px,1fr] gap-3">
        <AuthField id="countryCode" label="Country" error={errors.countryCode}>
          <select
            id="countryCode"
            className="md-input-outlined"
            value={identity.countryCode}
            onChange={(e) => setIdentity((s) => ({ ...s, countryCode: e.target.value }))}
          >
            {AUTH_COUNTRIES.map((c) => (
              <option key={c.code} value={c.code}>
                {c.name}
              </option>
            ))}
          </select>
        </AuthField>
        <AuthField
          id="phoneLocal"
          label="Phone"
          error={errors.phoneLocal}
          hint={`Will be sent as ${dialCode}${identity.phoneLocal || "…"}`}
        >
          <div className="flex">
            <span
              className="inline-flex items-center px-3 border border-r-0 rounded-l text-sm"
              style={{
                borderColor: "var(--color-md-outline)",
                background: "var(--color-md-surface-container-high)",
              }}
            >
              {dialCode}
            </span>
            <input
              id="phoneLocal"
              inputMode="numeric"
              className="md-input-outlined"
              style={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0 }}
              value={identity.phoneLocal}
              aria-invalid={!!errors.phoneLocal}
              onChange={(e) =>
                setIdentity((s) => ({ ...s, phoneLocal: e.target.value.replace(/\D/g, "") }))
              }
            />
          </div>
        </AuthField>
      </div>
      <div id="recaptcha-container" />
    </div>
  );
}

export function AuthRegisterVerificationStep({
  verification,
  setVerification,
  errors,
}: {
  verification: AuthVerificationStep;
  setVerification: Dispatch<SetStateAction<AuthVerificationStep>>;
  errors: Errors;
}) {
  return (
    <div className="grid gap-4">
      <AuthField
        id="otpCode"
        label="Verification code"
        error={errors.otpCode}
        hint="Enter the 6-digit code sent via SMS."
      >
        <input
          id="otpCode"
          inputMode="numeric"
          className="md-input-outlined tracking-widest text-lg font-mono text-center"
          value={verification.otpCode}
          maxLength={6}
          aria-invalid={!!errors.otpCode}
          onChange={(e) =>
            setVerification((s) => ({ ...s, otpCode: e.target.value.replace(/\D/g, "") }))
          }
        />
      </AuthField>
    </div>
  );
}

export function AuthRegisterProfileStep({
  profile,
  setProfile,
  errors,
  onVerifyInn,
  verifyingInn,
  innLookup,
  innError,
  taxIdLabel = "Company INN / STIR (9 digits)",
  taxIdPlaceholder = "Enter 9-digit INN",
  verifyButtonLabel = "Verify INN",
  verifyingLabel = "Verifying…",
  verifiedBadgeLabel = "Verified entity",
  existingCompanyHint,
}: {
  profile: AuthProfileStep;
  setProfile: Dispatch<SetStateAction<AuthProfileStep>>;
  errors: Errors;
  onVerifyInn?: () => void;
  verifyingInn?: boolean;
  innLookup?: INNLookupResponse | null;
  innError?: string | null;
  taxIdLabel?: string;
  taxIdPlaceholder?: string;
  verifyButtonLabel?: string;
  verifyingLabel?: string;
  verifiedBadgeLabel?: string;
  existingCompanyHint?: string | null;
}) {
  return (
    <div className="grid gap-4">
      <AuthField id="taxId" label={taxIdLabel} error={errors.taxId || innError || undefined}>
        <div className="flex gap-2 items-center">
          <input
            id="taxId"
            inputMode="numeric"
            autoComplete="off"
            placeholder={taxIdPlaceholder}
            className="md-input-outlined flex-1"
            value={profile.taxId}
            aria-invalid={!!(errors.taxId || innError)}
            onChange={(e) => setProfile((s) => ({ ...s, taxId: e.target.value }))}
          />
          {onVerifyInn ? (
            <button
              type="button"
              disabled={verifyingInn}
              onClick={onVerifyInn}
              className="portal-btn portal-btn--outline shrink-0"
            >
              {verifyingInn ? verifyingLabel : verifyButtonLabel}
            </button>
          ) : null}
        </div>
      </AuthField>
      {innLookup?.status === "FOUND" ? (
        <FormAlert variant="info">
          <span className="flex flex-col gap-1 text-xs">
            <span className="font-semibold">
              {verifiedBadgeLabel}: {innLookup.legal_name} ({innLookup.normalized_inn})
            </span>
            {innLookup.existing_join_eligible && existingCompanyHint ? (
              <span>{existingCompanyHint}</span>
            ) : null}
          </span>
        </FormAlert>
      ) : null}
      <AuthField id="legalName" label="Legal company name" error={errors.legalName}>
        <input
          id="legalName"
          className="md-input-outlined"
          value={profile.legalName}
          aria-invalid={!!errors.legalName}
          onChange={(e) => setProfile((s) => ({ ...s, legalName: e.target.value }))}
        />
      </AuthField>
      <AuthField id="contactName" label="Primary contact name" error={errors.contactName}>
        <input
          id="contactName"
          className="md-input-outlined"
          value={profile.contactName}
          aria-invalid={!!errors.contactName}
          onChange={(e) => setProfile((s) => ({ ...s, contactName: e.target.value }))}
        />
      </AuthField>
      <AuthField id="email" label="Work email" error={errors.email}>
        <input
          id="email"
          type="email"
          autoComplete="email"
          className="md-input-outlined"
          value={profile.email}
          aria-invalid={!!errors.email}
          onChange={(e) => setProfile((s) => ({ ...s, email: e.target.value }))}
        />
      </AuthField>
      <AuthField
        id="password"
        label="Password / Access PIN"
        error={errors.password}
        hint="Minimum 6 characters (6-digit PIN or password for sign-in)."
      >
        <input
          id="password"
          type="password"
          autoComplete="new-password"
          className="md-input-outlined"
          value={profile.password}
          aria-invalid={!!errors.password}
          onChange={(e) => setProfile((s) => ({ ...s, password: e.target.value }))}
        />
      </AuthField>
    </div>
  );
}

function AuthField({
  id,
  label,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  error?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <PortalField id={id} label={label} error={error} hint={hint}>
      {children}
    </PortalField>
  );
}
