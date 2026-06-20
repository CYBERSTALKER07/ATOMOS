"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { PortalField, PortalInput, PortalActions, FormAlert } from "@/components/portal";
import { SETUP_TAX_KEY } from "@/components/setup/constants";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

function getCookie(name: string) {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift();
}

export default function SetupAddressPage() {
  const router = useRouter();
  const [taxId, setTaxId] = useState("");
  const [billingAddress, setBillingAddress] = useState("");
  const [shippingAddress, setShippingAddress] = useState("");
  const [city, setCity] = useState("");
  const [postalCode, setPostalCode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const stored = sessionStorage.getItem(SETUP_TAX_KEY);
    if (!stored) {
      router.replace("/setup/tax");
      return;
    }
    setTaxId(stored);
  }, [router]);

  async function handleSubmit() {
    const errs: string[] = [];
    if (billingAddress.trim().length < 5) errs.push("Billing address is required.");
    if (shippingAddress.trim().length < 5) errs.push("Shipping address is required.");
    if (city.trim().length < 2) errs.push("City is required.");
    if (errs.length > 0) {
      setError(errs.join(" "));
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const token = getCookie("pegasus_retailer_jwt");
      const res = await fetch(`${API}/v1/retailer/setup`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          taxId,
          billingAddress: billingAddress.trim(),
          shippingAddress: shippingAddress.trim(),
          city: city.trim(),
          postalCode: postalCode.trim(),
        }),
      });

      if (!res.ok) {
        const errorData = await res.json().catch(() => null);
        throw new Error(errorData?.message || "Setup failed");
      }

      const data = await res.json();
      if (data?.token) {
        document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      }
      if (data?.refresh_token) {
        document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
      }

      sessionStorage.removeItem(SETUP_TAX_KEY);
      window.location.href = "/dashboard";
    } catch (err) {
      setError(err instanceof Error ? err.message : "Setup failed");
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="setup-form-title">Delivery addresses</h1>
        <p className="setup-form-sub">Where should we bill and deliver your orders?</p>
      </div>

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      <PortalField id="billing-address" label="Billing address">
        <PortalInput
          id="billing-address"
          value={billingAddress}
          onChange={(e) => setBillingAddress(e.target.value)}
          autoComplete="street-address"
        />
      </PortalField>

      <PortalField id="shipping-address" label="Shipping address (default)">
        <PortalInput
          id="shipping-address"
          value={shippingAddress}
          onChange={(e) => setShippingAddress(e.target.value)}
          autoComplete="shipping street-address"
        />
      </PortalField>

      <div className="grid gap-4 sm:grid-cols-2">
        <PortalField id="city" label="City">
          <PortalInput id="city" value={city} onChange={(e) => setCity(e.target.value)} />
        </PortalField>
        <PortalField id="postal" label="Postal code" optional>
          <PortalInput id="postal" value={postalCode} onChange={(e) => setPostalCode(e.target.value)} />
        </PortalField>
      </div>

      <PortalActions
        back={{ label: "Back", onClick: () => router.push("/setup/tax") }}
        primary={{ label: submitting ? "Saving…" : "Complete setup", onClick: handleSubmit, loading: submitting }}
      />
    </div>
  );
}
