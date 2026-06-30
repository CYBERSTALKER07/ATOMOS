export type MockPulseEvent = {
  id: string;
  kind: string;
  title: string;
  description?: string;
  occurred_at: string;
  order_id?: string;
  manifest_id?: string;
};

/** Fixed timestamps — avoid Date.now() so SSR and client hydration match. */
export const MOCK_PULSE_EVENTS: MockPulseEvent[] = [
  {
    id: "evt-1",
    kind: "ORDER_VETTED",
    title: "Order #4821 vetted",
    description: "Supplier approved retailer pre-order for dispatch eligibility.",
    occurred_at: "2026-06-30T14:42:00.000Z",
    order_id: "ord-4821",
  },
  {
    id: "evt-2",
    kind: "MANIFEST_SEALED",
    title: "Manifest M-109 sealed",
    description: "Payload gate confirmed seal — driver route activated.",
    occurred_at: "2026-06-30T14:09:00.000Z",
    manifest_id: "m-109",
  },
  {
    id: "evt-3",
    kind: "ROUTE_DEVIATION",
    title: "Route deviation flagged",
    description: "Driver telemetry exceeded planned corridor — ops notified.",
    occurred_at: "2026-06-30T13:24:00.000Z",
    order_id: "ord-4810",
  },
  {
    id: "evt-4",
    kind: "DISPATCH_AUTO",
    title: "Smart dispatch batch committed",
    description: "12 orders clustered into 3 H3 cells across 2 trucks.",
    occurred_at: "2026-06-30T12:54:00.000Z",
  },
];
