import { initializeApp, getApps, getApp } from "firebase/app";
import {
  getAuth,
  connectAuthEmulator,
  signInWithCustomToken,
  type Auth,
  type User,
} from "firebase/auth";

const firebaseConfig = {
  apiKey: process.env.NEXT_PUBLIC_FIREBASE_API_KEY || "demo-key",
  authDomain:
    process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN || "demo-thelab.firebaseapp.com",
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID || "demo-thelab",
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

export { auth };
