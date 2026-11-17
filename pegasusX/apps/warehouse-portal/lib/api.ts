import { ApiClient } from '@pegasusx/api-core';
import { readTokenFromCookie, warehouseApiBaseUrl } from "@/lib/auth";

export function createWarehouseApi(): ApiClient {
  return new ApiClient({
    baseUrl: warehouseApiBaseUrl(),
    getAuthToken: () => readTokenFromCookie() || null,
  });
}
