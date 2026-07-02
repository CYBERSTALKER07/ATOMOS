import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useRetailerSessionReconcile } from "../use-retailer-session-reconcile";

describe("useRetailerSessionReconcile", () => {
  it("runs callback on retailer:session-reconciled", () => {
    const onReconcile = vi.fn();
    renderHook(() => useRetailerSessionReconcile(onReconcile));

    act(() => {
      window.dispatchEvent(new Event("retailer:session-reconciled"));
    });

    expect(onReconcile).toHaveBeenCalledTimes(1);
  });
});
