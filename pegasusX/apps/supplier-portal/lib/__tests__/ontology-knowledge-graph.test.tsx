import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import { createTranslator, messageCatalogs } from "@pegasusx/i18n";
import {
  MARKING_FINANCE_ONLY,
  MARKING_LOT_RECALL_QUARANTINE,
  MARKING_PII,
  ONTOLOGY_LINK_TYPES,
  ONTOLOGY_OBJECT_TYPES,
} from "@pegasusx/types";

const { mockRequest, mockSearchParamsRef } = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockSearchParamsRef: { current: new URLSearchParams() },
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => createTranslator("en"),
}));

vi.mock("@/lib/use-supplier-session-reconcile", () => ({
  useSupplierSessionReconcile: () => {},
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => mockSearchParamsRef.current,
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    className,
  }: {
    children: ReactNode;
    href: string;
    className?: string;
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
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

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import {
  clampTraversalDepth,
  createSupplierOntologyApi,
  formatOntologyMarkingLabel,
  formatOntologyPropertyValue,
  securityMarkingTone,
} from "@/lib/ontology-api";
import KnowledgeGraphPage from "@/app/(portal)/analytics/knowledge-graph/page";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");
const repoRoot = join(portalRoot, "../..");

function collectTsFiles(dir: string, acc: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    if (entry === "node_modules" || entry === ".next" || entry === "out") continue;
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) {
      collectTsFiles(full, acc);
    } else if (/\.(ts|tsx|js|jsx|mjs)$/.test(entry)) {
      acc.push(full);
    }
  }
  return acc;
}

const SAMPLE_TYPES = ONTOLOGY_OBJECT_TYPES.map((objectType, idx) => ({
  object_type: objectType,
  version: "1.0.0",
  schema_file: `${objectType.toLowerCase()}.v1.schema.json`,
  primary_key: ["cell_id", "object_type", "object_id"],
  required_properties: ["supplier_id"],
  money_fields_minor: objectType === "Order" ? ["total_minor"] : [],
  security_markings:
    objectType === "Payment"
      ? [MARKING_FINANCE_ONLY]
      : objectType === "Driver"
        ? [MARKING_PII]
        : [],
  allowed_outgoing_links: ["supplies", "ordered"],
  description: `Canonical v1 ${objectType} entity (${idx + 1}/17).`,
  count: objectType === "Order" ? 2 : objectType === "Lot" ? 1 : 0,
}));

const ORDER_OBJ = {
  cell_id: "cell-uz",
  object_type: "Order",
  object_id: "ord-100",
  supplier_id: "sup-1",
  version: 4,
  properties: {
    order_id: "ord-100",
    supplier_id: "sup-1",
    retailer_id: "shop-9",
    status: "IN_TRANSIT",
    total_minor: 15000000,
    currency: "UZS",
  },
  security_markings: [MARKING_FINANCE_ONLY],
  valid_from: "2026-11-17T00:00:00Z",
  updated_at: "2026-11-17T00:10:00Z",
};

const LOT_OBJ = {
  cell_id: "cell-uz",
  object_type: "Lot",
  object_id: "lot-milk-889",
  supplier_id: "sup-1",
  version: 2,
  properties: {
    lot_id: "lot-milk-889",
    lot_code: "BATCH-889",
    status: "QUARANTINED",
    quantity_on_hand: 48,
  },
  security_markings: [MARKING_LOT_RECALL_QUARANTINE],
  valid_from: "2026-11-17T00:00:00Z",
  updated_at: "2026-11-17T00:12:00Z",
};

const SHOP_OBJ = {
  cell_id: "cell-uz",
  object_type: "Shop",
  object_id: "shop-9",
  supplier_id: "sup-1",
  version: 3,
  properties: {
    retailer_id: "shop-9",
    name: "Baraka Market",
    phone: "[REDACTED]",
  },
  security_markings: [MARKING_PII],
  valid_from: "2026-11-17T00:00:00Z",
  updated_at: "2026-11-17T00:11:00Z",
};

describe("M0.5 Ontology Knowledge Graph API, UI & Guards", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParamsRef.current = new URLSearchParams();
  });

  afterEach(() => {
    cleanup();
  });

  it("calls all /v1/ontology/* and /v1/supplier/knowledge-graph routes via ApiClient with exact params and headers", async () => {
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      return new Response(JSON.stringify({ ok: true, url: String(input), method: init?.method }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-m05",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const client = createSupplierOntologyApi(realClient);

    await client.listTypes();
    await client.listObjects({
      type: "Lot",
      q: "milk",
      cell_id: "cell-uz",
      marking: MARKING_LOT_RECALL_QUARANTINE,
      limit: 50,
      offset: 10,
    });
    await client.getNeighbors("Order", "ord-100", {
      depth: 99,
      cell_id: "cell-uz",
      link_type: "ordered",
    });
    await client.rebuild({ cell_id: "cell-uz" }, "idem-rebuild-1");
    await client.getKnowledgeGraph();

    const calls = fetchSpy.mock.calls.map(([url, init]) => ({
      url: String(url),
      method: init?.method,
      headers: init?.headers as Headers,
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
    }));

    expect(calls[0]).toMatchObject({
      url: "https://api.pegasusx.test/v1/ontology/types",
      method: "GET",
    });
    expect(calls[1].url).toBe(
      "https://api.pegasusx.test/v1/ontology/objects?type=Lot&q=milk&cell_id=cell-uz&marking=MARKING%3ALOT_RECALL_QUARANTINE&limit=50&offset=10",
    );
    expect(calls[2].url).toBe(
      "https://api.pegasusx.test/v1/ontology/objects/Order/ord-100/neighbors?depth=4&cell_id=cell-uz&link_type=ordered",
    );
    expect(calls[3].url).toBe("https://api.pegasusx.test/v1/ontology/rebuild");
    expect(calls[3].method).toBe("POST");
    expect(calls[3].headers.get("Idempotency-Key")).toBe("idem-rebuild-1");
    expect(calls[3].body).toEqual({ cell_id: "cell-uz" });
    expect(calls[4]).toMatchObject({
      url: "https://api.pegasusx.test/v1/supplier/knowledge-graph",
      method: "GET",
    });
  });

  it("formats minor-unit properties strictly via BigInt and classifies security markings", () => {
    expect(clampTraversalDepth(0)).toBe(1);
    expect(clampTraversalDepth(3)).toBe(3);
    expect(clampTraversalDepth(12)).toBe(4);

    expect(formatOntologyPropertyValue("total_minor", 15000000, "UZS")).toBe("150,000 UZS");
    expect(formatOntologyPropertyValue("credit_limit_minor", "900719925474099300", "UZS")).toBe(
      "9,007,199,254,740,993 UZS",
    );
    expect(formatOntologyPropertyValue("is_active", true)).toBe("true");

    expect(securityMarkingTone(MARKING_FINANCE_ONLY)).toBe("finance");
    expect(securityMarkingTone(MARKING_PII)).toBe("pii");
    expect(securityMarkingTone(MARKING_LOT_RECALL_QUARANTINE)).toBe("quarantine");
    expect(formatOntologyMarkingLabel(MARKING_FINANCE_ONLY)).toBe("FINANCE_ONLY");
    expect(formatOntologyMarkingLabel(MARKING_PII)).toBe("PII");
  });

  it("renders all 17 entity type filters, security marking badges, typed properties drawer, multi-hop traversal, and rebuild trigger", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, opts?: { body?: unknown }) => {
      if (path === "/v1/ontology/types" && method === "GET") {
        return {
          version: "1.0.0",
          types: SAMPLE_TYPES,
          link_types: [...ONTOLOGY_LINK_TYPES],
          security_markings: [MARKING_FINANCE_ONLY, MARKING_PII, MARKING_LOT_RECALL_QUARANTINE],
        };
      }
      if (path.startsWith("/v1/ontology/objects?") && method === "GET") {
        const url = new URL(`https://local.test${path}`);
        const typeParam = url.searchParams.get("type");
        const all = [ORDER_OBJ, LOT_OBJ];
        const filtered = typeParam ? all.filter((o) => o.object_type === typeParam) : all;
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          objects: filtered,
          total: filtered.length,
          limit: 100,
          offset: 0,
          pii_masked: true,
          finance_masked: false,
        };
      }
      if (path.startsWith("/v1/ontology/objects/Order/ord-100/neighbors") && method === "GET") {
        const url = new URL(`https://local.test${path}`);
        const depth = Number(url.searchParams.get("depth") || 2);
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          root: ORDER_OBJ,
          depth,
          nodes: [
            {
              object: SHOP_OBJ,
              hop: 1,
              via_link_type: "ordered",
              direction: "INBOUND",
              parent_type: "Order",
              parent_id: "ord-100",
            },
            {
              object: LOT_OBJ,
              hop: depth >= 2 ? 2 : 1,
              via_link_type: "stocks",
              direction: "OUTBOUND",
              parent_type: "Order",
              parent_id: "ord-100",
            },
          ],
          edges: [
            {
              cell_id: "cell-uz",
              source_object_type: "Shop",
              source_object_id: "shop-9",
              link_type: "ordered",
              target_object_type: "Order",
              target_object_id: "ord-100",
              supplier_id: "sup-1",
              created_by_action_id: "act-1",
              valid_from: "2026-11-17T00:00:00Z",
            },
          ],
          total_nodes: 2,
          total_edges: 1,
          pii_masked: true,
          finance_masked: false,
        };
      }
      if (path.startsWith("/v1/ontology/objects/Shop/shop-9/neighbors") && method === "GET") {
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          root: SHOP_OBJ,
          depth: 3,
          nodes: [
            {
              object: ORDER_OBJ,
              hop: 1,
              via_link_type: "ordered",
              direction: "OUTBOUND",
              parent_type: "Shop",
              parent_id: "shop-9",
            },
          ],
          edges: [],
          total_nodes: 1,
          total_edges: 0,
          pii_masked: true,
          finance_masked: false,
        };
      }
      if (path === "/v1/ontology/rebuild" && method === "POST") {
        expect(opts?.body).toBeDefined();
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          status: "COMMITTED",
          objects_upserted: 42,
          edges_upserted: 67,
          batches_committed: 1,
          duration_ms: 18,
          tables_scanned: ["Orders", "StockLots", "Retailers"],
          rebuilt_at: "2026-11-17T00:45:00Z",
        };
      }
      if (path === "/v1/supplier/knowledge-graph" && method === "GET") {
        return {
          supplier_id: "sup-1",
          source: "ontology_unified",
          ontology_objects_count: 42,
          ontology_edges_count: 67,
          nodes: [{ id: "sup-1", type: "supplier", name: "Samarkand Dairy" }],
          edges: [],
        };
      }
      throw new Error(`Unexpected call: ${method} ${path}`);
    });

    render(<KnowledgeGraphPage />);

    // All 17 entity type filter chips rendered
    const typeFilterGroup = await screen.findByRole("group", { name: "Entity type" });
    for (const typeName of ONTOLOGY_OBJECT_TYPES) {
      expect(
        within(typeFilterGroup).getByRole("button", { name: new RegExp(`^${typeName}(?:\\s|$)`) }),
      ).toBeInTheDocument();
    }

    // Security marking badges & PII masking notice
    expect(await screen.findAllByText("FINANCE_ONLY")).not.toHaveLength(0);
    expect(screen.getAllByText("LOT_RECALL_QUARANTINE").length).toBeGreaterThan(0);
    expect(screen.getAllByText("PII").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/PII fields masked for current role/i).length).toBeGreaterThan(0);

    // Typed properties drawer renders BigInt formatted minor units
    expect(screen.getAllByText("150,000 UZS").length).toBeGreaterThan(0);

    // Change hop depth to 3
    fireEvent.click(screen.getByRole("button", { name: "3" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/ontology/objects/Order/ord-100/neighbors?depth=3",
        "GET",
      );
    });

    // Traverse from neighbor Shop:shop-9
    const focusButtons = screen.getAllByRole("button", { name: "Focus & traverse" });
    fireEvent.click(focusButtons[0]);
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/ontology/objects/Shop/shop-9/neighbors?depth=3",
        "GET",
      );
    });
    const drawer = screen.getByRole("complementary", { name: "Typed Properties Inspector" });
    expect(await within(drawer).findByRole("heading", { name: "Baraka Market" })).toBeInTheDocument();
    expect(within(drawer).getByText("[REDACTED]")).toBeInTheDocument();

    // Filter by Lot type chip
    fireEvent.click(within(typeFilterGroup).getByRole("button", { name: /^Lot(?:\s|$)/ }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/ontology/objects?type=Lot&limit=100",
        "GET",
      );
    });

    // Trigger Rebuild Ontology
    fireEvent.click(screen.getByRole("button", { name: "Rebuild Ontology" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/ontology/rebuild",
        "POST",
        expect.objectContaining({
          idempotencyKey: expect.stringMatching(/^ontology-rebuild-/),
        }),
      );
    });
    expect(await screen.findByText(/Rebuilt 42 objects and 67 edges in 18 ms/i)).toBeInTheDocument();
  });

  it("initializes deep-linked object traversal from useSearchParams inside Suspense", async () => {
    mockSearchParamsRef.current = new URLSearchParams("type=Lot&id=lot-milk-889&depth=4");

    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path === "/v1/ontology/types" && method === "GET") {
        return {
          version: "1.0.0",
          types: SAMPLE_TYPES,
          link_types: [...ONTOLOGY_LINK_TYPES],
          security_markings: [MARKING_LOT_RECALL_QUARANTINE],
        };
      }
      if (path.startsWith("/v1/ontology/objects?") && method === "GET") {
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          objects: [LOT_OBJ],
          total: 1,
          limit: 100,
          offset: 0,
          pii_masked: false,
          finance_masked: false,
        };
      }
      if (path.startsWith("/v1/ontology/objects/Lot/lot-milk-889/neighbors") && method === "GET") {
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          root: LOT_OBJ,
          depth: 4,
          nodes: [],
          edges: [],
          total_nodes: 0,
          total_edges: 0,
          pii_masked: false,
          finance_masked: false,
        };
      }
      if (path === "/v1/supplier/knowledge-graph" && method === "GET") {
        return { supplier_id: "sup-1", nodes: [], edges: [] };
      }
      throw new Error(`Unexpected call: ${method} ${path}`);
    });

    render(<KnowledgeGraphPage />);

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/ontology/objects/Lot/lot-milk-889/neighbors?depth=4",
        "GET",
      );
    });
    expect(
      await screen.findByText(/No connected neighbor objects within 4 hop\(s\)\./i),
    ).toBeInTheDocument();
  });

  it("surfaces 403 rebuild permission errors inline without losing rendered graph data", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path === "/v1/ontology/types" && method === "GET") {
        return {
          version: "1.0.0",
          types: SAMPLE_TYPES,
          link_types: [...ONTOLOGY_LINK_TYPES],
          security_markings: [MARKING_FINANCE_ONLY],
        };
      }
      if (path.startsWith("/v1/ontology/objects?") && method === "GET") {
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          objects: [ORDER_OBJ],
          total: 1,
          limit: 100,
          offset: 0,
          pii_masked: false,
          finance_masked: false,
        };
      }
      if (path.startsWith("/v1/ontology/objects/Order/ord-100/neighbors") && method === "GET") {
        return {
          cell_id: "cell-uz",
          supplier_id: "sup-1",
          root: ORDER_OBJ,
          depth: 2,
          nodes: [],
          edges: [],
          total_nodes: 0,
          total_edges: 0,
          pii_masked: false,
          finance_masked: false,
        };
      }
      if (path === "/v1/supplier/knowledge-graph" && method === "GET") {
        return { supplier_id: "sup-1", nodes: [], edges: [] };
      }
      if (path === "/v1/ontology/rebuild" && method === "POST") {
        throw new ApiError("Forbidden", 403, {
          error: "forbidden",
          message: "config.manage permission required",
        });
      }
      throw new Error(`Unexpected call: ${method} ${path}`);
    });

    render(<KnowledgeGraphPage />);

    expect(await screen.findByRole("button", { name: "Rebuild Ontology" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Rebuild Ontology" }));

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("config.manage permission required");
    expect(screen.getAllByText("ord-100").length).toBeGreaterThan(0);
  });

  it("enforces zero @pegasusx/ui imports, Suspense wrapper, and en/ru/uz i18n parity", () => {
    const kgSrc = readFileSync(
      join(portalRoot, "app/(portal)/analytics/knowledge-graph/page.tsx"),
      "utf8",
    );
    expect(kgSrc).toContain("<Suspense");
    expect(kgSrc).toContain("useSearchParams");

    for (const appDir of ["apps/supplier-portal", "apps/warehouse-portal"]) {
      const files = collectTsFiles(join(repoRoot, appDir));
      for (const file of files) {
        const content = readFileSync(file, "utf8");
        expect(content, `Forbidden @pegasusx/ui import in ${file}`).not.toMatch(
          /from\s+["']@pegasusx\/ui|import\(\s*["']@pegasusx\/ui/,
        );
      }
    }

    const kgKeys = Object.keys(messageCatalogs.en).filter((k) =>
      k.startsWith("supplier_portal.analytics.knowledge_graph."),
    );
    expect(kgKeys.length).toBeGreaterThanOrEqual(30);
    for (const key of kgKeys) {
      expect(messageCatalogs.en[key], `en.${key}`).toBeTruthy();
      expect(messageCatalogs.ru[key], `ru.${key}`).toBeTruthy();
      expect(messageCatalogs.uz[key], `uz.${key}`).toBeTruthy();
      expect(existsSync(join(portalRoot, "lib/ontology-api.ts"))).toBe(true);
    }
  });
});
