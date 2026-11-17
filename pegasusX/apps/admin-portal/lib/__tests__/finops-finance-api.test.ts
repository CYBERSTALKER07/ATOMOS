import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "../api";
import { formatMinorUnits } from "../money";

const here = dirname(fileURLToPath(import.meta.url));

describe("Admin Portal FinOps GL & 4-Way Reconciliation API", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("queries GL trial balance, REV:PLATFORM:FEE ledger, and 4-way recon endpoints with entity scope & auth headers", async () => {
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

    await api.getFinanceTrialBalance("admin-tok", "sup-9", "2026-11");
    await api.getFinanceAccountLedger("admin-tok", "sup-9", "REV:PLATFORM:FEE", "2026-11", 15);
    await api.listFinanceReconRuns("admin-tok", "sup-9", 5);
    await api.triggerFinanceReconRun("admin-tok", "sup-9", "2026-11");
    await api.listFinanceReconBreaks("admin-tok", "sup-9", "OPEN", 50);
    await api.updateFinanceReconBreak("admin-tok", "sup-9", "brk-7", {
      action: "RESOLVE",
      note: "verified",
    });

    expect(fetchMock).toHaveBeenCalledTimes(6);

    const calls = fetchMock.mock.calls.map(([url, init]) => ({
      url: String(url),
      method: init?.method,
      headers: init?.headers as Record<string, string>,
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
    }));

    expect(calls[0].url).toBe("http://localhost:8080/v1/finance/gl/trial-balance?entity=sup-9&period=2026-11");
    expect(calls[0].headers.Authorization).toBe("Bearer admin-tok");

    expect(calls[1].url).toBe(
      "http://localhost:8080/v1/finance/gl/accounts/REV%3APLATFORM%3AFEE/ledger?entity=sup-9&limit=15&period=2026-11",
    );
    expect(calls[2].url).toBe("http://localhost:8080/v1/finance/recon/runs?entity=sup-9&limit=5");

    expect(calls[3].url).toBe("http://localhost:8080/v1/finance/recon/run");
    expect(calls[3].method).toBe("POST");
    expect(calls[3].body).toEqual({ entity_id: "sup-9", period_id: "2026-11", mode: "MANUAL" });
    expect(calls[3].headers["Idempotency-Key"]).toBeTruthy();

    expect(calls[4].url).toBe("http://localhost:8080/v1/finance/recon/breaks?entity=sup-9&limit=50&status=OPEN");
    expect(calls[5].url).toBe("http://localhost:8080/v1/finance/recon/breaks/brk-7?entity=sup-9");
    expect(calls[5].method).toBe("PATCH");
    expect(calls[5].body).toEqual({ action: "RESOLVE", note: "verified" });
  });

  it("propagates backend errors honestly and formats GL minor units deterministically via BigInt", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: false,
        status: 400,
        json: async () => ({ error: "entity query/body parameter is required for platform admin" }),
      })),
    );

    await expect(api.getFinanceTrialBalance("admin-tok", "")).rejects.toThrow(
      "entity query/body parameter is required for platform admin",
    );

    expect(formatMinorUnits(12500000, { currency: "UZS", minimumFractionDigits: 0, maximumFractionDigits: 2 })).toBe(
      "125,000 UZS",
    );
  });

  it("keeps finops/page.tsx free of @pegasusx/ui imports", () => {
    const finopsPath = join(here, "../../app/finops/page.tsx");
    expect(existsSync(finopsPath)).toBe(true);
    const src = readFileSync(finopsPath, "utf8");
    expect(src).not.toContain("@pegasusx/ui");
    expect(src).toContain("getFinanceTrialBalance");
    expect(src).toContain("listFinanceReconRuns");
    expect(src).toContain("REV:PLATFORM:FEE");
  });
});
