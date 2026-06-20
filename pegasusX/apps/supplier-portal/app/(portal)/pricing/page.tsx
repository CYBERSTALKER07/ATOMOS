"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { CatalogProduct } from "@pegasusx/types";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import { DataList, DataListRow } from "@/components/portal";

function formatPrice(product: CatalogProduct) {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: product.currency,
      maximumFractionDigits: 2,
    }).format(product.price_minor / 100);
  } catch {
    return `${product.price_minor} ${product.currency}`;
  }
}

export default function PricingPage() {
  const [products, setProducts] = useState<CatalogProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    supplierFetch("/v1/catalog/products")
      .then(async (res) => {
        if (!res.ok) throw new Error(`load_failed_${res.status}`);
        const rows = (await res.json()) as CatalogProduct[];
        setProducts(Array.isArray(rows) ? rows : []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_pricing_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      title="Pricing"
      description="Set list and sale pricing per catalog product. Add products in Catalog first."
      loading={loading}
      error={error}
      empty={products.length === 0}
      emptyMessage="No products to price. Create products in Catalog, then return here."
      actions={
        <Link href="/catalog" className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2 inline-flex">
          Open catalog
        </Link>
      }
    >
      <DataList>
        {products.map((product) => (
          <DataListRow key={product.product_id}>
            <Link
              href={`/pricing/${encodeURIComponent(product.product_id)}`}
              className="flex flex-1 items-center justify-between gap-4 min-w-0 text-inherit no-underline"
            >
              <div className="min-w-0">
                <div className="md-typescale-body-large font-medium">{product.name}</div>
                <div className="text-[var(--color-md-outline)] text-sm mt-1">{product.product_id}</div>
              </div>
              <div className="md-typescale-title-medium shrink-0">{formatPrice(product)}</div>
            </Link>
          </DataListRow>
        ))}
      </DataList>
    </PageChrome>
  );
}
