import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { notifyWarehouseSessionReconciled } from "../warehouse-reconnect";
import { useWarehouseSessionReconcile } from "../use-warehouse-session-reconcile";

describe("useWarehouseSessionReconcile", () => {
  it("runs callback when warehouse session reconcile event fires", () => {
    const onReconcile = vi.fn();
    renderHook(() => useWarehouseSessionReconcile(onReconcile));

    act(() => {
      notifyWarehouseSessionReconciled();
    });

    expect(onReconcile).toHaveBeenCalledTimes(1);
  });
});
