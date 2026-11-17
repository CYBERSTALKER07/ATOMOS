import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { useSupplierWsRefresh } from "../use-supplier-ws-refresh";

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

  emitMessage(data: string, eventType?: string) {
    if (eventType && this.listeners.has(eventType)) {
      const event = new MessageEvent(eventType, { data });
      this.listeners.get(eventType)?.forEach(l => l(event));
    }
    if (this.onmessage) {
      const event = new MessageEvent("message", { data });
      this.onmessage(event);
    }
  }

  emitOpen() {
    if (this.onopen) {
      this.onopen(new Event("open"));
    }
  }
}

describe("useSupplierWsRefresh (SSE migration)", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (globalThis as any).EventSource = MockEventSource;
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("connects to /v1/supplier/events with credentials", () => {
    const onSignal = vi.fn();
    const eventTypes = new Set(["ORDER_STATUS_CHANGED"]);

    renderHook(() =>
      useSupplierWsRefresh(onSignal, {
        eventTypes,
        debounceMs: 100,
        enabled: true,
      })
    );

    expect(MockEventSource.instances.length).toBe(1);
    const es = MockEventSource.instances[0];
    expect(es.url).toContain("/v1/supplier/events");
    expect(es.options?.withCredentials).toBe(true);
  });

  it("dispatches matching event signals with debouncing", async () => {
    const onSignal = vi.fn();
    const eventTypes = new Set(["ORDER_STATUS_CHANGED"]);

    renderHook(() =>
      useSupplierWsRefresh(onSignal, {
        eventTypes,
        debounceMs: 100,
        enabled: true,
      })
    );

    const es = MockEventSource.instances[0];
    act(() => {
      es.emitOpen();
      es.emitMessage(JSON.stringify({ type: "ORDER_STATUS_CHANGED", id: "ord-1" }), "ORDER_STATUS_CHANGED");
    });

    expect(onSignal).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(150);
    });

    expect(onSignal).toHaveBeenCalledTimes(1);
    expect(onSignal).toHaveBeenCalledWith("ORDER_STATUS_CHANGED", JSON.stringify({ type: "ORDER_STATUS_CHANGED", id: "ord-1" }));
  });

  it("ignores non-matching event types", () => {
    const onSignal = vi.fn();
    const eventTypes = new Set(["ORDER_STATUS_CHANGED"]);

    renderHook(() =>
      useSupplierWsRefresh(onSignal, {
        eventTypes,
        debounceMs: 50,
        enabled: true,
      })
    );

    const es = MockEventSource.instances[0];
    act(() => {
      es.emitMessage(JSON.stringify({ type: "UNMATCHED_EVENT" }), "UNMATCHED_EVENT");
      vi.advanceTimersByTime(100);
    });

    expect(onSignal).not.toHaveBeenCalled();
  });

  it("closes EventSource stream on unmount", () => {
    const onSignal = vi.fn();
    const eventTypes = new Set(["ORDER_STATUS_CHANGED"]);

    const { unmount } = renderHook(() =>
      useSupplierWsRefresh(onSignal, {
        eventTypes,
        debounceMs: 50,
        enabled: true,
      })
    );

    const es = MockEventSource.instances[0];
    expect(es.closed).toBe(false);

    unmount();
    expect(es.closed).toBe(true);
  });
});
