import re

with open("pegasus/apps/admin-portal/lib/auth.ts", "r") as f:
    content = f.read()

gate_code = """
/**
 * Global RequestGate blocks outbound requests during cooldown/jail 
 * and applies backpressure to non-critical polling or reads.
 */
class RequestGateManager {
  private jailUntil: number = 0;

  constructor() {
    if (typeof window \!== 'undefined') {
      window.addEventListener('cooldown', (ev: Event) => {
        const detail = (ev as CustomEvent).detail;
        if (detail?.jailUntil) {
          this.jailUntil = Math.max(this.jailUntil, detail.jailUntil);
        }
      });
    }
  }

  async checkGate(isMutable: boolean, isCritical: boolean = false): Promise<void> {
    const now = Math.floor(Date.now() / 1000);
    if (this.jailUntil > now && \!isCritical) {
      const msg = `[RequestGate] Outbound request dropped. Jailed for ${this.jailUntil - now}s`;
      console.warn(msg);
      // Throw a synthetic error to halt execution without hitting the network
      throw new Error(msg);
    }
  }
}
const RequestGate = new RequestGateManager();
"""

mut_methods_re = re.compile(r"(const MUTABLE_METHODS = new Set\(\['POST', 'PUT', 'PATCH', 'DELETE'\]\);)")
content = mut_methods_re.sub(r"\1\n" + gate_code, content)

perform_fetch_re = re.compile(r"async function performApiFetch\(path: string, init\?: RequestInit, options\?: ApiFetchOptions\): Promise<Response> \{")
new_perform_fetch = """
type ApiFetchOptions = {
  queueMutableOnNetworkError?: boolean;
  isCritical?: boolean;
};

async function performApiFetch(path: string, init?: RequestInit, options?: ApiFetchOptions): Promise<Response> {"""

content = re.sub(r"type ApiFetchOptions = \{\n  queueMutableOnNetworkError\?: boolean;\n\};", "", content)
content = perform_fetch_re.sub(new_perform_fetch, content)


headers_re = re.compile(r"(const headers: Record<string, string> = \{[^\}]+?\};)", re.MULTILINE)
new_headers = """const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Trace-Id': crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  const method = (init?.method || 'GET').toUpperCase();
  const isMutable = MUTABLE_METHODS.has(method);

  // Centralized Idempotency for mutating methods
  if (isMutable && \!headers['Idempotency-Key']) {
    headers['Idempotency-Key'] = crypto.randomUUID();
  }

  // RequestGate pre-flight check
  await RequestGate.checkGate(isMutable, options?.isCritical);
"""
content = headers_re.sub(new_headers, content)

with open("pegasus/apps/admin-portal/lib/auth.ts", "w") as f:
    f.write(content)

