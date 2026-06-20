"use client";

import { useState, type ChangeEvent } from "react";
import Icon from "@/components/Icon";
import { PortalField, PortalInput, FormAlert } from "@/components/portal";
import { warehouseApiBaseUrl } from "@/lib/auth";

interface WarehouseSetupState {
  warehouseName: string;
  address: string;
  city: string;
  postalCode: string;
  totalCapacitySqM: string;
}

const INITIAL: WarehouseSetupState = {
  warehouseName: "",
  address: "",
  city: "",
  postalCode: "",
  totalCapacitySqM: "",
};

export default function WarehouseLocationSetupPage() {
  const [state, setState] = useState<WarehouseSetupState>(INITIAL);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function validate(): Record<string, string> {
    const e: Record<string, string> = {};
    if (state.warehouseName.trim().length < 3) e.warehouseName = "Name required";
    if (state.address.trim().length < 5) e.address = "Address required";
    if (state.city.trim().length < 2) e.city = "City required";
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
      const res = await fetch(`${warehouseApiBaseUrl}/v1/warehouse/setup`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": cryptoRandomId(),
          Authorization: `Bearer ${getCookie("pegasus_warehouse_jwt")}`,
        },
        body: JSON.stringify({
          ...state,
          totalCapacitySqM: parseInt(state.totalCapacitySqM, 10),
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.message || `Setup failed: ${res.status}`);
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
          <Icon name="warehouse" size={22} />
        </div>
        <div>
          <h1>Warehouse location</h1>
          <p className="setup-header-sub">Configure your warehouse location and operating capacity.</p>
        </div>
      </header>

      <section className="setup-card space-y-4">
        <h2 className="setup-section-title">General</h2>
        <PortalField id="warehouseName" label="Warehouse name" error={errors.warehouseName}>
          <PortalInput
            id="warehouseName"
            value={state.warehouseName}
            onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, warehouseName: e.target.value }))}
            error={errors.warehouseName}
          />
        </PortalField>
        <PortalField id="totalCapacitySqM" label="Total capacity (square meters)" error={errors.totalCapacitySqM}>
          <PortalInput
            id="totalCapacitySqM"
            type="number"
            value={state.totalCapacitySqM}
            onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, totalCapacitySqM: e.target.value }))}
            error={errors.totalCapacitySqM}
          />
        </PortalField>

        <h2 className="setup-section-title mt-4">Location</h2>
        <PortalField id="address" label="Street address" error={errors.address}>
          <PortalInput
            id="address"
            value={state.address}
            onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, address: e.target.value }))}
            error={errors.address}
          />
        </PortalField>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <PortalField id="city" label="City" error={errors.city}>
            <PortalInput
              id="city"
              value={state.city}
              onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, city: e.target.value }))}
              error={errors.city}
            />
          </PortalField>
          <PortalField id="postalCode" label="Postal code" optional error={errors.postalCode}>
            <PortalInput
              id="postalCode"
              value={state.postalCode}
              onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setState((s) => ({ ...s, postalCode: e.target.value }))}
              error={errors.postalCode}
            />
          </PortalField>
        </div>
      </section>

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
