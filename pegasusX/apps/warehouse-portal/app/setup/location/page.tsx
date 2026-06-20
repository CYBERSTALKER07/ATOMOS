"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Icon from "@/components/Icon";
import { FormAlert, PortalField, PortalInput } from "@/components/portal";
import { LocationPicker, resolveLocationValue, type LocationValue } from "@/components/LocationPicker";
import { hasValidCoordinates } from "@/lib/geocode";
import {
  apiFetch,
  decodeJwtPayload,
  persistSession,
  readTokenFromCookie,
  refreshWarehouseSession,
} from "@/lib/auth";

type WarehouseLocation = {
  warehouse_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
};

const EMPTY_LOCATION: LocationValue = { address: "", lat: "0", lng: "0" };

export default function WarehouseLocationSetupPage() {
  const [warehouseName, setWarehouseName] = useState("");
  const [location, setLocation] = useState<LocationValue>(EMPTY_LOCATION);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const hasAssignedWarehouse = useMemo(() => {
    const token = readTokenFromCookie();
    const claims = token ? decodeJwtPayload(token) : null;
    return typeof claims?.home_node_id === "string" && claims.home_node_id.length > 0;
  }, []);

  const loadExisting = useCallback(async () => {
    setLoading(true);
    try {
      if (!hasAssignedWarehouse) return;

      const res = await apiFetch("/v1/warehouse/ops/location");
      if (!res.ok) return;
      const loc = (await res.json()) as WarehouseLocation;
      setWarehouseName(loc.name ?? "");
      setLocation({
        address: loc.address ?? "",
        lat: String(loc.lat ?? 0),
        lng: String(loc.lng ?? 0),
        place_id: loc.place_id,
      });
    } finally {
      setLoading(false);
    }
  }, [hasAssignedWarehouse]);

  useEffect(() => {
    void loadExisting();
  }, [loadExisting]);

  async function submit() {
    setSubmitError(null);
    if (!hasAssignedWarehouse && warehouseName.trim().length < 3) {
      setSubmitError("Warehouse name is required (at least 3 characters).");
      return;
    }
    if (!location.address.trim()) {
      setSubmitError("Select an address from the suggestions or share your location.");
      return;
    }
    if (!hasValidCoordinates(location.lat, location.lng)) {
      // submit handler will try forward geocode before failing
    } else {
      const lat = Number.parseFloat(location.lat);
      const lng = Number.parseFloat(location.lng);
      if (!Number.isFinite(lat) || !Number.isFinite(lng)) {
        setSubmitError("Valid coordinates are required. Use address search or share location.");
        return;
      }
    }

    setSubmitting(true);
    try {
      let resolvedLocation = location;
      if (!hasValidCoordinates(location.lat, location.lng)) {
        const resolved = await resolveLocationValue(location);
        if (!resolved) {
          setSubmitError("Select an address from the suggestions or share your location.");
          return;
        }
        resolvedLocation = resolved;
        setLocation(resolved);
      }

      const { address, lat, lng, place_id } = resolvedLocation;
      const latN = Number.parseFloat(lat);
      const lngN = Number.parseFloat(lng);

      if (hasAssignedWarehouse) {
        const res = await apiFetch("/v1/warehouse/ops/location", {
          method: "PATCH",
          body: JSON.stringify({
            address: address.trim(),
            place_id,
            lat: latN,
            lng: lngN,
          }),
        });
        if (!res.ok) {
          const body = await res.json().catch(() => null);
          throw new Error((body as { error?: string })?.error || `Setup failed: ${res.status}`);
        }
        const refreshed = await refreshWarehouseSession();
        if (!refreshed.ok) {
          throw new Error("Location saved but session could not be refreshed. Sign in again.");
        }
      } else {
        const res = await apiFetch("/v1/warehouse/setup", {
          method: "POST",
          body: JSON.stringify({
            name: warehouseName.trim(),
            address: address.trim(),
            place_id,
            lat: latN,
            lng: lngN,
          }),
        });
        if (!res.ok) {
          const body = await res.json().catch(() => null);
          throw new Error((body as { error?: string })?.error || `Setup failed: ${res.status}`);
        }
        const data = (await res.json()) as { token?: string; refresh_token?: string };
        if (!data.token) {
          throw new Error("Setup succeeded but no session was returned.");
        }
        persistSession(data.token, data.refresh_token);
      }

      window.location.href = "/";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <header className="setup-header">
        <div className="setup-header-icon" aria-hidden>
          <Icon name="warehouse" size={22} />
        </div>
        <div>
          <h1>Warehouse location</h1>
          <p className="setup-header-sub">
            {hasAssignedWarehouse
              ? "Confirm or update your depot address. Changes stay in sync with dispatch, delivery fees, and fleet routing."
              : "Name your warehouse and set the depot address so dispatch, delivery fees, and fleet routing can start."}
          </p>
        </div>
      </header>

      <section className="setup-card space-y-4">
        {loading ? (
          <p className="text-sm text-(--muted)">Loading warehouse details…</p>
        ) : (
          <>
            {!hasAssignedWarehouse ? (
              <PortalField id="warehouseName" label="Warehouse name">
                <PortalInput
                  id="warehouseName"
                  value={warehouseName}
                  onChange={(e) => setWarehouseName(e.target.value)}
                  placeholder="Central depot"
                />
              </PortalField>
            ) : warehouseName ? (
              <div>
                <p className="text-xs font-medium text-(--muted)">Warehouse</p>
                <p className="text-sm font-semibold">{warehouseName}</p>
              </div>
            ) : null}

            <LocationPicker value={location} onChange={setLocation} label="Depot address" />

            <p className="text-xs text-(--muted)">
              Search for your street address or use share location. You can edit this anytime under Settings.
            </p>
          </>
        )}
      </section>

      {submitError ? <FormAlert variant="error">{submitError}</FormAlert> : null}

      <footer className="setup-footer">
        <div />
        <button
          type="button"
          className="portal-btn portal-btn--primary"
          onClick={() => void submit()}
          disabled={submitting || loading}
        >
          {submitting ? "Saving…" : "Complete setup"}
          {!submitting ? <Icon name="arrow_forward" size={16} /> : null}
        </button>
      </footer>
    </>
  );
}
