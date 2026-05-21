import type {
  SupplierBillingSetupRequest,
  SupplierBillingSetupResponse,
  SupplierConfigureRequest,
  SupplierConfigureResponse,
  SupplierLoginRequest,
  SupplierLoginResponse,
  SupplierProfile,
  SupplierProfileUpdateRequest,
  SupplierRegisterRequest,
  SupplierRegisterResponse,
  SupplierTopologyResponse,
  SupplierTopologyUpdateRequest,
} from "@pegasusx/types";

export interface ApiClientConfig {
  baseUrl: string;
  getAuthToken?: () => string | null;
  fetchImpl?: typeof fetch;
}

interface RequestOptions {
  body?: unknown;
  idempotencyKey?: string;
  requiresAuth?: boolean;
  headers?: HeadersInit;
}

export class ApiError extends Error {
  public readonly status: number;
  public readonly payload: unknown;

  constructor(message: string, status: number, payload: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export class ApiClient {
  constructor(public readonly config: ApiClientConfig) {}

  async registerSupplier(request: SupplierRegisterRequest, idempotencyKey: string): Promise<SupplierRegisterResponse> {
    return this.request<SupplierRegisterResponse>("/v1/auth/supplier/register", "POST", {
      body: request,
      idempotencyKey,
      requiresAuth: false,
    });
  }

  async loginSupplier(request: SupplierLoginRequest): Promise<SupplierLoginResponse> {
    return this.request<SupplierLoginResponse>("/v1/auth/supplier/login", "POST", {
      body: request,
      requiresAuth: false,
    });
  }

  async configureSupplier(request: SupplierConfigureRequest): Promise<SupplierConfigureResponse> {
    return this.request<SupplierConfigureResponse>("/v1/supplier/configure", "POST", {
      body: request,
    });
  }

  async configureSupplierBilling(
    request: SupplierBillingSetupRequest,
    idempotencyKey: string,
  ): Promise<SupplierBillingSetupResponse> {
    return this.request<SupplierBillingSetupResponse>("/v1/supplier/billing/setup", "POST", {
      body: request,
      idempotencyKey,
    });
  }

  async getSupplierProfile(): Promise<SupplierProfile> {
    return this.request<SupplierProfile>("/v1/supplier/profile", "GET");
  }

  async updateSupplierProfile(request: SupplierProfileUpdateRequest): Promise<SupplierProfile> {
    return this.request<SupplierProfile>("/v1/supplier/profile", "PUT", { body: request });
  }

  async getSupplierTopology(): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "GET");
  }

  async updateSupplierTopology(request: SupplierTopologyUpdateRequest): Promise<SupplierTopologyResponse> {
    return this.request<SupplierTopologyResponse>("/v1/supplier/topology", "PUT", { body: request });
  }

  private async request<TResponse>(
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options: RequestOptions = {},
  ): Promise<TResponse> {
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const requiresAuth = options.requiresAuth ?? true;
    const headers = this.buildHeaders(options.headers, requiresAuth, options.idempotencyKey);

    const init: RequestInit = {
      method,
      headers,
      credentials: "include",
    };
    if (options.body !== undefined) {
      init.body = JSON.stringify(options.body);
    }

    const response = await fetchImpl(this.resolveURL(path), init);
    const payload = await parseResponsePayload(response);
    if (!response.ok) {
      const message = extractErrorMessage(response.status, payload);
      throw new ApiError(message, response.status, payload);
    }

    return payload as TResponse;
  }

  private buildHeaders(extra: HeadersInit | undefined, requiresAuth: boolean, idempotencyKey: string | undefined): Headers {
    const headers = new Headers(extra);
    headers.set("Content-Type", "application/json");

    if (requiresAuth && this.config.getAuthToken) {
      const token = this.config.getAuthToken();
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }
    }
    if (idempotencyKey) {
      headers.set("Idempotency-Key", idempotencyKey);
    }

    return headers;
  }

  private resolveURL(path: string): string {
    if (/^https?:\/\//i.test(path)) {
      return path;
    }
    const base = this.config.baseUrl.endsWith("/") ? this.config.baseUrl : `${this.config.baseUrl}/`;
    const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
    return new URL(normalizedPath, base).toString();
  }
}

async function parseResponsePayload(response: Response): Promise<unknown> {
  if (response.status === 204) {
    return undefined;
  }

  const contentType = response.headers.get("content-type") || "";
  if (contentType.toLowerCase().includes("application/json")) {
    try {
      return await response.json();
    } catch {
      return undefined;
    }
  }

  try {
    const text = await response.text();
    return text.length > 0 ? text : undefined;
  } catch {
    return undefined;
  }
}

function extractErrorMessage(status: number, payload: unknown): string {
  if (payload && typeof payload === "object" && "error" in payload) {
    const candidate = (payload as { error?: unknown }).error;
    if (typeof candidate === "string" && candidate.length > 0) {
      return candidate;
    }
  }
  if (typeof payload === "string" && payload.length > 0) {
    return payload;
  }
  return `request failed (${status})`;
}
