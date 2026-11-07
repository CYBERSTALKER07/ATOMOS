import { describe, expect, it } from "vitest";
import {
  FACTORY_TRANSFER_STATES,
  MANIFEST_STATES,
  canonicalizeFactoryVehicle,
  emptyFactoryTransferCounts,
} from "@pegasusx/types";

describe("GS-U5 factory command", () => {
  it("keeps the live transfer dictionary including zeros and canonical states", () => {
    expect(FACTORY_TRANSFER_STATES).toEqual(
      expect.arrayContaining(["CREATED", "APPROVED", "PENDING", "ASSIGNED", "LOADING", "DISPATCHED", "CANCELLED"]),
    );
    expect(MANIFEST_STATES).toEqual(
      expect.arrayContaining(["DRAFT", "LOADING", "SEALED", "DISPATCHED", "COMPLETED"]),
    );
    const empty = emptyFactoryTransferCounts();
    expect(Object.keys(empty)).toHaveLength(FACTORY_TRANSFER_STATES.length);
    expect(empty.PENDING).toBe(0);
    expect(canonicalizeFactoryVehicle("MAINTENANCE")).toBe("UNAVAILABLE");
  });
});
