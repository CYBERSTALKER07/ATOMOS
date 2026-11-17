import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useEnterpriseSWR, NetworkLifecycleEngine } from "../enterprise-swr";

describe("Supplier Portal Enterprise Offline SWR & Lifecycle Engine", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    NetworkLifecycleEngine.reset();
  });

  afterEach(() => {
    NetworkLifecycleEngine.reset();
  });

  it("suppresses automated polling fetch when offline", async () => {
    NetworkLifecycleEngine.setOnline(false);

    const fetcher = vi.fn().mockResolvedValue({ id: 1, title: "Supplier Goods" });

    const { result } = renderHook(() =>
      useEnterpriseSWR("/v1/supplier/products", fetcher, { interval: 5000, initialData: { id: 1, title: "Cached Goods" } })
    );

    expect(result.current.isOffline).toBe(true);
    expect(result.current.data).toEqual({ id: 1, title: "Cached Goods" });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(fetcher).not.toHaveBeenCalled();
  });

  it("preserves cached data non-destructively when background revalidation fails", async () => {
    NetworkLifecycleEngine.setOnline(true);

    const cached = { items: [{ sku: "SKU-99", qty: 450 }] };
    const failingFetcher = vi.fn().mockRejectedValue(new Error("503 Service Unavailable"));

    const { result } = renderHook(() =>
      useEnterpriseSWR("/v1/supplier/inventory", failingFetcher, { initialData: cached })
    );

    expect(result.current.data).toEqual(cached);

    await act(async () => {
      await result.current.revalidate();
    });

    expect(result.current.data).toEqual(cached);
    expect(result.current.isRevalidating).toBe(false);
  });

  it("coordinates single epoch refresh across multiple hook consumers without cascading timers", async () => {
    NetworkLifecycleEngine.setOnline(false);

    const fetcherOrders = vi.fn().mockResolvedValue({ orders: [1] });
    const fetcherLedger = vi.fn().mockResolvedValue({ balance: 5000 });

    renderHook(() => useEnterpriseSWR("/v1/supplier/orders", fetcherOrders, { interval: 10000 }));
    renderHook(() => useEnterpriseSWR("/v1/supplier/ledger", fetcherLedger, { interval: 15000 }));

    expect(fetcherOrders).not.toHaveBeenCalled();
    expect(fetcherLedger).not.toHaveBeenCalled();

    await act(async () => {
      NetworkLifecycleEngine.setOnline(true);
      await NetworkLifecycleEngine.triggerSyncEpoch();
    });

    await waitFor(() => {
      expect(fetcherOrders).toHaveBeenCalledTimes(1);
      expect(fetcherLedger).toHaveBeenCalledTimes(1);
    });
  });
});
