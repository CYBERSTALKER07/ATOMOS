"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getRetailerProfileFromServer, setupRetailer } from "@/lib/api";
import { persistFullAuthTokens } from "@/lib/multi-org-auth";
import { forwardGeocode } from "@/lib/geocode";
import { getRetailerId, setRetailerProfile } from "@/lib/retailer-profile";
import { PortalField, PortalInput, PortalActions, FormAlert } from "@/components/portal";
import {
  clearTaxSetupDraft,
  readTaxSetupDraft,
  type TaxIdentitySetupDraft,
} from "@/components/setup/constants";

export default function SetupAddressPage() {
  const t = usePortalT();
  const router = useRouter();
  const [taxDraft, setTaxDraft] = useState<TaxIdentitySetupDraft | null>(null);
  const [billingAddress, setBillingAddress] = useState("");
  const [shippingAddress, setShippingAddress] = useState("");
  const [city, setCity] = useState("");
  const [postalCode, setPostalCode] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const stored = readTaxSetupDraft();
    if (stored) {
      setTaxDraft(stored);
      return;
    }
    let cancelled = false;
    void getRetailerProfileFromServer().then((profile) => {
      if (cancelled) return;
      if (profile?.tax_id) {
        setTaxDraft({
          mode: "LEGAL_VERIFIED",
          taxId: profile.tax_id,
          legalName: profile.name,
          persisted: true,
        });
        return;
      }
      if (profile?.legal_entity_status === "INFORMAL_PHONE") {
        setTaxDraft({ mode: "INFORMAL_PHONE", taxId: "", persisted: true });
        return;
      }
      router.replace("/setup/tax");
    });
    return () => {
      cancelled = true;
    };
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
      const retailerId = getRetailerId() || "retailer";
      const isInformal = taxDraft?.mode === "INFORMAL_PHONE";
      const fullShipping = [shippingAddress.trim(), city.trim(), postalCode.trim()].filter(Boolean).join(", ");
      const geo = await forwardGeocode(fullShipping).catch(() => null);
      const res = await setupRetailer(
        {
          ...(isInformal
            ? { informal_shop: true, legal_entity_status: "INFORMAL_PHONE" }
            : {
                tax_id: taxDraft?.taxId || "",
                inn: taxDraft?.taxId || "",
                legal_entity_status: "LEGAL_VERIFIED",
              }),
          ...(taxDraft?.legalName ? { name: taxDraft.legalName } : {}),
          billingAddress: billingAddress.trim(),
          shippingAddress: shippingAddress.trim(),
          billing_address: billingAddress.trim(),
          delivery_address: shippingAddress.trim(),
          city: city.trim(),
          postalCode: postalCode.trim(),
          ...(geo ? { lat: geo.lat, lng: geo.lng } : {}),
        },
        retailerId,
      );

      if (!res.ok) {
        const errorData = await res.json().catch(() => null);
        throw new Error(errorData?.message || errorData?.error || "Setup failed");
      }

      const data = await res.json();
      if (data?.token) {
        persistFullAuthTokens({
          token: data.token,
          refresh_token: data.refresh_token,
        });
      }
      if (data?.retailer) {
        await setRetailerProfile(data.retailer);
      }

      clearTaxSetupDraft();
      window.location.href = "/dashboard";
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.billing_setup.error.setup_failed"));
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="setup-form-title">{t("retailer_desktop.setup.address.text.delivery_addresses")}</h1>
        <p className="setup-form-sub">{t("retailer_desktop.setup.address.text.where_should_we_bill_and_deliver_your_orders")}</p>
      </div>

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      <PortalField id="billing-address" label={t("retailer_desktop.residual.text.billing_address")}>
        <PortalInput
          id="billing-address"
          value={billingAddress}
          onChange={(e) => setBillingAddress(e.target.value)}
          autoComplete="street-address"
        />
      </PortalField>

      <PortalField id="shipping-address" label={t("retailer_desktop.residual.text.shipping_address_default")}>
        <PortalInput
          id="shipping-address"
          value={shippingAddress}
          onChange={(e) => setShippingAddress(e.target.value)}
          autoComplete="shipping street-address"
        />
      </PortalField>

      <div className="grid gap-4 sm:grid-cols-2">
        <PortalField id="city" label={t("supplier_portal.analytics.demand.signals.text.city")}>
          <PortalInput id="city" value={city} onChange={(e) => setCity(e.target.value)} />
        </PortalField>
        <PortalField id="postal" label={t("retailer_desktop.residual.text.postal_code")} optional>
          <PortalInput id="postal" value={postalCode} onChange={(e) => setPostalCode(e.target.value)} />
        </PortalField>
      </div>

      <PortalActions
        back={{ label: t("common.action.back"), onClick: () => router.push("/setup/tax") }}
        primary={{ label: submitting ? "Saving…" : "Complete setup", onClick: handleSubmit, loading: submitting }}
      />
    </div>
  );
}
