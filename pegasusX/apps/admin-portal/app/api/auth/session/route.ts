import { NextResponse } from "next/server";
import { cookies } from "next/headers";

export async function GET() {
  const cookieStore = await cookies();
  const tokenCookie = cookieStore.get("pegasusx_admin_token");

  if (!tokenCookie || !tokenCookie.value) {
    return NextResponse.json({ authenticated: false });
  }

  return NextResponse.json({
    authenticated: true,
    token: tokenCookie.value,
  });
}
