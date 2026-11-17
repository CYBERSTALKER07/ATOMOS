import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { verifyAdminJwt } from "@/lib/jwt-verify";

const BACKEND_BASE = () =>
  (
    process.env.ADMIN_BACKEND_BASE_URL ||
    process.env.BACKEND_BASE_URL ||
    process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
    "http://localhost:8080"
  ).replace(/\/$/, "");

function allowDirectTokenLogin(): boolean {
  if (process.env.NODE_ENV === "production") {
    return false;
  }
  return process.env.ADMIN_ALLOW_DIRECT_TOKEN !== "false";
}

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as {
      subject?: unknown;
      password?: unknown;
      directToken?: unknown;
    };
    const { subject, password, directToken } = body;

    let token = "";
    let role = "";
    let sub = typeof subject === "string" && subject.trim() ? subject.trim() : "platform_admin";

    if (directToken && typeof directToken === "string") {
      if (!allowDirectTokenLogin()) {
        return NextResponse.json(
          { error: "Direct token login is disabled in production" },
          { status: 403 },
        );
      }
      const candidate = directToken.trim();
      const verifyRes = await fetch(`${BACKEND_BASE()}/v1/auth/session`, {
        method: "GET",
        headers: { Authorization: `Bearer ${candidate}` },
        cache: "no-store",
      });
      if (!verifyRes.ok) {
        return NextResponse.json({ error: "Invalid or expired direct token" }, { status: 401 });
      }
      const sess = (await verifyRes.json()) as { role?: string; subject?: string; sub?: string };
      token = candidate;
      role = String(sess.role || "").trim().toUpperCase();
      sub = sess.subject || sess.sub || sub;
    } else if (typeof subject === "string" && subject.trim() && typeof password === "string" && password) {
      const res = await fetch(`${BACKEND_BASE()}/v1/auth/platform-admin/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ subject: subject.trim(), password }),
        cache: "no-store",
      });

      if (!res.ok) {
        return NextResponse.json({ error: "Invalid credentials" }, { status: res.status === 429 ? 429 : 401 });
      }

      const data = (await res.json()) as { token?: string; role?: string; subject?: string };
      token = data.token || "";
      role = String(data.role || "").trim().toUpperCase();
      sub = data.subject || sub;
    } else {
      return NextResponse.json({ error: "Missing credentials" }, { status: 400 });
    }

    if (!token) {
      return NextResponse.json({ error: "No token returned from authentication" }, { status: 401 });
    }
    const verifiedClaims = await verifyAdminJwt(token);
    if (!verifiedClaims) {
      return NextResponse.json({ error: "Invalid or unauthorized admin token" }, { status: 403 });
    }
    if (role && role !== "PLATFORM_ADMIN") {
      return NextResponse.json({ error: "PLATFORM_ADMIN role required" }, { status: 403 });
    }
    role = verifiedClaims.role;
    sub = verifiedClaims.subject || sub;

    // Store JWT strictly in httpOnly cookie; never echo raw token to browser JS.
    const cookieStore = await cookies();
    cookieStore.set("pegasusx_admin_token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "strict",
      path: "/",
      maxAge: 60 * 60 * 24,
    });

    return NextResponse.json({ ok: true, subject: sub, role });
  } catch {
    return NextResponse.json({ error: "Authentication service error" }, { status: 500 });
  }
}
