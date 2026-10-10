"use client";
import { useMarketPack } from '@pegasusx/api-react';


import { packCurrency, selectablePackPsps } from '@pegasusx/api-core';
import type { PSPListing } from "@pegasusx/types";
import { useEffect, useState } from "react";
import { getSupplierToken, supplierApiBaseUrl } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";

export function useSupplierPaymentCatalog() {
  const token = getSupplierToken();
  const { pack } = useMarketPack({ baseUrl: supplierApiBaseUrl(), token });
  const [catalog, setCatalog] = useState<PSPListing[]>([]);
  const [currency, setCurrency] = useState("");

  useEffect(() => {
    let cancelled = false;
    void createSupplierApi()
      .getSupplierPaymentCatalog()
      .then((resp) => {
        if (cancelled) return;
        setCatalog(resp.catalog ?? []);
        setCurrency(resp.currency_code || packCurrency(pack));
      })
      .catch(() => {
        if (cancelled) return;
        const fallback = (pack?.psp_adapters ?? []).map((code) => ({
          code,
          status: "live",
          selectable: true,
        }));
        setCatalog(fallback);
        setCurrency(packCurrency(pack));
      });
    return () => {
      cancelled = true;
    };
  }, [pack?.code, pack?.currency_code]);

  return {
    pack,
    catalog,
    currency: currency || packCurrency(pack),
    gateways: selectablePackPsps(catalog.length ? catalog : (pack?.psp_adapters ?? []).map((code) => ({ code, selectable: true }))),
  };
}
