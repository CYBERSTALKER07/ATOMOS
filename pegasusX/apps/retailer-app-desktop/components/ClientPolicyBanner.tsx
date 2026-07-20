"use client";

import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { desktopClientPolicyContext } from "@pegasusx/desktop-bridge";
import { getClientPolicy } from "../lib/api";

type PolicyState = {
  forceUpdate: boolean;
  outdated: boolean;
  minimumVersion: string;
  updateUrl?: string;
  deferReason?: string;
};

export default function ClientPolicyBanner() {
  const [policy, setPolicy] = useState<PolicyState | null>(null);

  useEffect(() => {
    const version =
      process.env.NEXT_PUBLIC_RETAILER_APP_VERSION?.trim() || "0.1.0";
    // Tauri: enterprise → website CDN; store → production (MS/Mac App Store).
    const { platform, channel } = desktopClientPolicyContext();
    void getClientPolicy(platform, version, channel)
      .then(async (res) => {
        if (!res.ok) return;
        const data = (await res.json()) as {
          force_update?: boolean;
          outdated?: boolean;
          minimum_version?: string;
          update_url?: string;
          defer_reason?: string;
        };
        if (data.outdated || data.force_update) {
          setPolicy({
            forceUpdate: Boolean(data.force_update),
            outdated: Boolean(data.outdated),
            minimumVersion: data.minimum_version ?? "",
            updateUrl: data.update_url,
            deferReason: data.defer_reason,
          });
        }
      })
      .catch(() => {
        // Policy endpoint optional on local dev stacks.
      });
  }, []);

  if (!policy) return null;

  return (
    <div className="mx-6 mt-4 mb-0 flex items-start gap-3 rounded-2xl border border-orange-200 bg-orange-50 px-4 py-3 text-orange-900">
      <AlertTriangle size={18} className="shrink-0 mt-0.5" />
      <div className="text-sm">
        <p className="font-light">
          {policy.forceUpdate ? "Update required" : "Update available"}
        </p>
        <p className="mt-1 opacity-90">
          Minimum supported version is {policy.minimumVersion || "newer"}.
          {policy.deferReason ? ` ${policy.deferReason}` : ""}
        </p>
        {policy.updateUrl && (
          <a
            href={policy.updateUrl}
            className="mt-2 inline-block font-light underline"
            target="_blank"
            rel="noreferrer"
          >
            Get the latest build
          </a>
        )}
      </div>
    </div>
  );
}
