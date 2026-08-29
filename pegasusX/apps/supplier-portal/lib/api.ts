import { ApiClient } from '@pegasusx/api-core';
import { getSupplierToken, supplierApiBaseUrl } from "@/lib/auth";

export function createSupplierApi(): ApiClient {
  return new ApiClient({
    baseUrl: supplierApiBaseUrl(),
    getAuthToken: () => getSupplierToken() || null,
  });
}
