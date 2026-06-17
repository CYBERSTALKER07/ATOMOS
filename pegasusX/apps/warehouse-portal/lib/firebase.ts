import { initializeApp, getApps, getApp } from "firebase/app";
import {
  getAuth,
  connectAuthEmulator,
  signInWithCustomToken,
  signInWithPhoneNumber,
  type Auth,
  type ConfirmationResult,
  type User,
} from "firebase/auth";

const firebaseConfig = {
  apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY || "demo-key",
  authDomain:
    process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN || "demo-pegasus.firebaseapp.com",
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID || "demo-pegasus",
};

const app = getApps().length === 0 ? initializeApp(firebaseConfig) : getApp();
const auth: Auth = getAuth(app);

if (
  typeof window !== "undefined" &&
  process.env.NODE_ENV === "development" &&
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  !(auth as any)._emulatorConnected
) {
  const emulatorHost =
    process.env.NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST ||
    "http://localhost:9099";
  connectAuthEmulator(auth, emulatorHost, { disableWarnings: true });
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (auth as any)._emulatorConnected = true;
}

export async function exchangeCustomToken(
  customToken: string
): Promise<string> {
  if (!customToken) return "";
  try {
    const cred = await signInWithCustomToken(auth, customToken);
    return await cred.user.getIdToken();
  } catch (err) {
    console.warn("[firebase] custom token exchange failed:", err);
    return "";
  }
}

export async function getFirebaseIdToken(): Promise<string> {
  const user: User | null = auth.currentUser;
  if (!user) return "";
  try {
    return await user.getIdToken(false);
  } catch {
    return "";
  }
}

export async function firebaseSignOut(): Promise<void> {
  try {
    await auth.signOut();
  } catch {
    // ignore
  }
}

let phoneConfirmation: ConfirmationResult | null = null;

class EmulatorRecaptchaVerifier {
  type = "recaptcha" as const;

  async verify(): Promise<string> {
    return "emulator-recaptcha-token";
  }

  _reset(): void {}
}

export async function sendPhoneOtp(phone: string): Promise<void> {
  const trimmed = phone.trim();
  if (!trimmed) throw new Error("Phone number required");
  const verifier = new EmulatorRecaptchaVerifier();
  phoneConfirmation = await signInWithPhoneNumber(auth, trimmed, verifier as never);
}

export async function verifyPhoneOtp(code: string): Promise<string> {
  if (!phoneConfirmation) {
    throw new Error("No verification in progress; request a code first");
  }
  const result = await phoneConfirmation.confirm(code.trim());
  phoneConfirmation = null;
  return result.user.getIdToken();
}

export function resetPhoneOtpFlow(): void {
  phoneConfirmation = null;
}

export { auth };
