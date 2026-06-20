"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch } from "@/lib/auth";
import PageTransition from "@/components/PageTransition";
import { PageChrome } from "@/components/PageChrome";
import { PortalSection } from "@/components/portal";
import { useToast } from "@/components/Toast";
import { LocationPicker, resolveLocationValue, type LocationValue } from "@/components/LocationPicker";
import { hasValidCoordinates } from "@/lib/geocode";

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
      let resolved = location;
      if (!hasValidCoordinates(location.lat, location.lng)) {
        const next = await resolveLocationValue(location);
        if (!next) {
          toast("Pick an address from the suggestions or share your location", "error");
          return;
        }
        resolved = next;
        setLocation(next);
      }
      const res = await apiFetch("/v1/factory/ops/location", {
        method: "PATCH",
        body: JSON.stringify({
          address: resolved.address.trim(),
          place_id: resolved.place_id,
          lat: Number.parseFloat(resolved.lat),
          lng: Number.parseFloat(resolved.lng),
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
        icon="loadingBay"
        title="Factory location"
        description="Street address used for supply routing and loading bay operations. Changes sync across the factory network."
        loading={loading}
        skeletonVariant="form"
      >
        <PortalSection icon="factory" title="Depot address" className="max-w-xl">
          {factoryName ? (
            <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
              <strong>{factoryName}</strong>
            </p>
          ) : null}
          <LocationPicker value={location} onChange={setLocation} label="Factory address" />
          <button
            type="button"
            className="portal-btn portal-btn--primary"
            onClick={() => void saveLocation()}
            disabled={saving}
          >
            {saving ? "Saving…" : "Save location"}
          </button>
        </PortalSection>
      </PageChrome>
    </PageTransition>
  );
}
