import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useEnterpriseSWR, NetworkLifecycleEngine } from "../enterprise-swr";

describe("Enterprise Offline SWR & Lifecycle Engine", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    NetworkLifecycleEngine.reset();
  });

  afterEach(() => {
    NetworkLifecycleEngine.reset();
  });

  it("suppresses automated polling fetch when offline", async () => {
    // Simulate offline state
    NetworkLifecycleEngine.setOnline(false);

    const fetcher = vi.fn().mockResolvedValue({ id: 1, title: "Test Item" });

    const { result } = renderHook(() =>
      useEnterpriseSWR("/api/v1/test", fetcher, { interval: 5000, initialData: { id: 1, title: "Cached" } })
    );

    expect(result.current.isOffline).toBe(true);
    expect(result.current.data).toEqual({ id: 1, title: "Cached" });

    // Advance time by 15 seconds
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    // Fetcher must NOT have been called while offline
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("preserves cached data non-destructively when background revalidation fails", async () => {
    NetworkLifecycleEngine.setOnline(true);

    const cached = { orders: [{ id: "ord-1", total: 100 }] };
    const failingFetcher = vi.fn().mockRejectedValue(new Error("503 Gateway Timeout"));

    const { result } = renderHook(() =>
      useEnterpriseSWR("/api/v1/orders", failingFetcher, { initialData: cached })
    );

    // Initial state holds cached data
    expect(result.current.data).toEqual(cached);

    // Trigger revalidation
    await act(async () => {
      await result.current.revalidate();
    });

    // Cached data must NEVER be wiped; isRevalidating must return to false
    expect(result.current.data).toEqual(cached);
    expect(result.current.isRevalidating).toBe(false);
  });

  it("coordinates single epoch refresh across multiple hook consumers without cascading timers", async () => {
    NetworkLifecycleEngine.setOnline(false);

    const fetcherA = vi.fn().mockResolvedValue({ a: 1 });
    const fetcherB = vi.fn().mockResolvedValue({ b: 2 });

    const { result: hookA } = renderHook(() =>
      useEnterpriseSWR("/api/v1/a", fetcherA, { interval: 10000 })
    );
    const { result: hookB } = renderHook(() =>
      useEnterpriseSWR("/api/v1/b", fetcherB, { interval: 15000 })
    );

    expect(fetcherA).not.toHaveBeenCalled();
    expect(fetcherB).not.toHaveBeenCalled();

    // Reconnection occurs -> single coordinated epoch trigger
    await act(async () => {
      NetworkLifecycleEngine.setOnline(true);
      await NetworkLifecycleEngine.triggerSyncEpoch();
    });

    await waitFor(() => {
      expect(fetcherA).toHaveBeenCalledTimes(1);
      expect(fetcherB).toHaveBeenCalledTimes(1);
    });
  });
});
