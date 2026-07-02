import { describe, expect, it } from "vitest";

type SupplyRequest = { request_id: string; state: string };
type FilterState = "ALL" | "SUBMITTED" | "IN_PRODUCTION" | "READY";

function filterSupplyRequests(requests: SupplyRequest[], filter: FilterState): SupplyRequest[] {
  return filter === "ALL" ? requests : requests.filter((request) => request.state === filter);
}

describe("factory supply-requests filter", () => {
  const rows: SupplyRequest[] = [
    { request_id: "r1", state: "SUBMITTED" },
    { request_id: "r2", state: "READY" },
    { request_id: "r3", state: "SUBMITTED" },
  ];

  it("returns all rows for ALL", () => {
    expect(filterSupplyRequests(rows, "ALL")).toHaveLength(3);
  });

  it("filters by state", () => {
    expect(filterSupplyRequests(rows, "SUBMITTED").map((r) => r.request_id)).toEqual([
      "r1",
      "r3",
    ]);
  });
});
