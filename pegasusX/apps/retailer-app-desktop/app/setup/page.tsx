"use client";

import { usePortalT } from "@/lib/i18n";
import { redirect } from "next/navigation";

export default function SetupIndexPage() {
  const t = usePortalT();
  redirect("/setup/tax");
}
