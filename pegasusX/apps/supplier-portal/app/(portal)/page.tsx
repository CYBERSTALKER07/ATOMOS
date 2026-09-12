"use client";

import { usePortalT } from "@/lib/i18n";
import { redirect } from "next/navigation";

export default function PortalIndex() {
  const t = usePortalT();
  redirect("/dashboard");
}
