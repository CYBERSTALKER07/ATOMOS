import {
  dispatchBackpressureFromResponse,
  etagFromResponse,
  type ConditionalGet,
} from "../conditional-get";

export interface ApiClientConfig {
  baseUrl: string;
  getAuthToken?: () => string | null;
  fetchImpl?: typeof fetch;
}

export interface RequestOptions {
  body?: unknown;
  rawBody?: string;
  idempotencyKey?: string;
  requiresAuth?: boolean;
  headers?: HeadersInit;
  ifNoneMatch?: string;
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


export class ApiClientBase {
  protected readonly inFlightGets = new Map<string, Promise<unknown>>();

  constructor(public readonly config: ApiClientConfig) {}

  async request<TResponse>(
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options: RequestOptions = {},
  ): Promise<TResponse> {
    const resolvedUrl = this.resolveURL(path);
    const requiresAuth = options.requiresAuth ?? true;
    const tokenKey = requiresAuth && this.config.getAuthToken ? (this.config.getAuthToken() ?? "") : "";

    if (method === "GET") {
      const coalesceKey = `${tokenKey}|${resolvedUrl}|${options.ifNoneMatch ?? ""}`;
      const existing = this.inFlightGets.get(coalesceKey);
      if (existing) {
        return existing as Promise<TResponse>;
      }
      const task = this.executeRequest<TResponse>(resolvedUrl, method, options).finally(() => {
        this.inFlightGets.delete(coalesceKey);
      });
      this.inFlightGets.set(coalesceKey, task);
      return task;
    }

    this.inFlightGets.clear();
    return this.executeRequest<TResponse>(resolvedUrl, method, options);
  }

  private async executeRequest<TResponse>(
    resolvedUrl: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options: RequestOptions,
  ): Promise<TResponse> {
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const requiresAuth = options.requiresAuth ?? true;
    const headers = this.buildHeaders(options.headers, requiresAuth, options.idempotencyKey, options.rawBody !== undefined, method);

    const init: RequestInit = {
      method,
      headers,
      credentials: "include",
    };
    if (options.rawBody !== undefined) {
      init.body = options.rawBody;
    } else if (options.body !== undefined) {
      init.body = JSON.stringify(options.body);
    }

    const response = await fetchImpl(resolvedUrl, init);
    dispatchBackpressureFromResponse(response);
    const payload = await parseResponsePayload(response);
    if (!response.ok) {
      const message = extractErrorMessage(response.status, payload);
      throw new ApiError(message, response.status, payload);
    }

    return payload as TResponse;
  }

  protected async requestConditional<TResponse>(
    path: string,
    method: "GET",
    options: RequestOptions = {},
  ): Promise<ConditionalGet<TResponse>> {
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const requiresAuth = options.requiresAuth ?? true;
    const headers = this.buildHeaders(options.headers, requiresAuth, options.idempotencyKey, options.rawBody !== undefined);
    if (options.ifNoneMatch) {
      headers.set("If-None-Match", options.ifNoneMatch);
    }

    const response = await fetchImpl(this.resolveURL(path), {
      method,
      headers,
      credentials: "include",
    });
    dispatchBackpressureFromResponse(response);
    if (response.status === 304) {
      return { notModified: true, etag: etagFromResponse(response) ?? options.ifNoneMatch ?? null };
    }
    const payload = await parseResponsePayload(response);
    if (!response.ok) {
      throw new ApiError(extractErrorMessage(response.status, payload), response.status, payload);
    }
    return { notModified: false, data: payload as TResponse, etag: etagFromResponse(response) };
  }

  protected async requestText(
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    options: RequestOptions = {},
  ): Promise<string> {
    const fetchImpl = this.config.fetchImpl ?? fetch;
    const requiresAuth = options.requiresAuth ?? true;
    const headers = this.buildHeaders(options.headers, requiresAuth, options.idempotencyKey, options.rawBody !== undefined);
    const init: RequestInit = {
      method,
      headers,
      credentials: "include",
    };
    if (options.rawBody !== undefined) {
      init.body = options.rawBody;
    } else if (options.body !== undefined) {
      init.body = JSON.stringify(options.body);
    }
    const response = await fetchImpl(this.resolveURL(path), init);
    const text = await response.text();
    if (!response.ok) {
      let payload: unknown = text;
      try {
        payload = JSON.parse(text);
      } catch {
        /* keep text */
      }
      throw new ApiError(extractErrorMessage(response.status, payload), response.status, payload);
    }
    return text;
  }


  protected buildHeaders(
    extra: HeadersInit | undefined,
    requiresAuth: boolean,
    idempotencyKey: string | undefined,
    hasRawBody: boolean,
    method = "GET",
  ): Headers {
    const headers = new Headers(extra);
    if (!hasRawBody) {
      headers.set("Content-Type", "application/json");
    }
    if (!headers.has("X-Trace-Id") && typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
      headers.set("X-Trace-Id", crypto.randomUUID());
    }

    if (requiresAuth && this.config.getAuthToken) {
      const token = this.config.getAuthToken();
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }
    }
    if (idempotencyKey) {
      headers.set("Idempotency-Key", idempotencyKey);
    } else if (
      method === "POST" &&
      !headers.has("Idempotency-Key") &&
      typeof crypto !== "undefined" &&
      typeof crypto.randomUUID === "function"
    ) {
      headers.set("Idempotency-Key", crypto.randomUUID());
    }

    return headers;
  }

  protected resolveURL(path: string): string {
    if (/^https?:\/\//i.test(path)) {
      return path;
    }
    const base = this.config.baseUrl.endsWith("/") ? this.config.baseUrl : `${this.config.baseUrl}/`;
    const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
    return new URL(normalizedPath, base).toString();
  }
}

export async function parseResponsePayload(response: Response): Promise<unknown> {
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

export function extractErrorMessage(status: number, payload: unknown): string {
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

export function appendQuery(path: string, query: Record<string, unknown>): string {
  const entries = Object.entries(query).filter(([, value]) => value !== undefined && value !== null && `${value}`.trim() !== "");
  if (entries.length === 0) {
    return path;
  }
  const params = new URLSearchParams();
  for (const [key, value] of entries) {
    params.set(key, String(value));
  }
  return `${path}?${params.toString()}`;
}
