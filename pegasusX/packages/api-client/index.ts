// pegasusX REST + WebSocket client surface. Populate alongside backend routes.

export interface ApiClientConfig {
  baseUrl: string;
  getAuthToken?: () => string | null;
}

export class ApiClient {
  constructor(public readonly config: ApiClientConfig) {}
}
