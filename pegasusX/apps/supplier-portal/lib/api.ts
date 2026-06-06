import { ApiClient } from "@pegasusx/api-client";
import { getSupplierToken, supplierApiBaseUrl } from "@/lib/auth";

export function createSupplierApi(): ApiClient {
  return new ApiClient({
    baseUrl: supplierApiBaseUrl(),
    getAuthToken: () => getSupplierToken() || null,
  });
}
