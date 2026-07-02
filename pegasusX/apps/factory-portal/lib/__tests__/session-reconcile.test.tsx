import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { notifyFactorySessionReconciled } from "../factory-reconnect";
import { useFactorySessionReconcile } from "../use-factory-session-reconcile";

describe("useFactorySessionReconcile", () => {
  it("runs callback when factory session reconcile event fires", () => {
    const onReconcile = vi.fn();
    renderHook(() => useFactorySessionReconcile(onReconcile));

    act(() => {
      notifyFactorySessionReconciled();
    });

    expect(onReconcile).toHaveBeenCalledTimes(1);
  });
});
