"use client";

import { useState } from "react";
import { supplierOrgMemberDeactivateKey, supplierOrgMemberUpdateKey } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { Role, HomeNodeType, SupplierTopologyResponse } from "@pegasusx/types";
import {
  ReadyState,
  supplierScopeId,
  toErrorMessage,
  StatusText,
  isErrorMessage,
  describeMemberNode,
  formatRole,
  orgRoleOptions,
} from "./utils";

export function OrgMemberTable({
  orgMembers,
  topology,
  onUpdated,
}: {
  orgMembers: ReadyState["orgMembers"];
  topology: SupplierTopologyResponse;
  onUpdated: () => void;
}) {
  const [editingMemberId, setEditingMemberId] = useState<string | null>(null);
  const [memberActionId, setMemberActionId] = useState<string | null>(null);
  const [orgMessage, setOrgMessage] = useState<string | null>(null);

  async function deactivateOrgMember(userId: string) {
    setMemberActionId(userId);
    setOrgMessage(null);
    try {
      const api = createSupplierApi();
      await api.deactivateSupplierOrgMember(
        userId,
        supplierOrgMemberDeactivateKey(supplierScopeId(), userId),
      );
      setOrgMessage("Org member deactivated.");
      onUpdated();
    } catch (error) {
      setOrgMessage(toErrorMessage(error));
    } finally {
      setMemberActionId(null);
    }
  }

  async function saveOrgMemberEdit(userId: string, role: Role, nodeType: HomeNodeType, nodeID: string) {
    setMemberActionId(userId);
    setOrgMessage(null);
    const request: import("@pegasusx/types").SupplierOrgMemberUpdateRequest = {
      supplier_role: role,
    };
    if (role === "WAREHOUSE_ADMIN") {
      request.assigned_warehouse_id = nodeID;
    } else if (role === "FACTORY_ADMIN") {
      request.assigned_factory_id = nodeID;
    } else if (role === "PAYLOAD") {
      if (nodeType === "WAREHOUSE") {
        request.assigned_warehouse_id = nodeID;
      } else {
        request.assigned_factory_id = nodeID;
      }
    }
    try {
      const api = createSupplierApi();
      const revision = `${role}:${nodeType}:${nodeID}`;
      await api.updateSupplierOrgMember(
        userId,
        request,
        supplierOrgMemberUpdateKey(supplierScopeId(), userId, revision),
      );
      setEditingMemberId(null);
      setOrgMessage("Org member updated.");
      onUpdated();
    } catch (error) {
      setOrgMessage(toErrorMessage(error));
    } finally {
      setMemberActionId(null);
    }
  }

  return (
    <article className="md-card md-shape-md p-6 overflow-x-auto">
      <h2 className="md-typescale-title-large">Current org roster</h2>
      {orgMessage && (
        <div className="mb-2 mt-2">
            <StatusText message={orgMessage} isError={isErrorMessage(orgMessage)} />
        </div>
      )}
      {orgMembers.length === 0 ? (
        <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
          No org members have been created yet.
        </p>
      ) : (
        <table className="w-full text-left mt-4">
          <thead>
            <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
              <th className="py-2 pr-4">Name</th>
              <th className="py-2 pr-4">Role</th>
              <th className="py-2 pr-4">Node</th>
              <th className="py-2 pr-4">Phone</th>
              <th className="py-2 pr-4">Status</th>
              <th className="py-2 pr-4">Actions</th>
            </tr>
          </thead>
          <tbody>
            {orgMembers.map((member) => (
              <tr key={member.user_id} className="md-typescale-body-medium">
                <td className="py-2 pr-4">{member.name}</td>
                <td className="py-2 pr-4">
                  {editingMemberId === member.user_id ? (
                    <select
                      className="md-input-outlined"
                      defaultValue={member.supplier_role}
                      id={`role-${member.user_id}`}
                    >
                      {orgRoleOptions.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                  ) : (
                    formatRole(member.supplier_role)
                  )}
                </td>
                <td className="py-2 pr-4">{describeMemberNode(member, topology)}</td>
                <td className="py-2 pr-4">{member.phone}</td>
                <td className="py-2 pr-4">{member.is_active ? "Active" : "Inactive"}</td>
                <td className="py-2 pr-4">
                  <div className="flex flex-wrap gap-2">
                    {editingMemberId === member.user_id ? (
                      <>
                        <button
                          type="button"
                          className="md-btn md-btn-tonal md-typescale-label-medium"
                          disabled={memberActionId === member.user_id}
                          onClick={() => {
                            const role = document.getElementById(`role-${member.user_id}`) as HTMLSelectElement;
                            void saveOrgMemberEdit(
                              member.user_id,
                              role.value as Role,
                              member.assigned_factory_id ? "FACTORY" : "WAREHOUSE",
                              member.assigned_warehouse_id ?? member.assigned_factory_id ?? "",
                            );
                          }}
                        >
                          Save
                        </button>
                        <button
                          type="button"
                          className="md-btn md-btn-outlined md-typescale-label-medium"
                          onClick={() => setEditingMemberId(null)}
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          type="button"
                          className="md-btn md-btn-outlined md-typescale-label-medium"
                          disabled={!member.is_active || memberActionId === member.user_id}
                          onClick={() => setEditingMemberId(member.user_id)}
                        >
                          Edit role
                        </button>
                        <button
                          type="button"
                          className="md-btn md-btn-outlined md-typescale-label-medium"
                          disabled={!member.is_active || memberActionId === member.user_id}
                          onClick={() => void deactivateOrgMember(member.user_id)}
                        >
                          Deactivate
                        </button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </article>
  );
}
