import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useEnterpriseSWR, NetworkLifecycleEngine } from "../enterprise-swr";

describe("Warehouse Portal Enterprise Offline SWR & Lifecycle Engine", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    NetworkLifecycleEngine.reset();
  });

  afterEach(() => {
    NetworkLifecycleEngine.reset();
  });

  it("suppresses automated polling fetch when offline", async () => {
    NetworkLifecycleEngine.setOnline(false);

    const fetcher = vi.fn().mockResolvedValue({ id: 1, bay: "Bay 4" });

    const { result } = renderHook(() =>
      useEnterpriseSWR("/v1/warehouse/bays", fetcher, { interval: 5000, initialData: { id: 1, bay: "Cached Bay" } })
    );

    expect(result.current.isOffline).toBe(true);
    expect(result.current.data).toEqual({ id: 1, bay: "Cached Bay" });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(fetcher).not.toHaveBeenCalled();
  });

  it("preserves cached data non-destructively when background revalidation fails", async () => {
    NetworkLifecycleEngine.setOnline(true);

    const cached = { pallets: [{ id: "PAL-88", status: "STAGED" }] };
    const failingFetcher = vi.fn().mockRejectedValue(new Error("503 Service Unavailable"));

    const { result } = renderHook(() =>
      useEnterpriseSWR("/v1/warehouse/pallets", failingFetcher, { initialData: cached })
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

    const fetcherBays = vi.fn().mockResolvedValue({ bays: [] });
    const fetcherTasks = vi.fn().mockResolvedValue({ tasks: [] });

    renderHook(() => useEnterpriseSWR("/v1/warehouse/bays", fetcherBays, { interval: 10000 }));
    renderHook(() => useEnterpriseSWR("/v1/warehouse/tasks", fetcherTasks, { interval: 15000 }));

    expect(fetcherBays).not.toHaveBeenCalled();
    expect(fetcherTasks).not.toHaveBeenCalled();

    await act(async () => {
      NetworkLifecycleEngine.setOnline(true);
      await NetworkLifecycleEngine.triggerSyncEpoch();
    });

    await waitFor(() => {
      expect(fetcherBays).toHaveBeenCalledTimes(1);
      expect(fetcherTasks).toHaveBeenCalledTimes(1);
    });
  });
});
