/** HH:MM receiving window helpers — mirror backend proximity.ValidateReceivingWindow. */

export function normalizeReceivingWindow(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";
  const match = /^([01]?\d|2[0-3]):([0-5]?\d)$/.exec(trimmed);
  if (!match) return trimmed;
  return `${match[1].padStart(2, "0")}:${match[2].padStart(2, "0")}`;
}

export function validateReceivingWindowField(value: string): string | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(normalizeReceivingWindow(trimmed))) {
    return "Use 24-hour HH:MM format (e.g. 09:00).";
  }
  return undefined;
}
