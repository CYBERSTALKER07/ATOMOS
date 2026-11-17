"use client";

import { usePortalT } from "@/lib/i18n";
import { redirect } from "next/navigation";

export default function HomePage() {
  const t = usePortalT();
  redirect("/dashboard");
}
