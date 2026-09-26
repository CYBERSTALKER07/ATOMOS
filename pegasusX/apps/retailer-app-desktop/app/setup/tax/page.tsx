"use client";

import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { PortalField, PortalInput, PortalActions, FormAlert } from "@/components/portal";
import { SETUP_TAX_KEY } from "@/components/setup/constants";

export default function SetupTaxPage() {
  const t = usePortalT();
  const router = useRouter();
  const [taxId, setTaxId] = useState("");
  const [error, setError] = useState<string | null>(null);

  function handleContinue() {
    if (taxId.trim().length < 5) {
      setError(t("retailer_desktop.residual.text.tax_id_is_required_minimum_5_characters"));
      return;
    }
    setError(null);
    sessionStorage.setItem(SETUP_TAX_KEY, taxId.trim());
    router.push("/setup/address");
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="setup-form-title">{t("retailer_desktop.setup.tax.text.business_tax_id")}</h1>
        <p className="setup-form-sub">{t("retailer_desktop.setup.tax.text.provide_your_business_identity_for_invoicing_and_compliance")}</p>
      </div>

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      <PortalField id="tax-id" label={t("retailer_desktop.residual.text.tax_id_vat")}>
        <PortalInput
          id="tax-id"
          value={taxId}
          onChange={(e) => setTaxId(e.target.value)}
          placeholder={t("retailer_desktop.setup.tax.text.e_g_123456789")}
          autoComplete="off"
        />
      </PortalField>

      <PortalActions
        primary={{ label: t("supplier_portal.bulk_import_wizard.text.continue"), onClick: handleContinue }}
      />
    </div>
  );
}
