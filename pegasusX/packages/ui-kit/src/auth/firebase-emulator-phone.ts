const DEFAULT_API_KEY = "demo-key";

function emulatorIdentityToolkitBase(): string {
  if (typeof window !== "undefined") {
    return `${window.location.origin}/identitytoolkit.googleapis.com`;
  }
  return "http://127.0.0.1:9099/identitytoolkit.googleapis.com";
}

let activeSessionInfo: string | null = null;

export async function sendPhoneOtpViaEmulator(
  phone: string,
  apiKey = DEFAULT_API_KEY,
): Promise<void> {
  const trimmed = phone.trim();
  if (!trimmed) throw new Error("Phone number required");

  const res = await fetch(
    `${emulatorIdentityToolkitBase()}/v1/accounts:sendVerificationCode?key=${encodeURIComponent(apiKey)}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        phoneNumber: trimmed,
        recaptchaToken: "emulator-recaptcha-token",
      }),
    },
  );

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `Failed to send verification code (${res.status})`);
  }

  const data = (await res.json()) as { sessionInfo?: string };
  if (!data.sessionInfo) {
    throw new Error("Auth emulator did not return sessionInfo");
  }
  activeSessionInfo = data.sessionInfo;
}

export async function verifyPhoneOtpViaEmulator(
  code: string,
  apiKey = DEFAULT_API_KEY,
): Promise<string> {
  if (!activeSessionInfo) {
    throw new Error("No verification in progress; request a code first");
  }

  const res = await fetch(
    `${emulatorIdentityToolkitBase()}/v1/accounts:signInWithPhoneNumber?key=${encodeURIComponent(apiKey)}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        sessionInfo: activeSessionInfo,
        code: code.trim(),
      }),
    },
  );

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `Invalid verification code (${res.status})`);
  }

  const data = (await res.json()) as { idToken?: string };
  activeSessionInfo = null;
  if (!data.idToken) {
    throw new Error("Auth emulator did not return idToken");
  }
  return data.idToken;
}

export function resetEmulatorPhoneOtpFlow(): void {
  activeSessionInfo = null;
}
