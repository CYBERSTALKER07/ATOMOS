"use client";

import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import type { Role, HomeNodeType } from "@pegasusx/types";
import { supplierOrgMemberCreateKey } from "@pegasusx/api-client";
import type { ApiClient } from "@pegasusx/api-client";
import {
  OrgFormState,
  defaultOrgForm,
  orgRoleOptions,
  buildOrgMemberRequest,
  StatusText,
  isErrorMessage,
  toErrorMessage,
  supplierScopeId,
  nodeOptionsFor,
  orgEffectiveNodeType,
  ReadyState,
} from "./utils";

export function OrgMemberForm({
  state,
  api,
  onCreated,
}: {
  state: ReadyState;
  api: ApiClient;
  onCreated: () => void;
}) {
  const [orgForm, setOrgForm] = useState<OrgFormState>(defaultOrgForm);
  const [orgSubmitting, setOrgSubmitting] = useState(false);
  const [orgMessage, setOrgMessage] = useState<string | null>(null);

  const activeNodeOptions = nodeOptionsFor(orgEffectiveNodeType(orgForm), state.topology);

  async function submitOrgMember(event: React.FormEvent<HTMLFormElement>) {
  const t = usePortalT();
    event.preventDefault();
    setOrgSubmitting(true);
    setOrgMessage(null);
    const request = buildOrgMemberRequest(orgForm);
    try {
      await api.createSupplierOrgMember(
        request,
        supplierOrgMemberCreateKey(supplierScopeId(), request.phone),
      );
      setOrgForm(defaultOrgForm);
      setOrgMessage("Org member created.");
      onCreated();
    } catch (error) {
      setOrgMessage(toErrorMessage(error));
    } finally {
      setOrgSubmitting(false);
    }
  }

  return (
    <article className="md-card md-shape-md p-6">
      <h2 className="md-typescale-title-large">{t("supplier_portal.org_fleet.components.org_member_form.text.org_members")}</h2>
      <p className="md-typescale-body-medium mt-2" style={{ color: "var(--color-md-outline)" }}>
        Create supplier operators, node admins, and payload staff with explicit node assignments.
      </p>
      <form className="grid gap-3 mt-4" onSubmit={submitOrgMember}>
        <input
          className="md-input-outlined"
          placeholder={t("supplier_portal.org_fleet.components.org_member_form.text.full_name")}
          value={orgForm.name}
          onChange={(event) => setOrgForm((current) => ({ ...current, name: event.target.value }))}
          disabled={orgSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder={t("supplier_portal.auth.login.email_label")}
          value={orgForm.email}
          onChange={(event) => setOrgForm((current) => ({ ...current, email: event.target.value }))}
          disabled={orgSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder={t("common.field.phone")}
          value={orgForm.phone}
          onChange={(event) => setOrgForm((current) => ({ ...current, phone: event.target.value }))}
          disabled={orgSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder={t("supplier_portal.org_fleet.components.org_member_form.text.temporary_password")}
          type="password"
          value={orgForm.password}
          onChange={(event) => setOrgForm((current) => ({ ...current, password: event.target.value }))}
          disabled={orgSubmitting}
        />
        <select
          className="md-input-outlined"
          value={orgForm.role}
          onChange={(event) =>
            setOrgForm((current) => ({
              ...current,
              role: event.target.value as Role,
              nodeType: event.target.value === "FACTORY_ADMIN" ? "FACTORY" : current.nodeType,
              nodeID: "",
            }))
          }
          disabled={orgSubmitting}
        >
          {orgRoleOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>

        {orgForm.role === "PAYLOAD" && (
          <select
            className="md-input-outlined"
            value={orgForm.nodeType}
            onChange={(event) =>
              setOrgForm((current) => ({
                ...current,
                nodeType: event.target.value as HomeNodeType,
                nodeID: "",
              }))
            }
            disabled={orgSubmitting}
          >
            <option value="WAREHOUSE">{t("supplier_portal.org_fleet.components.org_member_form.text.warehouse_payload_staff")}</option>
            <option value="FACTORY">{t("supplier_portal.org_fleet.components.org_member_form.text.factory_payload_staff")}</option>
          </select>
        )}

        {orgForm.role !== "ADMIN" && (
          <select
            className="md-input-outlined"
            value={orgForm.nodeID}
            onChange={(event) => setOrgForm((current) => ({ ...current, nodeID: event.target.value }))}
            disabled={orgSubmitting}
          >
            <option value="">{t("supplier_portal.org_fleet.components.org_member_form.text.select_node")}</option>
            {activeNodeOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        )}

        <button className="md-btn md-btn-filled" type="submit" disabled={orgSubmitting}>
          {orgSubmitting ? "Creating member..." : "Create org member"}
        </button>
        {orgMessage && <StatusText message={orgMessage} isError={isErrorMessage(orgMessage)} />}
      </form>
    </article>
  );
}
