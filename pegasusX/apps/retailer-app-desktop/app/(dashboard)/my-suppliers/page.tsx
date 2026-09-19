"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useState } from "react";
import {
  Store,
  RefreshCw,
  Search,
  Plus,
  Trash2,
  AlertTriangle,
  Building2,
  CheckCircle2,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { apiFetch } from "@/lib/auth";
import type { Supplier } from "@/lib/types";

export default function MySuppliersPage() {
  const t = usePortalT();
  const {
    data: suppliers,
    loading: suppliersLoading,
    error: suppliersError,
    isRefreshing: isSuppliersRefreshing,
    mutate: mutateSuppliers,
  } = useLiveData<Supplier[]>("/v1/retailer/suppliers");

  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<Supplier[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);

  const [actionMessage, setActionMessage] = useState<{ type: "success" | "error", text: string } | null>(null);

  const supplierList = suppliers || [];

  const handleSearch = async (query: string) => {
    setSearchQuery(query);
    if (query.trim().length < 2) {
      setSearchResults([]);
      return;
    }
    setIsSearching(true);
    setSearchError(null);
    try {
      const res = await apiFetch(`/v1/catalog/suppliers/search?q=${encodeURIComponent(query)}`);
      if (!res.ok) throw new Error("Search failed");
      const data = await res.json();
      const existingIds = new Set(supplierList.map(s => s.id));
      setSearchResults((Array.isArray(data) ? data : []).filter((s: Supplier) => !existingIds.has(s.id)));
    } catch {
      setSearchError("Failed to search suppliers.");
    } finally {
      setIsSearching(false);
    }
  };

  const addSupplier = async (supplierId: string) => {
    setActionMessage(null);
    try {
      const res = await apiFetch(`/v1/retailer/suppliers/${supplierId}/add`, { method: "POST" });
      if (!res.ok) throw new Error("Add failed");
      setActionMessage({ type: "success", text: "Supplier added successfully." });
      void mutateSuppliers();
      setSearchResults(prev => prev.filter(s => s.id !== supplierId));
    } catch (err) {
      setActionMessage({ type: "error", text: "Failed to add supplier." });
    }
  };

  const removeSupplier = async (supplierId: string) => {
    setActionMessage(null);
    try {
      const res = await apiFetch(`/v1/retailer/suppliers/${supplierId}/remove`, { method: "POST" });
      if (!res.ok) throw new Error("Remove failed");
      setActionMessage({ type: "success", text: "Supplier removed successfully." });
      void mutateSuppliers();
    } catch (err) {
      setActionMessage({ type: "error", text: "Failed to remove supplier." });
    }
  };

  const refreshAll = useCallback(() => {
    setActionMessage(null);
    void mutateSuppliers();
  }, [mutateSuppliers]);

  const isLoading = suppliersLoading;

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="store"
        title={t("portal.nav.my_suppliers")}
        description={t("retailer_desktop.residual.text.manage_your_approved_wholesale_suppliers_and_discover_new_partne")}
        loading={isLoading}
        skeletonVariant="table"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={isLoading || isSuppliersRefreshing}
              onClick={refreshAll}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isSuppliersRefreshing ? "animate-spin" : ""}`}
              />
              {isSuppliersRefreshing ? "Syncing" : "Sync"}
            </button>
          </div>
        }
      >
        {actionMessage && (
          <div className={`mb-6 flex items-center gap-2 p-3 rounded-xl border ${actionMessage.type === "error" ? "bg-[var(--desk-warning)]/10 text-[var(--desk-warning)] border-[var(--desk-warning)]/30" : "bg-[var(--desk-success)]/10 text-[var(--desk-success)] border-[var(--desk-success)]/30"}`}>
            {actionMessage.type === "error" ? <AlertTriangle size={16} /> : <CheckCircle2 size={16} />}
            <span className="md-typescale-body-small">{actionMessage.text}</span>
          </div>
        )}

        <div className="flex gap-8 flex-col lg:flex-row">
          {/* Active Suppliers List */}
          <div className="flex-1">
            <PageSection title={t("retailer_desktop.my_suppliers.text.connected_suppliers")} description={`You are connected to ${supplierList.length} suppliers.`}>
              <div className="space-y-4">
                {supplierList.length === 0 ? (
                  <div className="p-8 text-center text-[var(--desk-text-tertiary)] bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl">
                    <Store size={48} className="mx-auto mb-4 opacity-20" />
                    <p className="md-typescale-body-large">{t("retailer_desktop.my_suppliers.text.no_connected_suppliers")}</p>
                    <p className="md-typescale-body-small mt-2">{t("retailer_desktop.my_suppliers.text.search_and_add_suppliers_to_see_their_catalogs_and_start_orderin")}</p>
                  </div>
                ) : (
                  supplierList.map((supplier) => (
                    <div key={supplier.id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 rounded-full overflow-hidden bg-[var(--desk-surface-subtle)] flex items-center justify-center">
                          {supplier.logo_url ? (
                            <img src={supplier.logo_url} alt={supplier.name} className="w-full h-full object-cover" />
                          ) : (
                            <Building2 size={24} className="text-[var(--desk-text-tertiary)]" />
                          )}
                        </div>
                        <div>
                          <div className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">{supplier.name}</div>
                          <div className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase tracking-widest mt-1">
                            {supplier.category || "General"}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <div className="text-right hidden sm:block">
                          <div className="md-typescale-body-medium text-[var(--desk-text-primary)]">{supplier.order_count}</div>
                          <div className="md-typescale-label-small text-[var(--desk-text-tertiary)] uppercase">{t("portal.nav.orders")}</div>
                        </div>
                        <button
                          onClick={() => removeSupplier(supplier.id)}
                          className="w-10 h-10 flex items-center justify-center rounded-full text-[var(--desk-danger)] hover:bg-[var(--desk-danger)]/10 transition-colors"
                          title={t("retailer_desktop.my_suppliers.text.remove_supplier")}
                        >
                          <Trash2 size={18} />
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </PageSection>
          </div>

          {/* Supplier Directory / Search */}
          <div className="w-full lg:w-[360px] shrink-0">
            <PageSection title={t("retailer_desktop.my_suppliers.text.supplier_directory")} description={t("retailer_desktop.residual.text.find_and_connect_with_new_suppliers")}>
              <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
                <div className="relative mb-6">
                  <Search
                    className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)]"
                    size={18}
                  />
                  <input
                    type="text"
                    placeholder={t("retailer_desktop.my_suppliers.text.search_by_name_or_category")}
                    className="w-full h-11 pl-11 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] transition-all md-typescale-body-medium text-[var(--desk-text-primary)]"
                    value={searchQuery}
                    onChange={(e) => handleSearch(e.target.value)}
                  />
                </div>

                {isSearching ? (
                  <div className="flex justify-center py-8">
                    <RefreshCw size={24} className="animate-spin text-[var(--desk-text-tertiary)]" />
                  </div>
                ) : searchError ? (
                  <div className="text-center py-4 text-[var(--desk-danger)] text-sm">{searchError}</div>
                ) : searchQuery.trim().length > 0 && searchResults.length === 0 ? (
                  <div className="text-center py-8 text-[var(--desk-text-tertiary)] text-sm">
                    No matching suppliers found.
                  </div>
                ) : (
                  <div className="space-y-3">
                    {searchResults.map((supplier) => (
                      <div key={supplier.id} className="flex flex-col p-3 border border-[var(--desk-border)] rounded-xl hover:shadow-md transition-shadow">
                        <div className="flex items-center gap-3 mb-3">
                          <div className="w-10 h-10 rounded-full overflow-hidden bg-[var(--desk-surface-subtle)] flex items-center justify-center shrink-0">
                            {supplier.logo_url ? (
                              <img src={supplier.logo_url} alt={supplier.name} className="w-full h-full object-cover" />
                            ) : (
                              <Building2 size={20} className="text-[var(--desk-text-tertiary)]" />
                            )}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="md-typescale-body-medium font-medium text-[var(--desk-text-primary)] truncate">{supplier.name}</div>
                            <div className="md-typescale-label-small text-[var(--desk-text-tertiary)] truncate">{supplier.category || "General"}</div>
                          </div>
                        </div>
                        <button
                          onClick={() => addSupplier(supplier.id)}
                          className="w-full h-9 flex items-center justify-center gap-2 bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] rounded-lg text-sm font-medium hover:bg-[var(--desk-accent)] hover:text-white transition-colors"
                        >
                          <Plus size={16} />
                          Connect
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </PageSection>
          </div>
        </div>
      </PageChrome>
    </div>
  );
}
