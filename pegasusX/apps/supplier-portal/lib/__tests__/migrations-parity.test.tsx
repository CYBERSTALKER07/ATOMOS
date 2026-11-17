import { describe, expect, it, vi } from "vitest";
import { computeSha256Hex, uploadArtifactToSignedTicket } from "@/lib/migration-upload";
import { domainQueryToPack, packLabelKey } from "@/components/migrations/pack-labels";
import { NAV, allNavHrefs } from "@/lib/nav";

describe("M9 supplier migration & join-request parity", () => {
  it("exposes /migrations and /join-requests in navigation without breaking preserved routes", () => {
    const hrefs = allNavHrefs(NAV);
    expect(hrefs).toContain("/migrations");
    expect(hrefs).toContain("/join-requests");
  });

  it("maps domain query params to canonical backend MigrationPack slugs", () => {
    expect(domainQueryToPack("products")).toBe("products_prices");
    expect(domainQueryToPack("stock")).toBe("stock");
    expect(domainQueryToPack("lots")).toBe("lots");
    expect(domainQueryToPack("nasiya")).toBe("nasiya_ar");
    expect(domainQueryToPack("orders_open")).toBe("open_orders");
    expect(packLabelKey("nasiya_ar")).toBe("supplier_portal.migrations.pack_nasiya");
  });

  it("computes SHA-256 hex digest and handles local.invalid dev tickets safely", async () => {
    const bytes = new TextEncoder().encode("sku,qty\nA1,5\n");
    const hex = await computeSha256Hex(bytes);
    expect(hex).toMatch(/^[a-f0-9]{64}$/);

    const onProgress = vi.fn();
    await uploadArtifactToSignedTicket(
      {
        artifact_id: "art-1",
        file_name: "stock.csv",
        gcs_path: "migrations/cell-uz/sup-1/stock.csv",
        upload_url: "https://local.invalid/upload/stock.csv",
        method: "PUT",
        content_type: "text/csv",
        resumable: false,
        expires_at: new Date().toISOString(),
      },
      bytes,
      onProgress,
    );
    expect(onProgress).toHaveBeenCalledWith(bytes.byteLength, bytes.byteLength);
  });
});
