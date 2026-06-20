"use client";

import type { Dispatch, SetStateAction } from "react";
import { PortalField } from "../portal/PortalPrimitives";
import { AUTH_COUNTRIES } from "./countries";

export type AuthIdentityStep = {
  countryCode: string;
  phoneLocal: string;
};

export type AuthVerificationStep = {
  otpCode: string;
  idToken: string;
};

export type AuthProfileStep = {
  legalName: string;
  contactName: string;
  email: string;
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
}: {
  profile: AuthProfileStep;
  setProfile: Dispatch<SetStateAction<AuthProfileStep>>;
  errors: Errors;
}) {
  return (
    <div className="grid gap-4">
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
