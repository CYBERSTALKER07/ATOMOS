// Shared step-form types for the retailer registration wizard.

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
}

export interface ProfileStep {
  legalName: string;
  contactName: string;
  email: string;
  latitude: string;
  longitude: string;
  receivingWindowOpen: string;
  receivingWindowClose: string;
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
  },
  profile: {
    legalName: "",
    contactName: "",
    email: "",
    latitude: "41.2995",
    longitude: "69.2401",
    receivingWindowOpen: "09:00",
    receivingWindowClose: "18:00",
  },
};

export const STEP_ORDER: StepId[] = ["identity", "verification", "profile"];

export const STEP_LABELS: Record<StepId, string> = {
  identity: "Phone Verification",
  verification: "Confirm Code",
  profile: "Store Profile",
};

export {
  normalizeReceivingWindow,
  validateReceivingWindowField,
} from "../../../lib/receiving-window";

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
  const lat = Number.parseFloat(s.latitude.trim());
  const lng = Number.parseFloat(s.longitude.trim());
  if (!Number.isFinite(lat) || lat < -90 || lat > 90) e.latitude = "Valid latitude required";
  if (!Number.isFinite(lng) || lng < -180 || lng > 180) e.longitude = "Valid longitude required";
  if (lat === 0 && lng === 0) e.latitude = "Store coordinates required";
  const openError = validateReceivingWindowField(s.receivingWindowOpen);
  if (openError) e.receivingWindowOpen = openError;
  const closeError = validateReceivingWindowField(s.receivingWindowClose);
  if (closeError) e.receivingWindowClose = closeError;
  return e;
}
