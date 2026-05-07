"use client";

import { useMemo, useState } from "react";
import {
  ShoppingCart,
  Search,
  SlidersHorizontal,
  Package,
  Star,
  Layers,
  TrendingUp,
  Building2,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import CartDrawer from "../../../components/CartDrawer";
import CheckoutModal from "../../../components/CheckoutModal";
import ProductDetailDrawer from "../../../components/ProductDetailDrawer";
import EmptyState from "../../../components/EmptyState";
import PageTransition from "../../../components/PageTransition";
import { motion } from "framer-motion";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import type { Product, Category, Supplier } from "../../../lib/types";

const EMPTY_PRODUCTS: Product[] = [];
const EMPTY_CATEGORIES: Category[] = [];
const EMPTY_SUPPLIERS: Supplier[] = [];

export default function CatalogPage() {
  const { data: products, loading: loadingProducts } = useLiveData<Product[]>("/v1/catalog/products");
  const { data: categories } = useLiveData<Category[]>("/v1/catalog/categories");
  const { data: suppliers } = useLiveData<Supplier[]>("/v1/retailer/suppliers");
  const { items, addToCart } = useCart();

  const [isCartOpen, setIsCartOpen] = useState(false);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState("All");
  const [activeSupplier, setActiveSupplier] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);

  const productList = products ?? EMPTY_PRODUCTS;
  const categoryList = categories ?? EMPTY_CATEGORIES;
  const supplierList = suppliers ?? EMPTY_SUPPLIERS;
  const cartQuantity = items.reduce((sum, item) => sum + item.quantity, 0);

  const sparkOrders = useMemo(() => Array.from({ length: 12 }, (_, index) => 30 + Math.sin(index * 0.7) * 15 + index * 2), []);
  const sparkRevenue = useMemo(() => Array.from({ length: 12 }, (_, index) => 50 + index * 8 + Math.cos(index * 0.5) * 10), []);

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
          product.supplier_name.toLowerCase().includes(query),
      );
    }
    return list;
  }, [productList, activeCategory, activeSupplier, searchQuery]);

  const categoryTabs = useMemo(() => ["All", ...categoryList.map((category) => category.name)], [categoryList]);
  const activeSupplierRecord = supplierList.find((supplier) => supplier.id === activeSupplier) ?? null;

  if (loadingProducts) {
    return (
      <div className="min-h-full p-6 md:p-8">
        <Skeleton className="mb-2 h-8 w-64 rounded-lg" />
        <Skeleton className="mb-8 h-4 w-96 rounded-lg" />
        <div className="mb-8 grid grid-cols-4 gap-4">
          {[0, 1, 2, 3].map((item) => (
            <Skeleton key={item} className="h-28 rounded-2xl" />
          ))}
        </div>
        <div className="flex gap-6">
          <div className="hidden w-[280px] flex-col gap-2 lg:flex">
            {[0, 1, 2, 3].map((item) => (
              <Skeleton key={item} className="h-16 rounded-2xl" />
            ))}
          </div>
          <div className="grid flex-1 grid-cols-3 gap-4">
            {[0, 1, 2, 3, 4, 5].map((item) => (
              <Skeleton key={item} className="h-64 rounded-2xl" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <PageTransition className="min-h-full p-6 md:p-8">
      <header className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-large">Supplier catalog</h1>
          <p className="mt-1 md-typescale-body-medium text-muted">
            Search approved suppliers, compare stock status, and stage replenishment with desktop-grade controls.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="primary"
            onPress={() => setIsCartOpen(true)}
            className="md-btn md-btn-filled md-typescale-label-large flex h-10 items-center gap-2 px-5"
          >
            <ShoppingCart size={18} />
            Cart ({items.length})
          </Button>
        </div>
      </header>

      <BentoGrid className="mb-8">
        <BentoCard>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Products available</span>
              <Package size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <CountUp end={productList.length} className="md-kpi-value" />
            <span className="md-kpi-sub">{supplierList.length} suppliers in view</span>
          </div>
        </BentoCard>

        <BentoCard>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Categories</span>
              <Layers size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={categoryList.length} className="md-kpi-value" />
              <MiniSparkline data={sparkOrders} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">Active product groups</span>
          </div>
        </BentoCard>

        <BentoCard>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">In cart</span>
              <TrendingUp size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={cartQuantity} className="md-kpi-value" />
              <MiniSparkline data={sparkRevenue} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">Items ready to order</span>
          </div>
        </BentoCard>

        <BentoCard>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Active suppliers</span>
              <Star size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <CountUp end={supplierList.filter((supplier) => supplier.is_active).length} className="md-kpi-value" />
            <span className="md-kpi-sub">{filteredProducts.length} products match current filters</span>
          </div>
        </BentoCard>
      </BentoGrid>

      <div className="mb-6 flex flex-wrap items-center gap-3 border-b border-[var(--border)] pb-3">
        <div className="md-search-bar max-w-sm flex-1">
          <Search size={18} />
          <input
            type="text"
            placeholder="Search products or suppliers..."
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
          />
        </div>

        <label className="flex h-11 min-w-[240px] items-center gap-3 rounded-xl border border-[var(--border)] bg-[var(--surface)] px-3 text-sm text-muted">
          <Building2 size={16} />
          <span className="md-typescale-label-large whitespace-nowrap">Supplier</span>
          <select
            value={activeSupplier}
            onChange={(event) => setActiveSupplier(event.target.value)}
            className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none"
          >
            <option value="">All suppliers</option>
            {supplierList.map((supplier) => (
              <option key={supplier.id} value={supplier.id}>
                {supplier.name}
              </option>
            ))}
          </select>
        </label>

        <Button
          variant="ghost"
          className="text-muted md-typescale-label-large flex items-center gap-2"
          onPress={() => {
            setActiveCategory("All");
            setActiveSupplier("");
            setSearchQuery("");
          }}
        >
          <SlidersHorizontal size={16} />
          Reset filters
        </Button>
      </div>

      <div className="mb-6 flex flex-wrap items-center gap-2">
        {categoryTabs.map((category) => (
          <button
            key={category}
            onClick={() => setActiveCategory(category)}
            className={`rounded-full px-4 py-2 md-typescale-label-large font-semibold transition-colors ${
              activeCategory === category
                ? "bg-accent text-accent-foreground"
                : "text-muted hover:bg-surface hover:text-foreground"
            }`}
          >
            {category}
          </button>
        ))}
      </div>

      <div className="flex gap-6">
        <aside className="hidden w-[280px] shrink-0 lg:flex lg:flex-col lg:gap-3">
          <div className="bento-card">
            <p className="md-typescale-label-small uppercase tracking-[0.16em] text-muted">Supplier shortcuts</p>
            <div className="mt-4 flex flex-col gap-2">
              <button
                onClick={() => setActiveSupplier("")}
                className={`md-nav-item w-full ${activeSupplier === "" ? "md-nav-active" : ""}`}
                data-active={activeSupplier === ""}
              >
                <Building2 size={18} />
                <span>All suppliers</span>
              </button>
              {supplierList.slice(0, 6).map((supplier) => (
                <button
                  key={supplier.id}
                  onClick={() => setActiveSupplier(supplier.id)}
                  className={`md-nav-item w-full ${activeSupplier === supplier.id ? "md-nav-active" : ""}`}
                  data-active={activeSupplier === supplier.id}
                >
                  <Building2 size={18} />
                  <span className="truncate">{supplier.name}</span>
                </button>
              ))}
            </div>
          </div>
        </aside>

        <section className="min-w-0 flex-1">
          <div className="sticky top-0 z-10 mb-4 flex flex-wrap items-end justify-between gap-3 bg-background/85 py-2 backdrop-blur-sm">
            <div>
              <h2 className="md-typescale-title-large font-semibold text-foreground">{activeCategory}</h2>
              <p className="md-typescale-body-small text-muted">
                {filteredProducts.length} products
                {activeSupplierRecord ? ` from ${activeSupplierRecord.name}` : ` from ${supplierList.length} suppliers`}
              </p>
            </div>
            {activeSupplierRecord && (
              <Chip size="sm" color="default" variant="soft">
                Supplier filter: {activeSupplierRecord.name}
              </Chip>
            )}
          </div>

          {filteredProducts.length === 0 ? (
            <div className="py-16">
              <EmptyState 
                imageUrl="/images/empty-products.png"
                headline="No products found"
                body="Try a different supplier, category, or search query."
                action="Clear Filters"
                onAction={() => {
                  setActiveCategory("All");
                  setActiveSupplier("");
                  setSearchQuery("");
                }}
              />
            </div>
          ) : (
            <motion.div 
              className="grid gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
              initial="hidden"
              animate="show"
              variants={{
                hidden: { opacity: 0 },
                show: { opacity: 1, transition: { staggerChildren: 0.05 } }
              }}
            >
              {filteredProducts.map((product) => (
                <motion.article
                  key={product.id}
                  variants={{
                    hidden: { opacity: 0, y: 20 },
                    show: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 24 } }
                  }}
                  onClick={() => setSelectedProduct(product)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      setSelectedProduct(product);
                    }
                  }}
                  role="button"
                  tabIndex={0}
                  className="bento-card flex cursor-pointer flex-col gap-4 text-left transition-all duration-150 hover:ring-1 hover:ring-accent"
                  style={{ padding: 0, overflow: "hidden" }}
                >
                  <div className="relative flex h-36 items-center justify-center" style={{ background: "var(--surface)" }}>
                    {product.image_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={product.image_url} alt={product.name} className="h-full w-full object-cover" />
                    ) : (
                      <Package size={32} style={{ color: "var(--muted)", opacity: 0.4 }} />
                    )}
                    <div className="absolute left-3 top-3">
                      <StockBadge stock={product.available_stock} />
                    </div>
                  </div>

                  <div className="flex flex-1 flex-col justify-between gap-4 p-4">
                    <div className="space-y-1">
                      <p className="line-clamp-2 md-typescale-title-small font-semibold text-foreground">{product.name}</p>
                      <p className="md-typescale-body-small text-muted">{product.supplier_name}</p>
                      <p className="md-typescale-label-small text-muted">{product.category_name}</p>
                    </div>

                    <div className="flex items-end justify-between gap-3">
                      <div>
                        <p className="md-typescale-label-small uppercase tracking-[0.14em] text-muted">Unit price</p>
                        <p className="md-typescale-title-medium font-bold tabular-nums text-foreground">
                          {product.price.toLocaleString()}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="hidden md-typescale-label-small text-muted md:inline-flex">View details</span>
                        <Button
                          variant="primary"
                          className="md-btn md-btn-filled h-10 px-4"
                          isDisabled={product.available_stock !== undefined && product.available_stock <= 0}
                          onClick={(event) => {
                            event.stopPropagation();
                            addToCart(product);
                          }}
                        >
                          Add
                        </Button>
                      </div>
                    </div>
                  </div>
                </motion.article>
              ))}
            </motion.div>
          )}
        </section>
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
    </PageTransition>
  );
}

function StockBadge({ stock }: { stock?: number }) {
  if (stock !== undefined && stock <= 0) {
    return (
      <span
        className="rounded-full px-2.5 py-1 md-typescale-label-small font-semibold"
        style={{ background: "var(--danger)", color: "var(--danger-foreground)" }}
      >
        Out of stock
      </span>
    );
  }

  if (stock !== undefined && stock <= 5) {
    return (
      <span
        className="rounded-full px-2.5 py-1 md-typescale-label-small font-semibold"
        style={{ background: "var(--warning)", color: "var(--warning-foreground)" }}
      >
        Low stock
      </span>
    );
  }

  return (
    <span className="rounded-full bg-[var(--surface)] px-2.5 py-1 md-typescale-label-small font-semibold text-foreground">
      Ready
    </span>
  );
}
