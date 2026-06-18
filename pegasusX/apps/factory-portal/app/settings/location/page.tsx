"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch } from "@/lib/auth";
import PageTransition from "@/components/PageTransition";
import { PageChrome } from "@/components/PageChrome";
import { useToast } from "@/components/Toast";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

type FactoryLocation = {
  factory_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
};

export default function FactoryLocationSettingsPage() {
  const { toast } = useToast();
  const [factoryName, setFactoryName] = useState("");
  const [location, setLocation] = useState<LocationValue>({ address: "", lat: "0", lng: "0" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch("/v1/factory/ops/location");
      if (res.ok) {
        const loc = (await res.json()) as FactoryLocation;
        setFactoryName(loc.name ?? "");
        setLocation({
          address: loc.address ?? "",
          lat: String(loc.lat ?? 0),
          lng: String(loc.lng ?? 0),
          place_id: loc.place_id,
        });
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveLocation() {
    if (!location.address.trim()) {
      toast("Factory address is required", "error");
      return;
    }
    setSaving(true);
    try {
      const res = await apiFetch("/v1/factory/ops/location", {
        method: "PATCH",
        body: JSON.stringify({
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number(location.lat),
          lng: Number(location.lng),
        }),
      });
      if (res.ok) {
        toast("Factory location saved", "success");
        await load();
      } else {
        const data = await res.json().catch(() => ({}));
        toast((data as { error?: string }).error || "Save failed", "error");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        title="Factory location"
        subtitle="Street address used for supply routing and dispatch."
      />
      <section className="auth-card grid gap-4 max-w-xl">
        {loading ? (
          <p className="md-typescale-body-medium">Loading location…</p>
        ) : (
          <>
            {factoryName ? (
              <p className="md-typescale-body-medium">
                <strong>{factoryName}</strong>
              </p>
            ) : null}
            <LocationPicker value={location} onChange={setLocation} label="Factory address" />
            <button
              type="button"
              className="md-btn md-btn-filled w-fit"
              onClick={() => void saveLocation()}
              disabled={saving}
            >
              {saving ? "Saving…" : "Save location"}
            </button>
          </>
        )}
      </section>
    </PageTransition>
  );
}
