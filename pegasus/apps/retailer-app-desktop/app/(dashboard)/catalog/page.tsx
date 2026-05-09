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
  ChevronRight,
} from "lucide-react";
import { Button, Skeleton } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import CartDrawer from "../../../components/CartDrawer";
import CheckoutModal from "../../../components/CheckoutModal";
import ProductDetailDrawer from "../../../components/ProductDetailDrawer";
import EmptyState from "../../../components/EmptyState";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import type { Product, Category, Supplier } from "../../../lib/types";

const EMPTY_PRODUCTS: Product[] = [];
const EMPTY_CATEGORIES: Category[] = [];
const EMPTY_SUPPLIERS: Supplier[] = [];

export default function CatalogPage() {
  const { data: products, loading: loadingProducts } = useLiveData<Product[]>(
    "/v1/catalog/products",
  );
  const { data: categories } = useLiveData<Category[]>(
    "/v1/catalog/categories",
  );
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

  const categoryTabs = useMemo(
    () => ["All", ...categoryList.map((category) => category.name)],
    [categoryList],
  );

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-display-small font-bold tracking-tight text-[var(--desk-text-primary)]">
            Product Catalog
          </h1>
          <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
            Explore approved suppliers and stage procurement orders.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="solid"
            onPress={() => setIsCartOpen(true)}
            className="h-11 px-6 rounded-xl font-bold transition-all active:scale-95 shadow-[var(--shadow-sm)]"
            style={{ background: "var(--desk-accent)", color: "white" }}
          >
            <ShoppingCart size={18} className="mr-2" />
            Cart ({items.length})
          </Button>
        </div>
      </header>

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

      <div className="mb-8 flex flex-col gap-6">
        <div className="flex flex-wrap items-center gap-4 p-2 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
          <div className="flex-1 min-w-[280px] relative group">
            <Search
              className="absolute left-4 top-1/2 -translate-y-1/2 text-[var(--desk-text-tertiary)] group-focus-within:text-[var(--desk-accent)] transition-colors"
              size={18}
            />
            <input
              type="text"
              placeholder="Search assets and suppliers..."
              className="w-full h-11 pl-11 pr-4 bg-[var(--desk-canvas)] rounded-xl outline-none focus:ring-2 focus:ring-[var(--desk-accent-soft)] transition-all md-typescale-body-medium text-[var(--desk-text-primary)]"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          <div className="h-11 flex items-center gap-2">
            <button
              onClick={() => {
                setActiveCategory("All");
                setActiveSupplier("");
                setSearchQuery("");
              }}
              className="px-4 h-full rounded-xl hover:bg-[var(--desk-surface-subtle)] text-[var(--desk-text-secondary)] md-typescale-label-large transition-colors flex items-center gap-2"
            >
              <SlidersHorizontal size={16} />
              Reset
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {categoryTabs.map((category) => (
            <button
              key={category}
              onClick={() => setActiveCategory(category)}
              className={`px-5 py-2 rounded-full md-typescale-label-large font-bold transition-all ${
                activeCategory === category
                  ? "bg-[var(--desk-accent)] text-white shadow-[var(--shadow-sm)] scale-105"
                  : "bg-[var(--desk-surface)] text-[var(--desk-text-secondary)] border border-[var(--desk-border)] hover:bg-[var(--desk-surface-subtle)]"
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      </div>

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
                    ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-bold"
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
                      ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-bold"
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

        <section className="min-w-0 flex-1">
          <AnimatePresence mode="popLayout">
            {loadingProducts ? (
              <motion.div
                key="loading"
                className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6"
              >
                {[0, 1, 2, 3, 4, 5].map((i) => (
                  <div
                    key={i}
                    className="h-72 rounded-2xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]"
                  />
                ))}
              </motion.div>
            ) : filteredProducts.length === 0 ? (
              <motion.div
                key="empty"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="py-20"
              >
                <EmptyState
                  headline="No assets match criteria"
                  body="Try adjusting the search query or node selection."
                  variant="no-results"
                />
              </motion.div>
            ) : (
              <motion.div
                key="list"
                layout
                className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6"
              >
                {filteredProducts.map((product) => (
                  <motion.article
                    key={product.id}
                    layout
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.95 }}
                    onClick={() => setSelectedProduct(product)}
                    className="group bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl overflow-hidden cursor-pointer hover:shadow-md hover:-translate-y-1 transition-all active:scale-[0.98]"
                  >
                    <div className="relative h-44 bg-[var(--desk-surface-subtle)] overflow-hidden">
                      {product.image_url ? (
                        <img
                          src={product.image_url}
                          alt={product.name}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center opacity-20">
                          <Package size={48} />
                        </div>
                      )}
                      <div className="absolute top-3 right-3">
                        <StockBadge stock={product.available_stock} />
                      </div>
                    </div>

                    <div className="p-5 flex flex-col gap-4">
                      <div className="space-y-1">
                        <h3 className="md-typescale-title-medium font-bold text-[var(--desk-text-primary)] line-clamp-1 group-hover:text-[var(--desk-accent)] transition-colors">
                          {product.name}
                        </h3>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                          {product.supplier_name}
                        </p>
                      </div>

                      <div className="flex items-end justify-between">
                        <div className="flex flex-col">
                          <span className="md-typescale-label-small text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                            Price Point
                          </span>
                          <span className="md-typescale-title-medium font-bold tabular-nums text-[var(--desk-text-primary)]">
                            {product.price.toLocaleString()}{" "}
                            <small className="text-[var(--desk-text-tertiary)] ml-0.5">
                              UZS
                            </small>
                          </span>
                        </div>
                        <Button
                          variant="solid"
                          size="sm"
                          className="rounded-lg h-9 px-4 font-bold shadow-[var(--shadow-sm)] transition-all active:scale-95"
                          style={{
                            background: "var(--desk-text-primary)",
                            color: "white",
                          }}
                          onClick={(e) => {
                            e.stopPropagation();
                            addToCart(product);
                          }}
                          isDisabled={
                            product.available_stock != null &&
                            product.available_stock <= 0
                          }
                        >
                          Add to Cart
                        </Button>
                      </div>
                    </div>
                  </motion.article>
                ))}
              </motion.div>
            )}
          </AnimatePresence>
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
    </div>
  );
}

function StockBadge({ stock }: { stock?: number }) {
  if (stock !== undefined && stock <= 0) {
    return (
      <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-danger)] text-white md-typescale-label-small font-bold uppercase tracking-widest shadow-[var(--shadow-sm)]">
        Empty
      </span>
    );
  }

  if (stock !== undefined && stock <= 5) {
    return (
      <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-warning)] text-white md-typescale-label-small font-bold uppercase tracking-widest shadow-[var(--shadow-sm)]">
        Critical
      </span>
    );
  }

  return (
    <span className="px-2.5 py-1 rounded-lg bg-[var(--desk-success)] text-white md-typescale-label-small font-bold uppercase tracking-widest shadow-[var(--shadow-sm)]">
      Stable
    </span>
  );
}
