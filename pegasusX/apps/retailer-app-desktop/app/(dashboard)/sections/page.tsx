"use client";

import { useCallback, useEffect, useState } from "react";
import { Loader2, LayoutGrid } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type Section = {
  section_id: string;
  name: string;
  location_id: string;
  aisle_tag?: string;
  shelf_tag?: string;
  status: string;
  sku_count?: number;
};

export default function SectionsPage() {
  const [items, setItems] = useState<Section[]>([]);
  const [name, setName] = useState("Dairy");
  const [aisle, setAisle] = useState("");
  const [skuInput, setSkuInput] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const [skus, setSkus] = useState<string[]>([]);
  const [staffIds, setStaffIds] = useState("");
  const [unassigned, setUnassigned] = useState<string[]>([]);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch("/v1/retailer/sections");
      if (res.ok) {
        const json = (await res.json()) as { items?: Section[] };
        setItems(json.items ?? []);
      }
      const un = await apiFetch("/v1/retailer/sections/unassigned-skus");
      if (un.ok) {
        const json = (await un.json()) as { skus?: string[] };
        setUnassigned(json.skus ?? []);
      }
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    setBusy(true);
    try {
      const res = await apiFetch("/v1/retailer/sections", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `sec-${Date.now()}`,
        },
        body: JSON.stringify({ name, aisle_tag: aisle || undefined }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error((json as { error?: string }).error || "create_failed");
      setBanner("Section created (SECTIONS + STORE_STOCK auto-enabled if needed)");
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Create failed");
    } finally {
      setBusy(false);
    }
  };

  const loadDetail = async (id: string) => {
    setSelectedId(id);
    const res = await apiFetch(`/v1/retailer/sections/${id}`);
    if (!res.ok) return;
    const json = (await res.json()) as { skus?: string[]; staff_ids?: string[] };
    setSkus(json.skus ?? []);
    setStaffIds((json.staff_ids ?? []).join(","));
  };

  const saveSkus = async () => {
    if (!selectedId) return;
    setBusy(true);
    try {
      const list = skuInput
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      const res = await apiFetch(`/v1/retailer/sections/${selectedId}/skus`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ skus: list.length ? list : skus }),
      });
      if (!res.ok) throw new Error("sku_map_failed");
      setBanner("SKUs updated");
      await loadDetail(selectedId);
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "SKU update failed");
    } finally {
      setBusy(false);
    }
  };

  const saveStaff = async () => {
    if (!selectedId) return;
    setBusy(true);
    try {
      const ids = staffIds
        .split(/[\s,]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      const res = await apiFetch(`/v1/retailer/sections/${selectedId}/staff`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_ids: ids }),
      });
      if (!res.ok) throw new Error("staff_assign_failed");
      setBanner("Staff assigned");
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Staff update failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title="Sections"
      description="Departments and shelves — map SKUs and assign staff. Enables SECTIONS pack."
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-6 p-4">
        {banner && (
          <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        <section className="rounded-xl border border-border bg-card p-4">
          <div className="mb-3 flex items-center gap-2">
            <LayoutGrid className="h-5 w-5 text-muted-foreground" />
            <h2 className="font-semibold">New section</h2>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <input
              className="rounded-md border border-border bg-background px-3 py-2 text-sm"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Name"
            />
            <input
              className="rounded-md border border-border bg-background px-3 py-2 text-sm"
              value={aisle}
              onChange={(e) => setAisle(e.target.value)}
              placeholder="Aisle tag"
            />
          </div>
          <button
            type="button"
            disabled={busy}
            onClick={() => void create()}
            className="mt-3 inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Create
          </button>
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">Sections</h2>
          {items.length === 0 ? (
            <p className="text-sm text-muted-foreground">No sections yet.</p>
          ) : (
            <ul className="space-y-2">
              {items.map((s) => (
                <li key={s.section_id}>
                  <button
                    type="button"
                    onClick={() => void loadDetail(s.section_id)}
                    className={`w-full rounded-lg border px-3 py-2 text-left text-sm ${
                      selectedId === s.section_id
                        ? "border-primary bg-primary/5"
                        : "border-border"
                    }`}
                  >
                    {s.name}
                    {s.aisle_tag ? ` · ${s.aisle_tag}` : ""} · {s.sku_count ?? 0} SKUs
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        {selectedId && (
          <section className="rounded-xl border border-border bg-card p-4">
            <h2 className="mb-3 font-semibold">Map SKUs / staff</h2>
            <p className="mb-2 text-xs text-muted-foreground">
              Current SKUs: {skus.join(", ") || "none"}
            </p>
            <textarea
              className="mb-2 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              rows={2}
              placeholder="SKU list (comma-separated) — replaces map"
              value={skuInput}
              onChange={(e) => setSkuInput(e.target.value)}
            />
            <button
              type="button"
              disabled={busy}
              onClick={() => void saveSkus()}
              className="mr-2 rounded-lg border border-border px-3 py-1.5 text-sm"
            >
              Save SKUs
            </button>
            <input
              className="mt-3 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              placeholder="Staff user IDs (comma-separated)"
              value={staffIds}
              onChange={(e) => setStaffIds(e.target.value)}
            />
            <button
              type="button"
              disabled={busy}
              onClick={() => void saveStaff()}
              className="mt-2 rounded-lg border border-border px-3 py-1.5 text-sm"
            >
              Save staff
            </button>
          </section>
        )}

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">Unassigned SKUs</h2>
          <p className="text-sm text-muted-foreground">
            {unassigned.length === 0 ? "None (or no stock yet)." : unassigned.join(", ")}
          </p>
        </section>
      </div>
    </PageChrome>
  );
}
