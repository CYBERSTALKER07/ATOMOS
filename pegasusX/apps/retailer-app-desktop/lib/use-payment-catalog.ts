"use client";

import { packCurrency, readCachedAuthSession, selectablePackPsps } from "@pegasusx/api-client";
import type { PSPListing, RetailerPaymentCatalogResponse } from "@pegasusx/types";
import { useEffect, useState } from "react";
import { apiFetch } from "./auth";
import { retailerCatalogGateways } from "./payment-catalog";

export function useRetailerPaymentCatalog() {
  const pack = readCachedAuthSession()?.pack;
  const [catalog, setCatalog] = useState<PSPListing[]>([]);
  const [currency, setCurrency] = useState(packCurrency(pack));

  useEffect(() => {
    let cancelled = false;
    void apiFetch("/v1/retailer/payment-catalog")
      .then((res) => (res.ok ? res.json() : null))
      .then((data: RetailerPaymentCatalogResponse | null) => {
        if (cancelled) return;
        if (!data) {
          const fallback = (pack?.psp_adapters ?? []).map((code) => ({
            code,
            status: "live",
            selectable: true,
          }));
          setCatalog(fallback);
          setCurrency(packCurrency(pack));
          return;
        }
        setCatalog(data.catalog ?? []);
        setCurrency(data.currency_code || packCurrency(pack));
      })
      .catch(() => {
        if (cancelled) return;
        setCatalog(
          (pack?.psp_adapters ?? []).map((code) => ({
            code,
            status: "live",
            selectable: true,
          })),
        );
        setCurrency(packCurrency(pack));
      });
    return () => {
      cancelled = true;
    };
  }, [pack?.code, pack?.currency_code]);

  const gateways = retailerCatalogGateways(
    catalog.length
      ? catalog
      : (pack?.psp_adapters ?? []).map((code) => ({ code, status: "live", selectable: true })),
  );

  return {
    pack,
    catalog,
    currency: currency || packCurrency(pack),
    gateways: gateways.length ? gateways : selectablePackPsps(
      (pack?.psp_adapters ?? []).map((code) => ({ code, status: "live", selectable: true })),
    ),
    allowsCash: gateways.includes("CASH") || gateways.length === 0,
    allowsGlobalPay: gateways.includes("GLOBAL_PAY"),
  };
}
