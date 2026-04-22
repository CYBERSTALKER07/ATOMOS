"use client";

import { useState, useMemo } from "react";
import {
  ShoppingCart, Search, SlidersHorizontal, ChevronRight,
  Package, Star, ArrowUpRight, Layers, TrendingUp, AlertTriangle,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import CartDrawer from "../../../components/CartDrawer";
import CheckoutModal from "../../../components/CheckoutModal";
import ProductDetailDrawer from "../../../components/ProductDetailDrawer";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import type { Product, Category, Supplier } from "../../../lib/types";

export default function CatalogPage() {
  const { data: products, loading: loadingProducts } = useLiveData<Product[]>("/v1/catalog/products");
  const { data: categories } = useLiveData<Category[]>("/v1/catalog/categories");
  const { data: suppliers } = useLiveData<Supplier[]>("/v1/retailer/suppliers");
  const { items, addToCart } = useCart();

  const [isCartOpen, setIsCartOpen] = useState(false);
  const [isCheckoutOpen, setIsCheckoutOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState("All");
  const [activeSupplier, setActiveSupplier] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);

  const productList = products ?? [];
  const categoryList = categories ?? [];
  const supplierList = suppliers ?? [];

  const sparkOrders = useMemo(() => Array.from({ length: 12 }, (_, i) => 30 + Math.sin(i * 0.7) * 15 + i * 2), []);
  const sparkRevenue = useMemo(() => Array.from({ length: 12 }, (_, i) => 50 + i * 8 + Math.cos(i * 0.5) * 10), []);

  const filtered = useMemo(() => {
    let list = productList;
    if (activeCategory !== "All") {
      list = list.filter((p) => p.category_name === activeCategory);
    }
    if (activeSupplier) {
      list = list.filter((p) => p.supplier_id === activeSupplier);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter((p) => p.name.toLowerCase().includes(q) || p.supplier_name.toLowerCase().includes(q));
    }
    return list;
  }, [productList, activeCategory, activeSupplier, searchQuery]);

  const categoryTabs = useMemo(() => {
    const names = ["All", ...categoryList.map((c) => c.name)];
    return names;
  }, [categoryList]);

  /* ── Loading skeleton ── */
  if (loadingProducts) {
    return (
      <div className="min-h-full p-6 md:p-8">
        <Skeleton className="h-8 w-64 rounded-lg mb-2" />
        <Skeleton className="h-4 w-96 rounded-lg mb-8" />
        <div className="grid grid-cols-4 gap-4 mb-8">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-28 rounded-2xl" />)}
        </div>
        <div className="flex gap-6">
          <div className="w-[240px] flex flex-col gap-2">
            {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-16 rounded-2xl" />)}
          </div>
          <div className="flex-1 grid grid-cols-3 gap-4">
            {[0, 1, 2, 3, 4, 5].map((i) => <Skeleton key={i} className="h-64 rounded-2xl" />)}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8">
      {/* ── Header ── */}
      <header className="mb-6 flex items-end justify-between gap-4 flex-wrap">
        <div>
          <h1 className="md-typescale-headline-large">Supplier Catalog</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: "var(--muted)" }}>
            Browse and procure inventory directly from approved suppliers.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="primary"
            onPress={() => setIsCartOpen(true)}
            className="md-btn md-btn-filled md-typescale-label-large px-5 h-10 flex items-center gap-2"
          >
            <ShoppingCart size={18} />
            Cart ({items.length})
          </Button>
        </div>
      </header>

      {/* ── KPI Bento ── */}
      <BentoGrid className="mb-8">
        <BentoCard delay={0}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Products Available</span>
              <Package size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <CountUp end={productList.length} className="md-kpi-value" />
            <span className="md-kpi-sub">{supplierList.length} suppliers</span>
          </div>
        </BentoCard>

        <BentoCard delay={60}>
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

        <BentoCard delay={120}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">In Cart</span>
              <TrendingUp size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={items.reduce((s, i) => s + i.quantity, 0)} className="md-kpi-value" suffix=" items" />
              <MiniSparkline data={sparkRevenue} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">Ready to order</span>
          </div>
        </BentoCard>

        <BentoCard delay={180}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Suppliers</span>
              <Star size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <CountUp end={supplierList.filter((s) => s.is_active).length} className="md-kpi-value" />
            <span className="md-kpi-sub">Active partners</span>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* ── Search + Filters ── */}
      <div className="flex items-center gap-3 mb-6 border-b border-[var(--border)] pb-3 flex-wrap">
        <div className="md-search-bar flex-1 max-w-sm">
          <Search size={18} />
          <input
            type="text"
            placeholder="Search products..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-2 overflow-x-auto">
          {categoryTabs.map((cat) => (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat)}
              className={`md-typescale-label-large px-4 py-2 rounded-full font-semibold transition-colors cursor-pointer whitespace-nowrap ${
                activeCategory === cat
                  ? "bg-accent text-accent-foreground"
                  : "text-muted hover:text-foreground hover:bg-surface"
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
        <div className="flex-1" />
        <Button
          variant="ghost"
          className="text-muted md-typescale-label-large flex items-center gap-2"
          onPress={() => { setActiveCategory("All"); setActiveSupplier(null); setSearchQuery(""); }}
        >
          <SlidersHorizontal size={16} /> Reset Filters
        </Button>
      </div>

      {/* ── Split: Supplier List + Product Grid ── */}
      <div className="flex gap-6 min-h-[520px]">

        {/* Left: Supplier sidebar */}
        <div className="w-[240px] shrink-0 hidden lg:flex flex-col gap-1">
          <span className="md-typescale-label-small font-semibold uppercase tracking-widest mb-2" style={{ color: "var(--muted)" }}>
            Top Suppliers
          </span>
          {supplierList.slice(0, 6).map((sup) => (
            <button
              key={sup.id}
              onClick={() => setActiveSupplier(activeSupplier === sup.id ? null : sup.id)}
              className={`bento-card text-left cursor-pointer transition-all duration-150 ${activeSupplier === sup.id ? "ring-1 ring-accent" : ""}`}
              style={{ padding: "12px 16px" }}
            >
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" style={{ background: "var(--surface)" }}>
                  <Package size={16} style={{ color: "var(--muted)" }} />
                </div>
                <div className="flex-1 min-w-0">
                  <span className="md-typescale-title-small font-semibold text-foreground truncate block">{sup.name}</span>
                  <span className="md-typescale-body-small text-muted">{sup.order_count} orders</span>
                </div>
                <ChevronRight size={16} style={{ color: "var(--muted)" }} />
              </div>
            </button>
          ))}
        </div>

        {/* Right: Product Grid */}
        <div className="flex-1 overflow-y-auto max-h-[calc(100dvh-460px)] pr-1">
          <div className="flex items-center justify-between mb-4 sticky top-0 z-10 py-2 bg-background/80 backdrop-blur-sm">
            <div>
              <h2 className="md-typescale-title-large font-semibold text-foreground">{activeCategory}</h2>
              <p className="md-typescale-body-small text-muted">{filtered.length} products from {supplierList.length} suppliers</p>
            </div>
          </div>

          {filtered.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 gap-3">
              <Package size={40} style={{ color: "var(--muted)" }} />
              <p className="md-typescale-title-medium text-foreground">No products found</p>
              <p className="md-typescale-body-medium text-muted">Try adjusting your search or filters.</p>
            </div>
          ) : (
            <div className="grid grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-4 pb-8">
              {filtered.map((product) => (
                <div key={product.id} onClick={() => setSelectedProduct(product)} className="bento-card flex flex-col cursor-pointer group" style={{ padding: 0, overflow: "hidden" }}>
                  {/* Image area */}
                  <div className="h-36 flex items-center justify-center relative" style={{ background: "var(--surface)" }}>
                    {product.image_url ? (
                      <img src={product.image_url} alt={product.name} className="w-full h-full object-cover" />
                    ) : (
                      <Package size={32} style={{ color: "var(--muted)", opacity: 0.4 }} />
                    )}
                    {product.available_stock !== undefined && product.available_stock <= 0 && (
                      <div className="absolute inset-0 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.45)' }}>
                        <span className="md-typescale-label-large font-bold text-white">Out of Stock</span>
                      </div>
                    )}
                    {product.available_stock !== undefined && product.available_stock > 0 && product.available_stock <= 5 && (
                      <span className="absolute bottom-2 left-2 md-typescale-label-small font-bold px-2 py-0.5 rounded-full" style={{ background: 'var(--danger)', color: 'white' }}>
                        Low Stock
                      </span>
                    )}
                  </div>

                  {/* Content */}
                  <div className="p-4 flex-1 flex flex-col justify-between gap-3">
                    <div>
                      <p className="md-typescale-title-small font-semibold text-foreground leading-snug line-clamp-2">
                        {product.name}
                      </p>
                      <p className="md-typescale-body-small text-muted mt-1">{product.supplier_name}</p>
                    </div>

                    <div className="flex items-center justify-between">
                      <div>
                        <span className="md-typescale-title-medium font-bold text-foreground tabular-nums">
                          {product.price.toLocaleString()}
                        </span>
                        {product.category_name && (
                          <div className="flex items-center gap-1 mt-0.5">
                            <span className="md-typescale-label-small text-muted">{product.category_name}</span>
                          </div>
                        )}
                      </div>
                      <Button
                        isIconOnly
                        variant="secondary"
                        className="bg-accent text-accent-foreground rounded-full w-9 h-9 min-w-0 font-bold text-lg"
                        isDisabled={product.available_stock !== undefined && product.available_stock <= 0}
                        onPress={() => addToCart(product)}
                      >
                        +
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
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
        total={items.reduce((s, i) => s + i.price * i.quantity, 0)}
      />
      <ProductDetailDrawer
        product={selectedProduct}
        isOpen={!!selectedProduct}
        onClose={() => setSelectedProduct(null)}
      />
    </div>
  );
}
