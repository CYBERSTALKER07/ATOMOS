// Shared step-form types for the supplier onboarding wizard.
//
// HARD PRODUCT INVARIANT: this wizard has exactly three steps —
//   1. Identity     (country selector + phone with auto-prefix)
//   2. Verification (email / OTP confirmation)
//   3. Profile      (business + warehouse profile fields)
// Bank + payment-gateway setup is intentionally NOT in this wizard —
// it lives at /setup/billing post-registration. Do not collapse below 3 steps.

export type StepId = "identity" | "verification" | "profile";

export interface Country {
  code: string;       // ISO-3166 alpha-2
  name: string;
  dialCode: string;   // e.g. "+998"
  currency: string;   // ISO-4217
}

export const COUNTRIES: Country[] = [
  { code: "UZ", name: "Uzbekistan",    dialCode: "+998", currency: "UZS" },
  { code: "KZ", name: "Kazakhstan",    dialCode: "+7",   currency: "KZT" },
  { code: "KG", name: "Kyrgyzstan",    dialCode: "+996", currency: "KGS" },
  { code: "TJ", name: "Tajikistan",    dialCode: "+992", currency: "TJS" },
  { code: "TM", name: "Turkmenistan",  dialCode: "+993", currency: "TMT" },
  { code: "AE", name: "United Arab Emirates", dialCode: "+971", currency: "AED" },
  { code: "TR", name: "Türkiye",       dialCode: "+90",  currency: "TRY" },
  { code: "RU", name: "Russia",        dialCode: "+7",   currency: "RUB" },
  { code: "US", name: "United States", dialCode: "+1",   currency: "USD" },
  { code: "GB", name: "United Kingdom",dialCode: "+44",  currency: "GBP" },
];

export interface IdentityStep {
  countryCode: string;     // ISO-3166 alpha-2
  phoneLocal: string;      // local portion only — UI shows the dial code separately
}

export interface VerificationStep {
  otpCode: string;
  idToken: string;
}

export interface ProfileStep {
  legalName: string;
  contactName: string;
  email: string;
}

export interface WizardState {
  step: StepId;
  identity: IdentityStep;
  verification: VerificationStep;
  profile: ProfileStep;
}

export const INITIAL_STATE: WizardState = {
  step: "identity",
  identity: {
    countryCode: "UZ",
    phoneLocal: "",
  },
  verification: {
    otpCode: "",
    idToken: "",
  },
  profile: {
    legalName: "",
    contactName: "",
    email: "",
  },
};

export const STEP_ORDER: StepId[] = ["identity", "verification", "profile"];

export const STEP_LABELS: Record<StepId, string> = {
  identity: "Phone Verification",
  verification: "Confirm Code",
  profile: "Basic Profile",
};

export function validateIdentity(s: IdentityStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (!s.countryCode) e.countryCode = "Country is required";
  if (!/^\d{6,14}$/.test(s.phoneLocal)) e.phoneLocal = "Phone digits only (6–14)";
  return e;
}

export function validateVerification(s: VerificationStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (!/^\d{6}$/.test(s.otpCode)) e.otpCode = "Enter the 6-digit code";
  return e;
}

export function validateProfile(s: ProfileStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (s.legalName.trim().length < 2) e.legalName = "Legal name is required";
  if (s.contactName.trim().length < 2) e.contactName = "Contact name is required";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s.email)) e.email = "Valid email is required";
  return e;
}
