"use client";

import { useState } from "react";
import Icon from "@/components/Icon";
import { PortalField, PortalInput, PortalSelect, PortalSection, FormAlert } from "@/components/portal";
import { factoryApiBaseUrl, persistSession } from "@/lib/auth";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

interface FactorySetupState {
  factoryName: string;
  facilityType: string;
  totalCapacitySqM: string;
}

const INITIAL: FactorySetupState = {
  factoryName: "",
  facilityType: "MANUFACTURING",
  totalCapacitySqM: "",
};

const DEFAULT_LOCATION: LocationValue = { address: "", lat: "0", lng: "0" };

export default function FactorySetupPage() {
  const [state, setState] = useState<FactorySetupState>(INITIAL);
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.factoryName.trim().length < 3) e.factoryName = "Name required";
    if (location.address.trim().length < 5) e.address = "Address required";
    const lat = Number(location.lat);
    const lng = Number(location.lng);
    if (!Number.isFinite(lat) || !Number.isFinite(lng) || (lat === 0 && lng === 0)) {
      e.address = "Select a valid address from suggestions";
    }
    if (!/^\d+$/.test(state.totalCapacitySqM)) e.totalCapacitySqM = "Capacity must be a number";
    return e;
  }

  async function submit() {
    const e = validate();
    setErrors(e);
    if (Object.keys(e).length > 0) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      const res = await fetch(`${factoryApiBaseUrl}/v1/factory/setup`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": cryptoRandomId(),
          Authorization: `Bearer ${getCookie("pegasus_factory_jwt")}`,
        },
        body: JSON.stringify({
          factoryName: state.factoryName.trim(),
          facilityType: state.facilityType,
          totalCapacitySqM: parseInt(state.totalCapacitySqM, 10),
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number(location.lat),
          lng: Number(location.lng),
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.message || body?.error || `Setup failed: ${res.status}`);
      }

      const data = await res.json();
      if (data.token) {
        persistSession(data.token, data.refresh_token);
      }

      window.location.href = "/";
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  function getCookie(name: string) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(";").shift();
  }

  return (
    <>
      <header className="setup-header">
        <div className="setup-header-icon" aria-hidden>
          <Icon name="factory" size={22} />
        </div>
        <div>
          <h1>Factory details</h1>
          <p className="setup-header-sub">Configure your facility type and location.</p>
        </div>
      </header>

      <PortalSection icon="factory" title="General">
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
        <PortalField id="totalCapacitySqM" label="Total capacity (square meters)" error={errors.totalCapacitySqM}>
          <PortalInput
            id="totalCapacitySqM"
            type="number"
            value={state.totalCapacitySqM}
            onChange={(e) => setState((s) => ({ ...s, totalCapacitySqM: e.target.value }))}
            error={errors.totalCapacitySqM}
          />
        </PortalField>
      </PortalSection>

      <PortalSection icon="loadingBay" title="Location" className="mt-6">
        <PortalField id="address" label="Factory address" error={errors.address}>
          <LocationPicker value={location} onChange={setLocation} label="Street address" />
        </PortalField>
      </PortalSection>

      {submitError ? <FormAlert variant="error">{submitError}</FormAlert> : null}

      <footer className="setup-footer">
        <div />
        <button type="button" className="portal-btn portal-btn--primary" onClick={submit} disabled={submitting}>
          {submitting ? "Saving…" : "Complete setup"}
          {!submitting ? <Icon name="arrow_forward" size={16} /> : null}
        </button>
      </footer>
    </>
  );
}

function cryptoRandomId() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
