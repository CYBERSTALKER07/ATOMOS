"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PortalField, PortalInput, PortalActions, FormAlert } from "@/components/portal";
import { SETUP_TAX_KEY } from "@/components/setup/constants";

export default function SetupTaxPage() {
  const router = useRouter();
  const [taxId, setTaxId] = useState("");
  const [error, setError] = useState<string | null>(null);

  function handleContinue() {
    if (taxId.trim().length < 5) {
      setError("Tax ID is required (minimum 5 characters).");
      return;
    }
    setError(null);
    sessionStorage.setItem(SETUP_TAX_KEY, taxId.trim());
    router.push("/setup/address");
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="setup-form-title">Business tax ID</h1>
        <p className="setup-form-sub">Provide your business identity for invoicing and compliance.</p>
      </div>

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      <PortalField id="tax-id" label="Tax ID / VAT">
        <PortalInput
          id="tax-id"
          value={taxId}
          onChange={(e) => setTaxId(e.target.value)}
          placeholder="e.g. 123456789"
          autoComplete="off"
        />
      </PortalField>

      <PortalActions
        primary={{ label: "Continue", onClick: handleContinue }}
      />
    </div>
  );
}
