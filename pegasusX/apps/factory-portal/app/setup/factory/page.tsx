"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Icon from "@/components/Icon";
import { PortalField, PortalInput, PortalSelect, PortalSection, FormAlert } from "@/components/portal";
import { factoryOpsLocationKey } from "@pegasusx/api-client";
import { apiFetch, decodeJwtPayload, persistSession, readTokenFromCookie, refreshFactorySession } from "@/lib/auth";
import { factoryOperatorId } from "@/lib/factory-scope";
import { LocationPicker, resolveLocationValue, type LocationValue } from "@/components/LocationPicker";
import { hasValidCoordinates } from "@/lib/geocode";

interface FactorySetupState {
  factoryName: string;
  facilityType: string;
}

const INITIAL: FactorySetupState = {
  factoryName: "",
  facilityType: "MANUFACTURING",
};

const DEFAULT_LOCATION: LocationValue = { address: "", lat: "0", lng: "0" };

type FactoryLocation = {
  factory_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
};

export default function FactorySetupPage() {
  const [state, setState] = useState<FactorySetupState>(INITIAL);
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);
  const [loading, setLoading] = useState(true);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const hasAssignedFactory = useMemo(() => {
    const token = readTokenFromCookie();
    const claims = token ? decodeJwtPayload(token) : null;
    return typeof claims?.home_node_id === "string" && claims.home_node_id.length > 0;
  }, []);

  const loadExisting = useCallback(async () => {
    setLoading(true);
    try {
      if (!hasAssignedFactory) return;
      const res = await apiFetch("/v1/factory/ops/location");
      if (!res.ok) return;
      const loc = (await res.json()) as FactoryLocation;
      setState((s) => ({ ...s, factoryName: loc.name ?? s.factoryName }));
      setLocation({
        address: loc.address ?? "",
        lat: String(loc.lat ?? 0),
        lng: String(loc.lng ?? 0),
        place_id: loc.place_id,
      });
    } finally {
      setLoading(false);
    }
  }, [hasAssignedFactory]);

  useEffect(() => {
    void loadExisting();
  }, [loadExisting]);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (!hasAssignedFactory && state.factoryName.trim().length < 3) {
      e.factoryName = "Name required";
    }
    if (!location.address.trim()) {
      e.address = "Address required";
    }
    return e;
  }

  async function submit() {
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;

    setSubmitting(true);
    setSubmitError(null);
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

      if (hasAssignedFactory) {
        const res = await apiFetch("/v1/factory/ops/location", {
          method: "PATCH",
          headers: {
            "Idempotency-Key": factoryOpsLocationKey(
              factoryOperatorId() || "factory",
              latN,
              lngN,
              place_id,
            ),
          },
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
        const refreshed = await refreshFactorySession();
        if (!refreshed.ok) {
          throw new Error("Location saved but session could not be refreshed. Sign in again.");
        }
      } else {
        const res = await apiFetch("/v1/factory/setup", {
          method: "POST",
          body: JSON.stringify({
            factoryName: state.factoryName.trim(),
            facilityType: state.facilityType,
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
          <Icon name="factory" size={22} />
        </div>
        <div>
          <h1>Factory location</h1>
          <p className="setup-header-sub">
            {hasAssignedFactory
              ? "Confirm or update your facility address. Changes stay in sync with supply routing and loading bay operations."
              : "Name your factory and set the facility address so loading bay, transfers, and manifests can start."}
          </p>
        </div>
      </header>

      {loading ? (
        <p className="text-sm text-(--muted)">Loading factory details…</p>
      ) : (
        <>
          <PortalSection icon="factory" title="General">
            {!hasAssignedFactory ? (
              <>
                <PortalField id="factoryName" label="Factory name" error={errors.factoryName}>
                  <PortalInput
                    id="factoryName"
                    value={state.factoryName}
                    onChange={(e) => setState((s) => ({ ...s, factoryName: e.target.value }))}
                    error={errors.factoryName}
                  />
                </PortalField>
                <PortalField id="facilityType" label="Facility type" error={errors.facilityType}>
                  <PortalSelect
                    id="facilityType"
                    value={state.facilityType}
                    onChange={(e) => setState((s) => ({ ...s, facilityType: e.target.value }))}
                    error={errors.facilityType}
                  >
                    <option value="MANUFACTURING">Manufacturing</option>
                    <option value="ASSEMBLY">Assembly</option>
                    <option value="PACKAGING">Packaging</option>
                    <option value="PROCESSING">Processing</option>
                  </PortalSelect>
                </PortalField>
              </>
            ) : state.factoryName ? (
              <div>
                <p className="text-xs font-medium text-(--muted)">Factory</p>
                <p className="text-sm font-semibold">{state.factoryName}</p>
              </div>
            ) : null}
          </PortalSection>

          <PortalSection icon="loadingBay" title="Location" className="mt-6">
            <PortalField id="address" label="Factory address" error={errors.address}>
              <LocationPicker value={location} onChange={setLocation} label="Street address" />
            </PortalField>
          </PortalSection>
        </>
      )}

      {submitError ? <FormAlert variant="error">{submitError}</FormAlert> : null}

      <footer className="setup-footer">
        <div />
        <button type="button" className="portal-btn portal-btn--primary" onClick={() => void submit()} disabled={submitting || loading}>
          {submitting ? "Saving…" : "Complete setup"}
          {!submitting ? <Icon name="arrow_forward" size={16} /> : null}
        </button>
      </footer>
    </>
  );
}
