import { ApiClient } from "@pegasusx/api-core";
import { readTokenFromCookie, factoryApiBaseUrl } from "@/lib/auth";

export function createFactoryApi(): ApiClient {
  return new ApiClient({
    baseUrl: factoryApiBaseUrl(),
    getAuthToken: () => readTokenFromCookie() || null,
  });
}
