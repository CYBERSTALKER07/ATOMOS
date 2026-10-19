export function orderStatusFromWsRaw(raw: unknown): string {
  if (typeof raw !== "string" || !raw.trim()) return "";
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const data =
      parsed.data && typeof parsed.data === "object"
        ? (parsed.data as Record<string, unknown>)
        : parsed;
    return String(data.status ?? data.order_status ?? parsed.status ?? "");
  } catch {
    return "";
  }
}
