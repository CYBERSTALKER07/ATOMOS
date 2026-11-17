import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { useNotifications, SUPPLIER_SSE_EVENT_TYPES } from "../useNotifications";

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  options?: EventSourceInit;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  listeners: Map<string, ((event: MessageEvent) => void)[]> = new Map();
  closed = false;

  constructor(url: string, options?: EventSourceInit) {
    this.url = url;
    this.options = options;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    const list = this.listeners.get(type) || [];
    list.push(listener);
    this.listeners.set(type, list);
  }

  removeEventListener(type: string, listener: (event: MessageEvent) => void) {
    const list = this.listeners.get(type) || [];
    this.listeners.set(type, list.filter(l => l !== listener));
  }

  close() {
    this.closed = true;
  }

  emitNamedEvent(eventType: string, data: string, lastEventId = "") {
    const event = new MessageEvent(eventType, { data, lastEventId });
    this.listeners.get(eventType)?.forEach(l => l(event));
  }

  emitMessage(data: string, lastEventId = "") {
    if (this.onmessage) {
      const event = new MessageEvent("message", { data, lastEventId });
      this.onmessage(event);
    }
  }

  emitOpen() {
    if (this.onopen) {
      this.onopen(new Event("open"));
    }
  }
}

vi.mock("../auth", () => ({
  readTokenFromCookie: () => "mock-jwt-token",
  resolveSupplierToken: async () => "mock-jwt-token",
  supplierFetch: vi.fn(async (url: string) => {
    if (url.includes("/v1/user/notifications?")) {
      return {
        ok: true,
        json: async () => ({
          notifications: [],
          unread_count: 0,
          has_more: false,
        }),
      };
    }
    if (url.includes("/v1/supplier/ws-session")) {
      return {
        ok: true,
        json: async () => ({
          token: "mock-ws-ticket",
          expires_at: "2099-01-01T00:00:00Z",
        }),
      };
    }
    return { ok: true, json: async () => ({}) };
  }),
}));

vi.mock("../session-reconcile", () => ({
  runSupplierSessionReconcile: vi.fn(),
}));

describe("useNotifications Hook & Named SSE Handling", () => {
  const originalEventSource = globalThis.EventSource;

  beforeEach(() => {
    MockEventSource.instances = [];
    // @ts-expect-error test mock
    globalThis.EventSource = MockEventSource;
  });

  afterEach(() => {
    globalThis.EventSource = originalEventSource;
    vi.restoreAllMocks();
  });

  it("exports canonical SUPPLIER_SSE_EVENT_TYPES", () => {
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("notification");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("ORDER_CREATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("ORDER_UPDATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("ORDER_STATUS_CHANGED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("ORDER_CANCELLED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("ORDER_DELIVERED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("CLAIM_CREATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("CLAIM_UPDATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("PAYMENT_CLEARED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("PAYMENT_FAILED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("INVENTORY_UPDATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("LOW_STOCK_ALERT");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("EXCEPTION_CREATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("EXCEPTION_RESOLVED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("PLAYBOOK_RUN_UPDATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("FISCAL_RECEIPT_ISSUED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("FISCAL_RECEIPT_FAILED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("SUPPLY_REQUEST_UPDATED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("CASHIER_SESSION_STARTED");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("CASHIER_SESSION_TRANSITION");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("HANDOFF_CARD");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("order.created");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("order.updated");
    expect(SUPPLIER_SSE_EVENT_TYPES).toContain("order.status_changed");
  });

  it("subscribes to both onmessage and named event listeners on EventSource", async () => {
    const { result } = renderHook(() => useNotifications());
    await waitFor(() => expect(MockEventSource.instances.length).toBeGreaterThan(0));

    const es = MockEventSource.instances[0];
    expect(es.url).toContain("token=mock-ws-ticket");
    expect(es.url).not.toContain("mock-jwt-token");
    expect(es.onmessage).toBeTypeOf("function");
    expect(es.listeners.has("ORDER_CREATED")).toBe(true);
    expect(es.listeners.has("CLAIM_CREATED")).toBe(true);
    expect(es.listeners.has("notification")).toBe(true);
  });

  it("processes unnamed messages received via onmessage", async () => {
    const { result } = renderHook(() => useNotifications());
    await waitFor(() => expect(MockEventSource.instances.length).toBeGreaterThan(0));

    const es = MockEventSource.instances[0];
    act(() => {
      es.emitMessage(
        JSON.stringify({
          id: "notif-1",
          type: "notification",
          title: "System Update",
          body: "Maintenance complete",
        }),
        "notif-1",
      );
    });

    expect(result.current.items).toHaveLength(1);
    expect(result.current.items[0].id).toBe("notif-1");
    expect(result.current.items[0].title).toBe("System Update");
    expect(result.current.unreadCount).toBe(1);
  });

  it("processes named SSE event frames via addEventListener", async () => {
    const { result } = renderHook(() => useNotifications());
    await waitFor(() => expect(MockEventSource.instances.length).toBeGreaterThan(0));

    const es = MockEventSource.instances[0];
    act(() => {
      es.emitNamedEvent(
        "ORDER_CREATED",
        JSON.stringify({
          id: "notif-2",
          type: "ORDER_CREATED",
          title: "New Order",
          body: "Order #552 placed",
        }),
        "notif-2",
      );
    });

    expect(result.current.items).toHaveLength(1);
    expect(result.current.items[0].id).toBe("notif-2");
    expect(result.current.items[0].type).toBe("ORDER_CREATED");
  });

  it("deduplicates frames when both onmessage and addEventListener fire with the same id", async () => {
    const { result } = renderHook(() => useNotifications());
    await waitFor(() => expect(MockEventSource.instances.length).toBeGreaterThan(0));

    const es = MockEventSource.instances[0];
    const payload = JSON.stringify({
      id: "notif-dup",
      type: "ORDER_UPDATED",
      title: "Order Updated",
      body: "Status changed to confirmed",
    });

    act(() => {
      // Simulate named event listener firing
      es.emitNamedEvent("ORDER_UPDATED", payload, "notif-dup");
      // And onmessage firing for the same frame
      es.emitMessage(payload, "notif-dup");
    });

    expect(result.current.items).toHaveLength(1);
    expect(result.current.unreadCount).toBe(1);
  });
});
