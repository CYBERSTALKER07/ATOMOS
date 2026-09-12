import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  FACTORY_TRANSFER_STATES,
  MANIFEST_STATES,
  canonicalizeFactoryVehicle,
  emptyFactoryTransferCounts,
} from "@pegasusx/types";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/page.tsx"), "utf8");

describe("GS-U5 factory command", () => {
  it("labels the factory-truck plane and does not bind last-mile boards", () => {
    expect(pageSource).toMatch(/Factory trucks/);
    expect(pageSource).toMatch(/gs-u-factory-trucks/);
    expect(pageSource).toMatch(/FACTORY_TRANSFER_STATES/);
    expect(pageSource).toMatch(/MANIFEST_STATES/);
    expect(pageSource).toMatch(/gs-u-factory-source/);
    expect(pageSource).toMatch(/\/transfers\?state=/);
    expect(pageSource).not.toMatch(/ORDER_STATUS_FUNNEL/);
    expect(pageSource).not.toMatch(/TRUCK_DUTY_STATUSES/);
    expect(pageSource).toMatch(/Last-mile retailer IN_TRANSIT is not a factory truck/);
  });

  it("keeps the live transfer dictionary including zeros", () => {
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
