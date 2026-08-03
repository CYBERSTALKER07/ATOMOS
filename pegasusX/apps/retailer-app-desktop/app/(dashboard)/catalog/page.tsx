"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ShoppingCart,
  Search,
  SlidersHorizontal,
  Package,
  Star,
  Layers,
  TrendingUp,
  Building2,
  ChevronRight,
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import CartDrawer from "../../../components/CartDrawer";
import CheckoutModal from "../../../components/CheckoutModal";
import ProductDetailDrawer from "../../../components/ProductDetailDrawer";
import EmptyState from "../../../components/EmptyState";
import { CatalogFilters } from "../../../components/catalog/CatalogFilters";
import { ProductGrid } from "../../../components/catalog/ProductGrid";
import { PageSection } from "../../../components/PageSection";
import { Skeleton } from "../../../components/Skeleton";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { useCart } from "../../../lib/cart";
import { isCatalogBlocked } from "../../../lib/stock-policy";
import { useWebSocket } from "../../../lib/ws";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import type { Product, Category, Supplier } from "../../../lib/types";
import {
  productDisplayPrice,
  productListPrice,
  productSalePrice,
} from "../../../lib/types";

const EMPTY_PRODUCTS: Product[] = [];
const EMPTY_CATEGORIES: Category[] = [];
const EMPTY_SUPPLIERS: Supplier[] = [];

