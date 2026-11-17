import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { verifyAdminJwt } from "@/lib/jwt-verify";

export async function GET() {
  const cookieStore = await cookies();
  const tokenCookie = cookieStore.get("pegasusx_admin_token");

  if (!tokenCookie || !tokenCookie.value) {
    return NextResponse.json({ authenticated: false });
  }

  const verified = await verifyAdminJwt(tokenCookie.value);
  if (!verified) {
    return NextResponse.json({ authenticated: false });
  }

  return NextResponse.json({
    authenticated: true,
    subject: verified.subject,
    role: verified.role,
  });
}
