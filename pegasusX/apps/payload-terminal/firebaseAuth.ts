import { FirebaseApp, getApps, initializeApp } from 'firebase/app';
import {
  Auth,
  ConfirmationResult,
  connectAuthEmulator,
  getAuth,
  signInWithPhoneNumber,
} from 'firebase/auth';

const firebaseConfig = {
  apiKey: process.env.EXPO_PUBLIC_FIREBASE_API_KEY ,
  authDomain: process.env.EXPO_PUBLIC_FIREBASE_AUTH_DOMAIN ,
  projectId: process.env.EXPO_PUBLIC_FIREBASE_PROJECT_ID ,
  appId: process.env.EXPO_PUBLIC_FIREBASE_APP_ID ?? '1:000000000000:web:0000000000000001',
};

let authInstance: Auth | null = null;
let confirmation: ConfirmationResult | null = null;
let emulatorConnected = false;

/** Minimal verifier for Firebase Auth Emulator (no real reCAPTCHA in local dev). */
class EmulatorRecaptchaVerifier {
  type = 'recaptcha' as const;

  async verify(): Promise<string> {
    return 'emulator-recaptcha-token';
  }

  _reset(): void {}
}

function getFirebaseAuth(): Auth {
  if (authInstance) return authInstance;
  
  if (!firebaseConfig.apiKey && typeof process !== "undefined" && process.env.NODE_ENV === "development") {
    firebaseConfig.apiKey = "demo-key";
    firebaseConfig.authDomain = "demo-pegasus.firebaseapp.com";
    firebaseConfig.projectId = "demo-pegasus";
  } else if (!firebaseConfig.apiKey) {
    throw new Error("Firebase config missing in production");
  }
  const app: FirebaseApp = getApps().length > 0 ? getApps()[0]! : initializeApp(firebaseConfig);
  authInstance = getAuth(app);
  const emulatorHost = (process.env.EXPO_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST ?? '').trim();
  if (emulatorHost && !emulatorConnected) {
    const url = emulatorHost.startsWith('http') ? emulatorHost : `http://${emulatorHost}`;
    connectAuthEmulator(authInstance, url, { disableWarnings: true });
    emulatorConnected = true;
  }
  return authInstance;
}

export async function sendPhoneOtp(phone: string): Promise<void> {
  const trimmed = phone.trim();
  if (!trimmed) throw new Error('Phone number required');
  const auth = getFirebaseAuth();
  const verifier = new EmulatorRecaptchaVerifier();
  confirmation = await signInWithPhoneNumber(auth, trimmed, verifier as never);
}

export async function verifyPhoneOtp(code: string): Promise<string> {
  if (!confirmation) {
    throw new Error('No verification in progress; request a code first');
  }
  const result = await confirmation.confirm(code.trim());
  confirmation = null;
  const token = await result.user.getIdToken();
  return token;
}

export function resetPhoneOtpFlow(): void {
  confirmation = null;
}
