"use client";

import { useState } from "react";
import { supplierOrgMemberCreateKey } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { SupplierTopologyResponse, HomeNodeType, Role } from "@pegasusx/types";
import {
  defaultOrgForm,
  OrgFormState,
  orgRoleOptions,
  buildOrgMemberRequest,
  supplierScopeId,
  toErrorMessage,
  StatusText,
  isErrorMessage,
  orgEffectiveNodeType,
  nodeOptionsFor,
} from "./utils";

export function OrgMemberForm({
  topology,
  onCreated,
}: {
  topology: SupplierTopologyResponse;
  onCreated: () => void;
}) {
  const [orgForm, setOrgForm] = useState<OrgFormState>(defaultOrgForm);
  const [orgSubmitting, setOrgSubmitting] = useState(false);
  const [orgMessage, setOrgMessage] = useState<string | null>(null);

  const activeNodeOptions = nodeOptionsFor(orgEffectiveNodeType(orgForm), topology);

  async function submitOrgMember(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setOrgSubmitting(true);
    setOrgMessage(null);
    const request = buildOrgMemberRequest(orgForm);
    try {
      const api = createSupplierApi();
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
      <h2 className="md-typescale-title-large">Org members</h2>
      <p className="md-typescale-body-medium mt-2" style={{ color: "var(--color-md-outline)" }}>
        Create supplier operators, node admins, and payload staff with explicit node assignments.
      </p>
      <form className="grid gap-3 mt-4" onSubmit={submitOrgMember}>
        <input
          className="md-input-outlined"
          placeholder="Full name"
          value={orgForm.name}
          onChange={(event) => setOrgForm((current) => ({ ...current, name: event.target.value }))}
          disabled={orgSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder="Email"
          value={orgForm.email}
          onChange={(event) => setOrgForm((current) => ({ ...current, email: event.target.value }))}
          disabled={orgSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder="Phone"
          value={orgForm.phone}
          onChange={(event) => setOrgForm((current) => ({ ...current, phone: event.target.value }))}
          disabled={orgSubmitting}
        />
        <input
          className="md-input-outlined"
          placeholder="Temporary password"
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
            <option value="WAREHOUSE">Warehouse payload staff</option>
            <option value="FACTORY">Factory payload staff</option>
          </select>
        )}

        {orgForm.role !== "ADMIN" && (
          <select
            className="md-input-outlined"
            value={orgForm.nodeID}
            onChange={(event) => setOrgForm((current) => ({ ...current, nodeID: event.target.value }))}
            disabled={orgSubmitting}
          >
            <option value="">Select node</option>
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
