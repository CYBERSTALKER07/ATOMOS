import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import { messageCatalogs } from "@pegasusx/i18n";

const { mockRequest, mockSearchParams } = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockSearchParams: new URLSearchParams(),
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => (key: string, values?: Record<string, string | number>) => {
    if (!values) return key;
    return `${key}:${JSON.stringify(values)}`;
  },
}));

vi.mock("@/lib/use-supplier-session-reconcile", () => ({
  useSupplierSessionReconcile: () => {},
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => mockSearchParams,
}));

vi.mock("@/components/PageChrome", () => ({
  PageChrome: ({
    title,
    description,
    actions,
    loading,
    error,
    children,
  }: {
    title: string;
    description?: string;
    actions?: ReactNode;
    loading?: boolean;
    error?: string | null;
    children: ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      {description ? <p>{description}</p> : null}
      {actions}
      {loading ? <p>loading</p> : error ? <p role="alert">{error}</p> : children}
    </div>
  ),
}));

vi.mock("@/components/StatusBadge", () => ({
  default: ({ state }: { state: string }) => <span>{state}</span>,
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import { createSupplierAuditIamApi } from "@/lib/audit-iam-api";
import { ObjectHistoryPanel } from "@/components/audit/ObjectHistoryPanel";
import SupplierGovernancePage from "@/app/(portal)/governance/page";
import { NAV, allNavHrefs } from "@/lib/nav";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");

describe("M0.4 Supplier Portal Audit v2, ObjectHistoryPanel, RBAC/SoD & Governance", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls all M0.4 Audit and IAM routes with proper query params and headers via ApiClient", async () => {
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      return new Response(JSON.stringify({ ok: true, url: String(input), method: init?.method }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-m04",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const client = createSupplierAuditIamApi(realClient);

    await client.listEvents({ object: "order", object_id: "ord-1", actor_type: "agent", date: "2026-11-16", limit: 25 });
    await client.getObjectHistory("order", "ord-1", { limit: 50 });
    await client.verifyDayChain({ date: "2026-11-16" });
    await client.listAnchors({ date: "2026-11-16", limit: 10 });
    await client.writeAnchor({ date: "2026-11-16" }, "idem-anchor-1");
    await client.listLegalHolds({ status: "ACTIVE", limit: 20 });
    await client.createLegalHold(
      { object_type: "order", object_id: "ord-1", reason: "dispute" },
      "idem-hold-1",
    );
    await client.listRoles();
    await client.listPermissions();
    await client.assignUserRole("usr-2", { role_name: "FINANCE_MANAGER", action: "ASSIGN" }, "idem-role-1");
    await client.listSodPolicies();
    await client.updateSodPolicy("pol-1", { enabled: false }, "idem-pol-1");
    await client.listSodViolations({ actor: "usr-2", limit: 20 });

    const calls = fetchSpy.mock.calls.map(([url, init]) => ({
      url: String(url),
      method: init?.method,
      headers: init?.headers as Headers,
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
    }));

    expect(calls[0].url).toBe(
      "https://api.pegasusx.test/v1/audit/events?object=order&object_id=ord-1&actor_type=agent&date=2026-11-16&limit=25",
    );
    expect(calls[1].url).toBe("https://api.pegasusx.test/v1/audit/history/order/ord-1?limit=50");
    expect(calls[2].url).toBe("https://api.pegasusx.test/v1/audit/verify?date=2026-11-16");
    expect(calls[3].url).toBe("https://api.pegasusx.test/v1/audit/anchors?date=2026-11-16&limit=10");
    expect(calls[4].url).toBe("https://api.pegasusx.test/v1/audit/anchors");
    expect(calls[4].method).toBe("POST");
    expect(calls[4].headers.get("Idempotency-Key")).toBe("idem-anchor-1");
    expect(calls[5].url).toBe("https://api.pegasusx.test/v1/audit/legal-holds?status=ACTIVE&limit=20");
    expect(calls[6].url).toBe("https://api.pegasusx.test/v1/audit/legal-hold");
    expect(calls[6].method).toBe("POST");
    expect(calls[6].body).toEqual({ object_type: "order", object_id: "ord-1", reason: "dispute" });
    expect(calls[7].url).toBe("https://api.pegasusx.test/v1/iam/roles");
    expect(calls[8].url).toBe("https://api.pegasusx.test/v1/iam/permissions");
    expect(calls[9].url).toBe("https://api.pegasusx.test/v1/iam/users/usr-2/roles");
    expect(calls[9].method).toBe("POST");
    expect(calls[9].body).toEqual({ role_name: "FINANCE_MANAGER", action: "ASSIGN" });
    expect(calls[10].url).toBe("https://api.pegasusx.test/v1/iam/sod-policies");
    expect(calls[11].url).toBe("https://api.pegasusx.test/v1/iam/sod-policies/pol-1");
    expect(calls[11].method).toBe("PUT");
    expect(calls[11].headers.get("Idempotency-Key")).toBe("idem-pol-1");
    expect(calls[12].url).toBe("https://api.pegasusx.test/v1/iam/sod-violations?actor=usr-2&limit=20");
  });

  it("renders ObjectHistoryPanel with ActorType badges, TraceId, PII masking note, payload diff, and chain verification", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/audit/history/order/ord-42") && method === "GET") {
        return {
          tenant_id: "sup-1",
          object_type: "order",
          object_id: "ord-42",
          pii_masked: true,
          count: 2,
          events: [
            {
              tenant_id: "sup-1",
              day_bucket: "2026-11-16",
              event_id: "aud-2",
              seq_num: 2,
              actor_id: "agent-replenish-1",
              actor_type: "agent",
              role: "AGENT",
              action: "PROPOSE_ADJUSTMENT",
              object_type: "order",
              object_id: "ord-42",
              after_json: '{"phone":"***masked***","amount_minor":50000}',
              trace_id: "trc-agent-2",
              prev_hash: "11112222333344445555",
              entry_hash: "aaaaaabbbbbbccccccdd",
              legal_hold: true,
              legal_hold_reason: "tax audit",
              created_at: "2026-11-16T10:05:00Z",
            },
            {
              tenant_id: "sup-1",
              day_bucket: "2026-11-16",
              event_id: "aud-1",
              seq_num: 1,
              actor_id: "user-human-1",
              actor_type: "human",
              role: "FINANCE_MANAGER",
              action: "CREATE_ORDER",
              object_type: "order",
              object_id: "ord-42",
              trace_id: "trc-human-1",
              prev_hash: "00000000000000000000",
              entry_hash: "11112222333344445555",
              legal_hold: false,
              created_at: "2026-11-16T10:00:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/audit/verify") && method === "GET") {
        return {
          tenant_id: "sup-1",
          day_bucket: "2026-11-16",
          valid: true,
          event_count: 2,
          root_hash: "aaaaaabbbbbbccccccdd",
          anchor_matched: true,
        };
      }
      return {};
    });

    render(<ObjectHistoryPanel objectType="order" objectId="ord-42" autoVerifyChain />);

    await waitFor(() => {
      expect(screen.getByText("PROPOSE_ADJUSTMENT")).toBeInTheDocument();
    });
    expect(screen.getByText("supplier_portal.audit.pii_masked_note")).toBeInTheDocument();
    expect(screen.getByText("supplier_portal.audit.actor_agent")).toBeInTheDocument();
    expect(screen.getByText("supplier_portal.audit.actor_human")).toBeInTheDocument();
    expect(screen.getByText(/trc-agent-2/)).toBeInTheDocument();
    expect(screen.getByText("supplier_portal.audit.legal_hold_badge")).toBeInTheDocument();
    expect(screen.getByTestId("chain-verify-badge")).toHaveTextContent(
      "supplier_portal.audit.chain_valid",
    );

    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.audit.show_payload" }));
    expect(screen.getByText(/\*\*\*masked\*\*\*/)).toBeInTheDocument();
  });

  it("handles 403 audit permission error gracefully in ObjectHistoryPanel", async () => {
    mockRequest.mockRejectedValue(new ApiError("missing_permission", 403, { detail: "missing_permission" }));

    render(<ObjectHistoryPanel objectType="order" objectId="ord-403" />);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("supplier_portal.audit.permission_denied");
    });
  });

  it("renders SupplierGovernancePage tabs, places legal hold, assigns role, surfaces SoD 403, and toggles SoD policy", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/audit/verify") && method === "GET") {
        return {
          tenant_id: "sup-1",
          day_bucket: "2026-11-16",
          valid: true,
          event_count: 3,
          root_hash: "root1234567890abcdef",
          anchored_root_hash: "root1234567890abcdef",
          anchor_matched: true,
        };
      }
      if (path.startsWith("/v1/audit/anchors") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          anchors: [
            {
              tenant_id: "sup-1",
              day_bucket: "2026-11-16",
              head_event_id: "aud-3",
              event_count: 3,
              root_hash: "root1234567890abcdef",
              worm_uri: "file:///worm/sup-1/2026-11-16.json",
              anchored_by: "system",
              anchored_at: "2026-11-16T23:59:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/audit/events") && method === "GET") {
        return {
          tenant_id: "sup-1",
          pii_masked: false,
          count: 1,
          events: [
            {
              tenant_id: "sup-1",
              day_bucket: "2026-11-16",
              event_id: "aud-3",
              seq_num: 3,
              actor_id: "fin-1",
              actor_type: "human",
              action: "POST /v1/finance/journal/manual",
              object_type: "journal",
              object_id: "gle-1",
              trace_id: "trc-gl-1",
              prev_hash: "prev123",
              entry_hash: "root1234567890abcdef",
              legal_hold: false,
              created_at: "2026-11-16T12:00:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/audit/legal-holds") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, holds: [] };
      }
      if (path === "/v1/audit/legal-hold" && method === "POST") {
        return {
          tenant_id: "sup-1",
          hold_id: "hold-99",
          object_type: "order",
          object_id: "ord-99",
          reason: "litigation hold",
          status: "ACTIVE",
          created_by: "owner-1",
          created_at: "2026-11-16T13:00:00Z",
        };
      }
      if (path.startsWith("/v1/iam/roles") && method === "GET") {
        return {
          tenant_id: "sup-1",
          roles: [
            {
              role_name: "FINANCE_MANAGER",
              description: "Approves GL journals and payouts",
              read_only: false,
              agent_role: false,
              permissions: ["journal.post", "journal.approve", "audit.read"],
            },
            {
              role_name: "AUDITOR",
              description: "Read-only auditor",
              read_only: true,
              agent_role: false,
              permissions: ["audit.read"],
            },
          ],
          user_roles: [],
        };
      }
      if (path === "/v1/iam/permissions" && method === "GET") {
        return {
          count: 2,
          permissions: [
            { key: "journal.approve", category: "GL", description: "Approve journal" },
            { key: "audit.read", category: "Audit", description: "Read audit log" },
          ],
        };
      }
      if (path.startsWith("/v1/iam/users/") && method === "POST") {
        throw new ApiError("sod_violation", 403, {
          error: "sod_violation",
          code: "sod_role_conflict",
          policy_id: "pol-gl",
        });
      }
      if (path.startsWith("/v1/iam/sod-policies") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          policies: [
            {
              tenant_id: "sup-1",
              policy_id: "pol-gl",
              name: "Journal Maker vs Checker",
              permission_a: "journal.post",
              permission_b: "journal.approve",
              enforce_same_object: true,
              enabled: true,
              created_at: "2026-11-16T08:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/iam/sod-policies/pol-gl" && method === "PUT") {
        return {
          policy: {
            tenant_id: "sup-1",
            policy_id: "pol-gl",
            name: "Journal Maker vs Checker",
            permission_a: "journal.post",
            permission_b: "journal.approve",
            enforce_same_object: true,
            enabled: false,
            created_at: "2026-11-16T08:00:00Z",
          },
        };
      }
      if (path.startsWith("/v1/iam/sod-violations") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          violations: [
            {
              tenant_id: "sup-1",
              violation_id: "vio-1",
              actor_id: "usr-self",
              maker_id: "usr-self",
              action: "APPROVE_DRAFT",
              object_type: "FinancialDraft",
              object_id: "drf-1",
              rule_code: "FOUR_EYES_SELF_APPROVAL",
              trace_id: "trc-vio-1",
              created_at: "2026-11-16T11:00:00Z",
            },
          ],
        };
      }
      return {};
    });

    render(<SupplierGovernancePage />);

    await waitFor(() => {
      expect(screen.getByText("supplier_portal.governance.chain_valid_badge")).toBeInTheDocument();
    });
    expect(screen.getByText(/file:\/\/\/worm\/sup-1\/2026-11-16\.json/)).toBeInTheDocument();

    // Legal holds tab
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.governance.tab_holds" }));
    fireEvent.change(screen.getByLabelText("supplier_portal.governance.object_id_label"), {
      target: { value: "ord-99" },
    });
    fireEvent.change(screen.getByLabelText("supplier_portal.governance.hold_reason_label"), {
      target: { value: "litigation hold" },
    });
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.governance.place_hold_button" }));

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/audit/legal-hold",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            object_type: "order",
            object_id: "ord-99",
            reason: "litigation hold",
          }),
        }),
      );
    });

    // RBAC Roles tab -> test SoD conflict alert
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.governance.tab_roles" }));
    expect(screen.getByText("journal.approve")).toBeInTheDocument();
    expect(screen.getByText("supplier_portal.governance.read_only_badge")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("supplier_portal.governance.user_id_label"), {
      target: { value: "usr-conflict" },
    });
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.governance.submit_role_button" }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("sod_role_conflict");
    });

    // SoD & Violations tab
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.governance.tab_sod" }));
    expect(screen.getByText("Journal Maker vs Checker")).toBeInTheDocument();
    expect(screen.getByText("FOUR_EYES_SELF_APPROVAL")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.governance.toggle_disable" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/iam/sod-policies/pol-gl",
        "PUT",
        expect.objectContaining({
          body: expect.objectContaining({ enabled: false }),
        }),
      );
    });
  });

  it("registers /governance in nav, localizes all new i18n keys across en/ru/uz, and imports zero @pegasusx/ui", () => {
    expect(allNavHrefs(NAV)).toContain("/governance");

    const requiredKeys = [
      "portal.nav.governance",
      "supplier_portal.audit.panel_title",
      "supplier_portal.audit.pii_masked_note",
      "supplier_portal.audit.chain_valid",
      "supplier_portal.governance.title",
      "supplier_portal.governance.tab_verify",
      "supplier_portal.governance.tab_holds",
      "supplier_portal.governance.tab_roles",
      "supplier_portal.governance.tab_sod",
      "supplier_portal.governance.sod_policies_title",
    ];
    for (const key of requiredKeys) {
      expect(messageCatalogs.en[key]).toBeTruthy();
      expect(messageCatalogs.ru[key]).toBeTruthy();
      expect(messageCatalogs.uz[key]).toBeTruthy();
    }

    const filesToCheck = [
      "lib/audit-iam-api.ts",
      "components/audit/ObjectHistoryPanel.tsx",
      "app/(portal)/governance/page.tsx",
      "app/(portal)/orders/[id]/page.tsx",
      "app/(portal)/credit/collections/page.tsx",
      "app/(portal)/credit/policy/page.tsx",
      "app/(portal)/finance/ledger/page.tsx",
      "app/(portal)/finance/reconciliation/page.tsx",
    ];
    for (const rel of filesToCheck) {
      const fullPath = join(portalRoot, rel);
      expect(existsSync(fullPath), rel).toBe(true);
      const src = readFileSync(fullPath, "utf8");
      expect(src).not.toContain("@pegasusx/ui");
    }
    const govSrc = readFileSync(join(portalRoot, "app/(portal)/governance/page.tsx"), "utf8");
    expect(govSrc).toContain("<Suspense");
  });
});
