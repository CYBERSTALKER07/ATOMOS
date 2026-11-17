import { ApiClient } from '@pegasusx/api-core';
import { getSupplierToken, supplierApiBaseUrl, supplierSessionFetch } from "@/lib/auth";

export function createSupplierApi(): ApiClient {
  return new ApiClient({
    baseUrl: supplierApiBaseUrl(),
    getAuthToken: () => getSupplierToken() || null,
    fetchImpl: supplierSessionFetch,
  });
}

export type * from "./finance-api";
export type * from "./ontology-api";
export type * from "./control-accounting-api";
export type * from "./intel-roi-api";
