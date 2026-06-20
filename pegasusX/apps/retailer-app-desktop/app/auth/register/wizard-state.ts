// Shared step-form types for desktop portal registration wizards.

export type StepId = "identity" | "verification" | "profile";

export {
  AUTH_COUNTRIES as COUNTRIES,
  dialCodeForCountry,
} from "@pegasusx/ui-kit/auth";

export type { AuthCountry as Country } from "@pegasusx/ui-kit/auth";

export interface IdentityStep {
  countryCode: string;
  phoneLocal: string;
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
