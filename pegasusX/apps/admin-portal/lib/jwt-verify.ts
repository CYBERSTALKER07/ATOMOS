export interface VerifiedAdminClaims {
  subject: string;
  role: string;
  exp: number;
}

function decodeBase64UrlBytes(input: string): Uint8Array {
  const normalized = input.replace(/-/g, "+").replace(/_/g, "/");
  const padLen = (4 - (normalized.length % 4)) % 4;
  const padded = normalized + "=".repeat(padLen);
  if (typeof atob === "function") {
    const bin = atob(padded);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) {
      bytes[i] = bin.charCodeAt(i);
    }
    return bytes;
  }
  return new Uint8Array(Buffer.from(padded, "base64"));
}

function decodePayload(part: string): Record<string, unknown> | null {
  try {
    const bytes = decodeBase64UrlBytes(part);
    const text = new TextDecoder().decode(bytes);
    const parsed = JSON.parse(text);
    return parsed && typeof parsed === "object" ? (parsed as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}

function configuredHmacSecret(): string {
  return (process.env.ADMIN_JWT_SECRET || process.env.JWT_SECRET || "").trim();
}

export async function verifyAdminJwt(token: string): Promise<VerifiedAdminClaims | null> {
  if (!token || typeof token !== "string") return null;
  const parts = token.trim().split(".");
  if (parts.length !== 3) return null;

  const header = decodePayload(parts[0]);
  const payload = decodePayload(parts[1]);
  if (!header || !payload) return null;

  const exp = payload.exp;
  if (typeof exp !== "number" || !Number.isFinite(exp) || exp * 1000 <= Date.now()) {
    return null;
  }

  const role = String(payload.role || "").trim().toUpperCase();
  if (role !== "PLATFORM_ADMIN") {
    return null;
  }

  const secret = configuredHmacSecret();
  if (process.env.NODE_ENV === "production" && !secret) {
    // Production fail-closed when JWT signing secret is unconfigured on frontend BFF.
    return null;
  }

  if (secret) {
    const alg = String(header.alg || "").toUpperCase();
    if (alg !== "HS256") {
      return null;
    }
    try {
      const key = await crypto.subtle.importKey(
        "raw",
        new TextEncoder().encode(secret),
        { name: "HMAC", hash: "SHA-256" },
        false,
        ["verify"],
      );
      const signingInput = new TextEncoder().encode(`${parts[0]}.${parts[1]}`);
      const sigBytes = decodeBase64UrlBytes(parts[2]);
      const validSig = await crypto.subtle.verify(
        "HMAC",
        key,
        sigBytes as unknown as BufferSource,
        signingInput,
      );
      if (!validSig) {
        return null;
      }
    } catch {
      return null;
    }
  }

  const subject = String(payload.sub || payload.subject || "platform_admin");
  return { subject, role, exp };
}