export default function CatalogPage() {
  const {
    data: products,
    loading: loadingProducts,
    error: productsError,
    isRefreshing: isProductsRefreshing,
    mutate: mutateProducts,
  } = useLiveData<Product[]>("/v1/catalog/products");
  const {
    data: categories,
    error: categoriesError,
    isRefreshing: isCategoriesRefreshing,
    mutate: mutateCategories,
  } = useLiveData<Category[]>("/v1/catalog/categories");
  const {
    data: suppliers,
    error: suppliersError,
    isRefreshing: isSuppliersRefreshing,
    mutate: mutateSuppliers,
  } = useLiveData<Supplier[]>("/v1/retailer/suppliers");
  const { items, addToCart } = useCart();
  const { subscribe } = useWebSocket();

  const [isCartOpen, setIsCartOpen] = useState(false);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState("All");
  const [activeSupplier, setActiveSupplier] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [categorySuppliers, setCategorySuppliers] = useState<Supplier[]>([]);
  const [categorySuppliersLoading, setCategorySuppliersLoading] = useState(false);
  const [categorySuppliersError, setCategorySuppliersError] = useState<string | null>(null);

  const productList = products ?? EMPTY_PRODUCTS;
  const categoryList = categories ?? EMPTY_CATEGORIES;
  const supplierList = suppliers ?? EMPTY_SUPPLIERS;
  const cartQuantity = items.reduce((sum, item) => sum + item.quantity, 0);
  const isRefreshing =
    isProductsRefreshing || isCategoriesRefreshing || isSuppliersRefreshing;

  const refreshAll = useCallback(() => {
    void mutateProducts();
    void mutateCategories();
    void mutateSuppliers();
  }, [mutateCategories, mutateProducts, mutateSuppliers]);

  useRetailerSessionReconcile(refreshAll);

  useEffect(() => {
    return subscribe("PROMOTION_CHANGED", () => {
      void mutateProducts();
    });
  }, [mutateProducts, subscribe]);

  useEffect(() => {
    if (!activeSupplier) return;
    void apiFetch("/v1/retailer/promotions/watch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ supplier_id: activeSupplier }),
    });
  }, [activeSupplier]);

  const sparkOrders = useMemo(
    () =>
      Array.from(
        { length: 12 },
        (_, index) => 30 + Math.sin(index * 0.7) * 15 + index * 2,
      ),
    [],
  );
  const sparkRevenue = useMemo(
    () =>
      Array.from(
        { length: 12 },
        (_, index) => 50 + index * 8 + Math.cos(index * 0.5) * 10,
      ),
    [],
  );

  const clearFilters = useCallback(() => {
    setActiveCategory("All");
    setActiveSupplier("");
    setSearchQuery("");
  }, []);

  const filteredProducts = useMemo(() => {
    let list = productList;
    if (activeCategory !== "All") {
      list = list.filter((product) => product.category_name === activeCategory);
    }
    if (activeSupplier) {
      list = list.filter((product) => product.supplier_id === activeSupplier);
    }
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      list = list.filter(
        (product) =>
          product.name.toLowerCase().includes(query) ||
          (product.supplier_name || "").toLowerCase().includes(query),
      );
    }
    return list;
  }, [productList, activeCategory, activeSupplier, searchQuery]);

  const categoryTabs = useMemo(() => {
    const unique = new Set<string>(["All"]);
    categoryList.forEach((category) => {
      if (category.name) unique.add(category.name);
    });
    return Array.from(unique);
  }, [categoryList]);

  const hasActiveFilters =
    activeCategory !== "All" || activeSupplier !== "" || searchQuery.trim() !== "";

  const loadIssue = useMemo<"restricted" | "offline" | "error" | null>(() => {
    const errors = [productsError, categoriesError, suppliersError].filter(
      Boolean,
    ) as Array<Error & { status?: number }>;

    if (errors.length === 0) return null;
    if (errors.some((err) => err.status === 401 || err.status === 403)) {
      return "restricted";
    }
    if (
      (typeof navigator !== "undefined" && !navigator.onLine) ||
      errors.some((err) => /Failed to fetch|NetworkError|Load failed/i.test(err.message))
    ) {
      return "offline";
    }
    return "error";
  }, [categoriesError, productsError, suppliersError]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Catalog access restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: "Offline mode active. Showing the latest cached catalog.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Catalog sync degraded. Auto-retry is active.",
      };
    }
    if (isRefreshing && !loadingProducts) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: "Syncing catalog feeds...",
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loadingProducts]);

  const emptyStateConfig = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: "Catalog access restricted",
        body: "Your account currently cannot load supplier catalog data.",
        variant: "restricted" as const,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: "Catalog is offline",
        body: "Reconnect your network and retry to refresh product availability.",
        variant: "offline" as const,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: "Catalog unavailable",
        body: "Product feeds could not be loaded right now.",
        variant: "error" as const,
      };
    }
    if (productList.length === 0) {
      return {
        headline: "No approved products yet",
        body: "Supplier catalogs are still syncing into your workspace.",
        variant: "no-products" as const,
      };
    }
    return {
      headline: "No assets match criteria",
      body: "Try adjusting the search query or node selection.",
      variant: "no-results" as const,
    };
  }, [loadIssue, productList.length]);

  useEffect(() => {
    if (activeCategory === "All") {
      setCategorySuppliers([]);
      setCategorySuppliersError(null);
      return;
    }
    const category = categoryList.find((item) => item.name === activeCategory);
    if (!category?.id) {
      setCategorySuppliers([]);
      return;
    }
    let cancelled = false;
    setCategorySuppliersLoading(true);
    setCategorySuppliersError(null);
    void apiFetch(`/v1/catalog/categories/${category.id}/suppliers`)
      .then(async (res) => {
        if (!res.ok) {
          throw new Error(`Category suppliers unavailable (${res.status})`);
        }
        return (await res.json()) as Supplier[];
      })
      .then((rows) => {
        if (!cancelled) {
          setCategorySuppliers(Array.isArray(rows) ? rows : []);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setCategorySuppliers([]);
          setCategorySuppliersError(
            err instanceof Error ? err.message : "Category suppliers unavailable",
          );
        }
      })
      .finally(() => {
        if (!cancelled) setCategorySuppliersLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [activeCategory, categoryList]);

  useEffect(() => {
    if (activeSupplier && !supplierList.some((supplier) => supplier.id === activeSupplier)) {
      setActiveSupplier("");
    }
  }, [activeSupplier, supplierList]);

  useEffect(() => {
    if (
      activeCategory !== "All" &&
      !categoryList.some((category) => category.name === activeCategory)
    ) {
      setActiveCategory("All");
    }
  }, [activeCategory, categoryList]);

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="catalog"
        title="Product Catalog"
        description="Explore approved suppliers and stage procurement orders."
        loading={loadingProducts}
        skeletonVariant="catalog"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={isRefreshing}
              onClick={refreshAll}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
              />
              {isRefreshing ? "Syncing" : "Sync"}
            </button>
            <button
              type="button"
              onClick={() => setIsCartOpen(true)}
              className="portal-btn portal-btn--primary h-11 px-6 rounded-xl font-light shadow-[var(--shadow-sm)]"
            >
              <ShoppingCart size={18} className="mr-2" />
              Cart ({cartQuantity})
            </button>
          </div>
        }
      >

      {syncBanner && (
        <motion.div
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          className={`mb-6 flex items-center justify-between gap-3 rounded-2xl border px-4 py-3 ${
            syncBanner.kind === "refreshing"
              ? "border-[var(--desk-info)]/30 bg-[var(--desk-info)]/5 text-[var(--desk-info)]"
              : "border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 text-[var(--desk-warning)]"
          }`}
        >
          <div className="flex items-center gap-2">
            <syncBanner.icon
              size={16}
              className={syncBanner.kind === "refreshing" ? "animate-spin" : ""}
            />
            <span className="md-typescale-body-small font-light uppercase tracking-wide">
              {syncBanner.message}
            </span>
          </div>
          {syncBanner.kind !== "refreshing" && (
            <button
              onClick={refreshAll}
              className="rounded-lg border border-current/30 px-3 py-1 text-[11px] font-light uppercase tracking-wide hover:bg-current/10"
            >
              Retry
            </button>
          )}
        </motion.div>
      )}

      <BentoGrid className="mb-8">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Available
              </span>
              <Package size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <CountUp
              end={productList.length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <span className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              {supplierList.length} global suppliers
            </span>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Categories
              </span>
              <Layers size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={categoryList.length}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkOrders} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Staged
              </span>
              <TrendingUp size={18} style={{ color: "var(--desk-success)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={cartQuantity}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkRevenue} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Active
              </span>
              <Star size={18} style={{ color: "var(--desk-warning)" }} />
            </div>
            <CountUp
              end={supplierList.filter((s) => s.is_active).length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <span className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Operational nodes
            </span>
          </div>
        </BentoCard>
      </BentoGrid>

      <CatalogFilters
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        hasActiveFilters={hasActiveFilters}
        clearFilters={clearFilters}
        categoryTabs={categoryTabs}
        activeCategory={activeCategory}
        setActiveCategory={setActiveCategory}
        categorySuppliersLoading={categorySuppliersLoading}
        categorySuppliersError={categorySuppliersError}
        categorySuppliers={categorySuppliers}
        activeSupplier={activeSupplier}
        setActiveSupplier={setActiveSupplier}
        supplierList={supplierList}
      />
      <div className="flex gap-8">
        <aside className="hidden w-[280px] shrink-0 lg:flex lg:flex-col lg:gap-4">
          <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
            <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-4">
              Supplier Nodes
            </h3>
            <div className="flex flex-col gap-1">
              <button
                onClick={() => setActiveSupplier("")}
                className={`flex items-center justify-between w-full h-10 px-3 rounded-lg transition-all ${
                  activeSupplier === ""
                    ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-light"
                    : "text-[var(--desk-text-secondary)] hover:bg-[var(--desk-surface-subtle)]"
                }`}
              >
                <span className="md-typescale-body-medium">Global Network</span>
                <Building2 size={16} />
              </button>
              {supplierList.slice(0, 8).map((supplier) => (
                <button
                  key={supplier.id}
                  onClick={() => setActiveSupplier(supplier.id)}
                  className={`flex items-center justify-between w-full h-10 px-3 rounded-lg transition-all ${
                    activeSupplier === supplier.id
                      ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-light"
                      : "text-[var(--desk-text-secondary)] hover:bg-[var(--desk-surface-subtle)]"
                  }`}
                >
                  <span className="truncate md-typescale-body-medium">
                    {supplier.name}
                  </span>
                  <ChevronRight size={14} opacity={0.5} />
                </button>
              ))}
            </div>
          </div>
        </aside>

        <PageSection
          title="Browse products"
          description={`${filteredProducts.length} SKUs match your filters.`}
          className="flex-1 min-w-0"
        >
          <ProductGrid
            loadingProducts={loadingProducts}
            filteredProducts={filteredProducts}
            emptyStateConfig={emptyStateConfig}
            loadIssue={loadIssue}
            refreshAll={refreshAll}
            hasActiveFilters={hasActiveFilters}
            clearFilters={clearFilters}
            setSelectedProduct={setSelectedProduct}
            addToCart={addToCart}
          />
          
        </PageSection>
      </div>

      <CartDrawer
        isOpen={isCartOpen}
        onClose={() => setIsCartOpen(false)}
        onCheckout={() => {
          setIsCartOpen(false);
          setIsCheckoutOpen(true);
        }}
      />
      <CheckoutModal
        isOpen={isCheckoutOpen}
        onClose={() => setIsCheckoutOpen(false)}
        total={items.reduce((sum, item) => sum + item.price * item.quantity, 0)}
      />
      <ProductDetailDrawer
        product={selectedProduct}
        isOpen={!!selectedProduct}
        onClose={() => setSelectedProduct(null)}
      />
      </PageChrome>
    </div>
  );
}

