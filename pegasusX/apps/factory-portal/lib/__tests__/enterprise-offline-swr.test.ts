import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useEnterpriseSWR, NetworkLifecycleEngine } from "../enterprise-swr";

describe("Factory Portal Enterprise Offline SWR & Lifecycle Engine", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    NetworkLifecycleEngine.reset();
  });

  afterEach(() => {
    NetworkLifecycleEngine.reset();
  });

  it("suppresses automated polling fetch when offline", async () => {
    NetworkLifecycleEngine.setOnline(false);

    const fetcher = vi.fn().mockResolvedValue({ id: 1, name: "Assembly Line A" });

    const { result } = renderHook(() =>
      useEnterpriseSWR("/v1/factory/lines", fetcher, { interval: 5000, initialData: { id: 1, name: "Cached Line" } })
    );

    expect(result.current.isOffline).toBe(true);
    expect(result.current.data).toEqual({ id: 1, name: "Cached Line" });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });

    expect(fetcher).not.toHaveBeenCalled();
  });

  it("preserves cached data non-destructively when background revalidation fails", async () => {
    NetworkLifecycleEngine.setOnline(true);

    const cached = { batches: [{ batchId: "B-101", status: "PROCESSING" }] };
    const failingFetcher = vi.fn().mockRejectedValue(new Error("503 Service Unavailable"));

    const { result } = renderHook(() =>
      useEnterpriseSWR("/v1/factory/batches", failingFetcher, { initialData: cached })
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

    const fetcherLines = vi.fn().mockResolvedValue({ lines: [] });
    const fetcherAlerts = vi.fn().mockResolvedValue({ alerts: [] });

    renderHook(() => useEnterpriseSWR("/v1/factory/lines", fetcherLines, { interval: 10000 }));
    renderHook(() => useEnterpriseSWR("/v1/factory/alerts", fetcherAlerts, { interval: 15000 }));

    expect(fetcherLines).not.toHaveBeenCalled();
    expect(fetcherAlerts).not.toHaveBeenCalled();

    await act(async () => {
      NetworkLifecycleEngine.setOnline(true);
      await NetworkLifecycleEngine.triggerSyncEpoch();
    });

    await waitFor(() => {
      expect(fetcherLines).toHaveBeenCalledTimes(1);
      expect(fetcherAlerts).toHaveBeenCalledTimes(1);
    });
  });
});
