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
    process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN || "demo-pegasus.firebaseapp.com",
  projectId: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID || "demo-pegasus",
};

const app = getApps().length === 0 ? initializeApp(firebaseConfig) : getApp();
const auth: Auth = getAuth(app);

// Connect to Firebase Auth Emulator in development
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

/**
 * Exchange a Firebase Custom Token (from the backend) for a signed-in
 * Firebase user session. Returns the Firebase ID token string.
 * Returns "" on failure (graceful degradation — legacy JWT still works).
 */
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

/**
 * Get a fresh Firebase ID token for the currently signed-in user.
 * Returns "" if no Firebase session exists.
 */
export async function getFirebaseIdToken(): Promise<string> {
  const user: User | null = auth.currentUser;
  if (!user) return "";
  try {
    return await user.getIdToken(/* forceRefresh */ false);
  } catch {
    return "";
  }
}

/**
 * Sign out of Firebase Auth (call alongside legacy cookie clearing).
 */
export async function firebaseSignOut(): Promise<void> {
  try {
    await auth.signOut();
  } catch {
    // ignore
  }
}

export { auth };
