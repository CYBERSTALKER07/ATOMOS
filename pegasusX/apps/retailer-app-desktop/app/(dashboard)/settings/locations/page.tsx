"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  MapPin,
  Star,
  Plus,
  AlertTriangle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type Location = {
  location_id: string;
  name: string;
  delivery_address?: string;
  lat?: number;
  lng?: number;
  is_primary: boolean;
  is_active: boolean;
};

export default function LocationsPage() {
  const router = useRouter();
  const [items, setItems] = useState<Location[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({
    name: "",
    delivery_address: "",
    lat: "",
    lng: "",
  });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/locations");
      if (!res.ok) throw new Error(`load_failed_${res.status}`);
      const json = (await res.json()) as {
        items?: Location[];
        active_location_id?: string;
      };
      setItems(json.items ?? []);
      setActiveId(json.active_location_id ?? "");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load locations");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/locations", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `loc-create-${Date.now()}`,
        },
        body: JSON.stringify({
          name: form.name,
          delivery_address: form.delivery_address,
          lat: form.lat ? Number(form.lat) : 0,
          lng: form.lng ? Number(form.lng) : 0,
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (json as { error?: string }).error || `create_failed_${res.status}`,
        );
      }
      setBanner("Location created (LOCATIONS pack auto-enabled if needed)");
      setForm({ name: "", delivery_address: "", lat: "", lng: "" });
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Create failed");
    } finally {
      setBusy(false);
    }
  };

  const setPrimary = async (id: string) => {
    setBusy(true);
    try {
      const res = await apiFetch(`/v1/retailer/locations/${id}/set-primary`, {
        method: "POST",
        headers: { "Idempotency-Key": `loc-primary-${id}-${Date.now()}` },
      });
      if (!res.ok) throw new Error(`set_primary_failed_${res.status}`);
      setBanner("Primary location updated (shop delivery address mirrored)");
      await load();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Set primary failed");
    } finally {
      setBusy(false);
    }
  };

  const switchActive = async (id: string) => {
    setBusy(true);
    try {
      const res = await apiFetch("/v1/auth/retailer/switch-location", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ location_id: id }),
      });
      const json = (await res.json().catch(() => ({}))) as {
        token?: string;
        error?: string;
      };
      if (!res.ok) throw new Error(json.error || `switch_failed_${res.status}`);
      if (json.token && typeof document !== "undefined") {
        document.cookie = `pegasus_retailer_jwt=${json.token}; path=/; SameSite=Lax`;
        try {
          localStorage.setItem("pegasus_retailer_jwt", json.token);
        } catch {
          /* ignore */
        }
      }
      setActiveId(id);
      setBanner("Active branch switched");
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Switch failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title="Locations"
      description="Branches and delivery addresses. Solo shops get one primary automatically."
      actions={
        <button
          type="button"
          onClick={() => router.push("/settings")}
          className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm hover:bg-muted"
        >
          <ArrowLeft className="h-4 w-4" />
          Settings
        </button>
      }
    >
      <div className="mx-auto max-w-3xl space-y-6 px-4 pb-16 pt-2">
        {banner && (
          <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        {error && (
          <div className="flex items-center gap-2 text-sm text-red-600">
            <AlertTriangle className="h-4 w-4" />
            {error}
            <button type="button" className="underline" onClick={() => void load()}>
              Retry
            </button>
          </div>
        )}

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 flex items-center gap-2 font-semibold">
            <Plus className="h-4 w-4" />
            Add branch
          </h2>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="text-sm">
              Name
              <input
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </label>
            <label className="text-sm">
              Address
              <input
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.delivery_address}
                onChange={(e) =>
                  setForm((f) => ({ ...f, delivery_address: e.target.value }))
                }
              />
            </label>
            <label className="text-sm">
              Lat
              <input
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.lat}
                onChange={(e) => setForm((f) => ({ ...f, lat: e.target.value }))}
              />
            </label>
            <label className="text-sm">
              Lng
              <input
                className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2"
                value={form.lng}
                onChange={(e) => setForm((f) => ({ ...f, lng: e.target.value }))}
              />
            </label>
          </div>
          <button
            type="button"
            disabled={busy}
            onClick={() => void create()}
            className="mt-4 rounded-lg bg-foreground px-4 py-2 text-sm font-medium text-background disabled:opacity-50"
          >
            {busy ? "…" : "Create location"}
          </button>
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 flex items-center gap-2 font-semibold">
            <MapPin className="h-4 w-4" />
            Branches
          </h2>
          {loading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading…
            </div>
          )}
          <ul className="divide-y divide-border">
            {items.map((loc) => (
              <li
                key={loc.location_id}
                className="flex flex-wrap items-center justify-between gap-2 py-3"
              >
                <div>
                  <div className="font-medium">
                    {loc.name}{" "}
                    {loc.is_primary && (
                      <span className="ml-1 inline-flex items-center gap-0.5 rounded-full bg-muted px-2 py-0.5 text-[10px] uppercase">
                        <Star className="h-3 w-3" /> Primary
                      </span>
                    )}
                    {activeId === loc.location_id && (
                      <span className="ml-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] uppercase text-emerald-700">
                        Active
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {loc.delivery_address || "No address"} · {loc.location_id}
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  {activeId !== loc.location_id && loc.is_active && (
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void switchActive(loc.location_id)}
                      className="rounded-lg border border-border px-3 py-1.5 text-xs hover:bg-muted"
                    >
                      Use for checkout
                    </button>
                  )}
                  {!loc.is_primary && loc.is_active && (
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void setPrimary(loc.location_id)}
                      className="rounded-lg border border-border px-3 py-1.5 text-xs hover:bg-muted"
                    >
                      Set primary
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </PageChrome>
  );
}
