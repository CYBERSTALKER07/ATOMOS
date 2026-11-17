import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "../api";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");

describe("M0.4 Admin Portal Governance Audit v2 & SoD API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("queries daily chain anchors, chain verification, cross-tenant SoD violations, policies, and legal holds with auth & idempotency headers", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      return {
        ok: true,
        status: 200,
        json: async () => ({
          ok: true,
          url: String(input),
          method: init?.method,
        }),
      } as unknown as Response;
    });
    vi.stubGlobal("fetch", fetchMock);

    await api.listAuditAnchors("admin-tok", { date: "2026-11-16", limit: 50 });
    await api.verifyAuditChain("admin-tok", "sup-1", "2026-11-16");
    await api.writeAuditAnchor("admin-tok", "sup-1", "2026-11-16");
    await api.listAuditLegalHolds("admin-tok", { status: "ACTIVE", limit: 25 });
    await api.createAuditLegalHold("admin-tok", {
      entity_id: "sup-1",
      object_type: "order",
      object_id: "ord-1",
      reason: "tax audit",
    });
    await api.listAuditV2Events("admin-tok", { actor_type: "agent", date: "2026-11-16", limit: 30 });
    await api.listIamRoles("admin-tok", "sup-1");
    await api.listSodPolicies("admin-tok", "sup-1");
    await api.updateSodPolicy("admin-tok", "pol-1", { entity_id: "sup-1", enabled: false });
    await api.listSodViolations("admin-tok", { actor: "usr-1", limit: 40 });

    expect(fetchMock).toHaveBeenCalledTimes(10);

    const calls = fetchMock.mock.calls.map(([url, init]) => ({
      url: String(url),
      method: init?.method,
      headers: init?.headers as Record<string, string>,
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
    }));

    expect(calls[0].url).toBe("http://localhost:8080/v1/audit/anchors?date=2026-11-16&limit=50");
    expect(calls[0].headers.Authorization).toBe("Bearer admin-tok");

    expect(calls[1].url).toBe("http://localhost:8080/v1/audit/verify?entity=sup-1&date=2026-11-16");

    expect(calls[2].url).toBe("http://localhost:8080/v1/audit/anchors");
    expect(calls[2].method).toBe("POST");
    expect(calls[2].body).toEqual({ entity_id: "sup-1", date: "2026-11-16" });
    expect(calls[2].headers["Idempotency-Key"]).toBeTruthy();

    expect(calls[3].url).toBe("http://localhost:8080/v1/audit/legal-holds?status=ACTIVE&limit=25");

    expect(calls[4].url).toBe("http://localhost:8080/v1/audit/legal-hold");
    expect(calls[4].method).toBe("POST");
    expect(calls[4].body).toEqual({
      entity_id: "sup-1",
      object_type: "order",
      object_id: "ord-1",
      reason: "tax audit",
    });

    expect(calls[5].url).toBe("http://localhost:8080/v1/audit/events?actor_type=agent&date=2026-11-16&limit=30");
    expect(calls[6].url).toBe("http://localhost:8080/v1/iam/roles?entity=sup-1");
    expect(calls[7].url).toBe("http://localhost:8080/v1/iam/sod-policies?entity=sup-1");

    expect(calls[8].url).toBe("http://localhost:8080/v1/iam/sod-policies/pol-1");
    expect(calls[8].method).toBe("PUT");
    expect(calls[8].body).toEqual({ entity_id: "sup-1", enabled: false });
    expect(calls[8].headers["Idempotency-Key"]).toBeTruthy();

    expect(calls[9].url).toBe("http://localhost:8080/v1/iam/sod-violations?actor=usr-1&limit=40");
  });

  it("propagates backend errors honestly on chain verify and SoD calls", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: false,
        status: 403,
        json: async () => ({ error: "missing_permission" }),
      })),
    );

    await expect(api.verifyAuditChain("admin-tok", "sup-1", "2026-11-16")).rejects.toThrow(
      "missing_permission",
    );
  });

  it("wires app/governance/page.tsx and AdminSidebar without @pegasusx/ui imports and wraps useSearchParams in Suspense", () => {
    const govPath = join(portalRoot, "app/governance/page.tsx");
    const sidebarPath = join(portalRoot, "components/layout/AdminSidebar.tsx");
    expect(existsSync(govPath)).toBe(true);
    expect(existsSync(sidebarPath)).toBe(true);

    const govSrc = readFileSync(govPath, "utf8");
    expect(govSrc).not.toContain("@pegasusx/ui");
    expect(govSrc).toContain("<Suspense");
    expect(govSrc).toContain("verifyAuditChain");
    expect(govSrc).toContain("listAuditAnchors");
    expect(govSrc).toContain("listAuditLegalHolds");
    expect(govSrc).toContain("listSodViolations");

    const sidebarSrc = readFileSync(sidebarPath, "utf8");
    expect(sidebarSrc).toContain('href: "/governance"');
  });
});
