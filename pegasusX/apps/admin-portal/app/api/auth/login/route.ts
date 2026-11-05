import { NextResponse } from "next/server";
import { cookies } from "next/headers";

const BACKEND_BASE = () =>
  (process.env.NEXT_PUBLIC_BACKEND_BASE_URL || "http://localhost:8080").replace(/\/$/, "");

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const { subject, password, directToken } = body;

    let token = "";
    let role = "PLATFORM_ADMIN";
    let sub = subject || "platform_admin";

    if (directToken && typeof directToken === "string") {
      const candidate = directToken.trim();
      // Verify token authenticity against backend session endpoint
      const verifyRes = await fetch(`${BACKEND_BASE()}/v1/auth/session`, {
        method: "GET",
        headers: { Authorization: `Bearer ${candidate}` },
      });
      if (!verifyRes.ok) {
        return NextResponse.json(
          { error: "Invalid or expired direct token" },
          { status: 401 },
        );
      }
      const sess = await verifyRes.json();
      token = candidate;
      role = sess.role || role;
      sub = sess.subject || sess.sub || sub;
    } else if (subject && password) {
      const res = await fetch(`${BACKEND_BASE()}/v1/auth/platform-admin/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ subject, password }),
      });

      if (!res.ok) {
        const errText = await res.text();
        return NextResponse.json(
          { error: errText || `Login failed (HTTP ${res.status})` },
          { status: res.status },
        );
      }

      const data = await res.json();
      token = data.token;
      role = data.role || role;
      sub = data.subject || sub;
    } else {
      return NextResponse.json({ error: "Missing credentials" }, { status: 400 });
    }

    if (!token) {
      return NextResponse.json({ error: "No token returned from authentication" }, { status: 401 });
    }

    // Set secure HTTP-only session cookie
    const cookieStore = await cookies();
    cookieStore.set("pegasusx_admin_token", token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "strict",
      path: "/",
      maxAge: 60 * 60 * 24, // 24 hours
    });

    return NextResponse.json({ ok: true, subject: sub, role, token });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "Internal authentication error" },
      { status: 500 },
    );
  }
}
