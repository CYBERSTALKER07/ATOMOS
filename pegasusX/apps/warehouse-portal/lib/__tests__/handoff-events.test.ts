import { describe, expect, it } from "vitest";
import type { PulseEvent } from "@pegasusx/types";

const HANDOFF_KINDS = new Set([
  "PREORDER",
  "ORDER_ACCEPTED",
  "ORDER_DISPATCHED",
  "MANIFEST_SEALED",
  "MANIFEST_DISPATCHED",
  "DISPATCH",
]);

function isHandoffEvent(event: PulseEvent): boolean {
  const haystack = `${event.kind} ${event.title}`.toUpperCase();
  if (HANDOFF_KINDS.has(event.kind.toUpperCase())) return true;
  return /PREORDER|ACCEPT|DISPATCH|SEAL|MANIFEST/.test(haystack);
}

describe("warehouse handoff pulse filter", () => {
  it("keeps manifest seal events", () => {
    expect(
      isHandoffEvent({
        id: "evt-1",
        kind: "MANIFEST_SEALED",
        title: "Manifest sealed",
        occurred_at: new Date().toISOString(),
      } as PulseEvent),
    ).toBe(true);
  });

  it("drops unrelated pulse kinds", () => {
    expect(
      isHandoffEvent({
        id: "evt-2",
        kind: "INVENTORY_ADJUSTED",
        title: "Stock change",
        occurred_at: new Date().toISOString(),
      } as PulseEvent),
    ).toBe(false);
  });
});
