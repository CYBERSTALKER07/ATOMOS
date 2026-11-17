"use client";

import { usePortalT } from "@/lib/i18n";
import { Suspense, useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  PortalField,
  PortalInput,
  PortalActions,
  SelectionCard,
  FormAlert,
} from "@/components/portal";
import {
  readTaxSetupDraft,
  writeTaxSetupDraft,
  type RetailerTaxSetupMode,
} from "@/components/setup/constants";
import {
  approveRetailerJoinRequest,
  extractJoinRequestConflict,
  getRetailerProfileFromServer,
  listRetailerJoinRequests,
  lookupRetailerInn,
  rejectRetailerJoinRequest,
  updateRetailerTaxIdentity,
} from "@/lib/api";
import { getRetailerId, mergeRetailerProfile } from "@/lib/retailer-profile";
import type {
  INNLookupResponse,
  JoinRequest,
  JoinRequestConflictPayload,
} from "@pegasusx/types";

function SetupTaxContent() {
  const t = usePortalT();
  const router = useRouter();
  const searchParams = useSearchParams();
  const focusedRequestId = searchParams?.get("requestId") ?? "";

  const [mode, setMode] = useState<RetailerTaxSetupMode>("LEGAL_VERIFIED");
  const [taxId, setTaxId] = useState("");
  const [lookupLoading, setLookupLoading] = useState(false);
  const [lookupResult, setLookupResult] = useState<INNLookupResponse | null>(null);
  const [joinConflict, setJoinConflict] = useState<JoinRequestConflictPayload | null>(null);
  const [joinRequests, setJoinRequests] = useState<JoinRequest[]>([]);
  const [decidingId, setDecidingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadJoinRequests = useCallback(async () => {
    try {
      const res = await listRetailerJoinRequests("PENDING");
      setJoinRequests(Array.isArray(res.requests) ? res.requests : []);
    } catch {
      // Newly created stores before setup may not have pending join requests yet.
    }
  }, []);

  useEffect(() => {
    const draft = readTaxSetupDraft();
    if (draft) {
      setMode(draft.mode);
      if (draft.taxId) setTaxId(draft.taxId);
    }
    void getRetailerProfileFromServer().then((profile) => {
      if (!profile) return;
      if (!draft && profile.is_configured && profile.legal_entity_status === "INFORMAL_PHONE") {
        setMode("INFORMAL_PHONE");
      }
      if (!draft && profile.tax_id) {
        setMode("LEGAL_VERIFIED");
        setTaxId(profile.tax_id);
      }
    });
    void loadJoinRequests();
  }, [loadJoinRequests]);

  async function handleLookupInn() {
    const trimmed = taxId.trim();
    if (trimmed.length < 9) {
      setError(t("retailer_desktop.tax_identity.tax_id_label"));
      return;
    }
    setLookupLoading(true);
    setError(null);
    setJoinConflict(null);
    try {
      const res = await lookupRetailerInn(trimmed);
      setLookupResult(res);
      if (res.status === "UNAVAILABLE") {
        setError(t("retailer_desktop.tax_identity.registry_unavailable"));
      } else if (res.status === "NOT_FOUND") {
        setError(res.normalized_inn ? `${res.normalized_inn}: INN not found` : "INN not found");
      }
    } catch (err) {
      setLookupResult(null);
      setError(
        err instanceof Error
          ? err.message
          : t("retailer_desktop.tax_identity.registry_unavailable"),
      );
    } finally {
      setLookupLoading(false);
    }
  }

  async function handleContinue() {
    setError(null);
    setJoinConflict(null);

    const retailerId = getRetailerId() || "retailer";

    if (mode === "INFORMAL_PHONE") {
      setSaving(true);
      try {
        await updateRetailerTaxIdentity(
          { informal_shop: true, legal_entity_status: "INFORMAL_PHONE" },
          retailerId,
        );
        writeTaxSetupDraft({ mode: "INFORMAL_PHONE", taxId: "", persisted: true });
        router.push("/setup/address");
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("supplier_portal.billing_setup.error.setup_failed"),
        );
      } finally {
        setSaving(false);
      }
      return;
    }

    const trimmed = taxId.trim();
    if (trimmed.length < 9) {
      setError(t("retailer_desktop.tax_identity.tax_id_label"));
      return;
    }

    setSaving(true);
    try {
      const updated = await updateRetailerTaxIdentity(
        {
          tax_id: trimmed,
          inn: trimmed,
          legal_entity_status: "LEGAL_VERIFIED",
          ...(lookupResult?.legal_name ? { legal_name: lookupResult.legal_name } : {}),
        },
        retailerId,
      );
      const verifiedName =
        updated.name || lookupResult?.legal_name || undefined;
      await mergeRetailerProfile({
        retailer_id: updated.retailer_id || retailerId,
        ...(verifiedName ? { name: verifiedName, company: verifiedName } : {}),
      });
      writeTaxSetupDraft({
        mode: "LEGAL_VERIFIED",
        taxId: updated.tax_id || trimmed,
        legalName: verifiedName,
        persisted: true,
      });
      router.push("/setup/address");
    } catch (err) {
      const conflict = extractJoinRequestConflict(err);
      if (conflict) {
        setJoinConflict(conflict);
        return;
      }
      setError(
        err instanceof Error
          ? err.message
          : t("retailer_desktop.tax_identity.registry_unavailable"),
      );
    } finally {
      setSaving(false);
    }
  }

  async function handleJoinDecision(requestId: string, decision: "approve" | "reject") {
    setDecidingId(requestId);
    setError(null);
    try {
      if (decision === "approve") {
        await approveRetailerJoinRequest(requestId);
      } else {
        await rejectRetailerJoinRequest(requestId);
      }
      await loadJoinRequests();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update join request");
    } finally {
      setDecidingId(null);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="setup-form-title">{t("retailer_desktop.tax_identity.title")}</h1>
        <p className="setup-form-sub">{t("retailer_desktop.tax_identity.subtitle")}</p>
      </div>

      {error ? <FormAlert variant="error">{error}</FormAlert> : null}

      {joinConflict ? (
        <FormAlert variant="info">
          <div className="space-y-1">
            <p className="font-medium">
              {t("retailer_desktop.tax_identity.existing_retailer_join_hint")}
            </p>
            <p className="text-xs opacity-80">
              {joinConflict.legal_name} ({joinConflict.normalized_inn}) — #{joinConflict.request_id}
            </p>
          </div>
        </FormAlert>
      ) : null}

      <div className="setup-selection-grid setup-selection-grid--2">
        <SelectionCard
          selected={mode === "LEGAL_VERIFIED"}
          icon="verified"
          title={t("retailer_desktop.tax_identity.mode_legal_verified")}
          description={t("retailer_desktop.tax_identity.tax_id_label")}
          onClick={() => {
            setMode("LEGAL_VERIFIED");
            setError(null);
          }}
        />
        <SelectionCard
          selected={mode === "INFORMAL_PHONE"}
          icon="store"
          title={t("retailer_desktop.tax_identity.mode_informal_phone")}
          description={t("retailer_desktop.tax_identity.legal_status_informal")}
          onClick={() => {
            setMode("INFORMAL_PHONE");
            setLookupResult(null);
            setJoinConflict(null);
            setError(null);
          }}
        />
      </div>

      {mode === "LEGAL_VERIFIED" ? (
        <div className="space-y-4">
          <PortalField
            id="tax-id"
            label={t("retailer_desktop.tax_identity.tax_id_label")}
          >
            <div className="flex gap-2">
              <PortalInput
                id="tax-id"
                value={taxId}
                onChange={(e) => {
                  setTaxId(e.target.value);
                  setLookupResult(null);
                }}
                placeholder={t("retailer_desktop.setup.tax.text.e_g_123456789")}
                autoComplete="off"
              />
              <button
                type="button"
                onClick={() => void handleLookupInn()}
                disabled={lookupLoading || saving}
                className="portal-btn portal-btn--outline shrink-0"
              >
                {lookupLoading ? "..." : t("supplier_portal.onboarding_inn.verify_button")}
              </button>
            </div>
          </PortalField>

          {lookupResult?.status === "FOUND" ? (
            <FormAlert variant="info">
              <div className="space-y-1">
                <p className="font-medium">
                  {t("retailer_desktop.tax_identity.legal_status_verified").replace(
                    "{legal_name}",
                    lookupResult.legal_name || lookupResult.normalized_inn,
                  )}
                </p>
                {lookupResult.address ? (
                  <p className="text-xs opacity-80">{lookupResult.address}</p>
                ) : null}
                {lookupResult.existing_join_eligible ? (
                  <p className="text-xs font-medium">
                    {t("retailer_desktop.tax_identity.existing_retailer_join_hint")}
                  </p>
                ) : null}
              </div>
            </FormAlert>
          ) : null}
        </div>
      ) : (
        <FormAlert variant="info">
          {t("retailer_desktop.tax_identity.legal_status_informal")}
        </FormAlert>
      )}

      {joinRequests.length > 0 ? (
        <div className="rounded-xl border border-border bg-card p-4 space-y-3">
          <div className="text-sm font-semibold">
            {t("supplier_portal.join_requests.inbox_title")}
          </div>
          {joinRequests.map((jr) => (
            <div
              key={jr.request_id}
              className={`flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3 text-xs ${
                focusedRequestId && jr.request_id === focusedRequestId
                  ? "border-primary bg-primary/5"
                  : "border-border"
              }`}
            >
              <div>
                <div className="font-medium">
                  {jr.requester_name || jr.requester_phone}
                </div>
                <div className="opacity-75">
                  {jr.requester_phone} · {jr.normalized_inn} · {jr.role_sought}
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  disabled={decidingId === jr.request_id}
                  onClick={() => void handleJoinDecision(jr.request_id, "approve")}
                  className="portal-btn portal-btn--primary"
                >
                  {t("supplier_portal.join_requests.approve_button")}
                </button>
                <button
                  type="button"
                  disabled={decidingId === jr.request_id}
                  onClick={() => void handleJoinDecision(jr.request_id, "reject")}
                  className="portal-btn portal-btn--outline"
                >
                  {t("supplier_portal.join_requests.reject_button")}
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : null}

      <PortalActions
        primary={{
          label:
            mode === "INFORMAL_PHONE"
              ? t("retailer_desktop.tax_identity.save_informal")
              : t("retailer_desktop.tax_identity.verify_and_save"),
          onClick: () => void handleContinue(),
          loading: saving,
        }}
      />
    </div>
  );
}

export default function SetupTaxPage() {
  return (
    <Suspense fallback={null}>
      <SetupTaxContent />
    </Suspense>
  );
}
