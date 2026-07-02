import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { notifySupplierSessionReconciled } from "../supplier-reconnect";
import { useSupplierSessionReconcile } from "../use-supplier-session-reconcile";

describe("useSupplierSessionReconcile", () => {
  it("runs callback when supplier session reconcile event fires", () => {
    const onReconcile = vi.fn();
    renderHook(() => useSupplierSessionReconcile(onReconcile));

    act(() => {
      notifySupplierSessionReconciled();
    });

    expect(onReconcile).toHaveBeenCalledTimes(1);
  });
});
