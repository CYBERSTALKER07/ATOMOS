import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import React from "react";
import { ORDER_STATUS_FUNNEL, emptyOrderStatusCounts, incrementOrderStatusCount } from "@pegasusx/types";
import { KpiStat, StatusStack, StatusStackPreview } from "@pegasusx/ui-kit/portal";

const liveCounts = {
  ...emptyOrderStatusCounts(),
  PENDING: 4,
  LOADED: 2,
  IN_TRANSIT: 3,
  COMPLETED: 1,
};

describe("GS-U1 StatusStack", () => {
  it("renders empty without inventing chips", () => {
    render(<StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={null} />);
    expect(screen.getByTestId("gs-u-status-stack")).toHaveAttribute("data-mode", "empty");
    expect(screen.getByTestId("gs-u-status-stack-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("gs-u-status-stack-chips")).not.toBeInTheDocument();
    expect(screen.queryByTestId("gs-u-status-stack-bar")).not.toBeInTheDocument();
  });

  it("renders zero as 17 chips of 0 and no bar", () => {
    render(<StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={emptyOrderStatusCounts()} />);
    expect(screen.getByTestId("gs-u-status-stack")).toHaveAttribute("data-mode", "zero");
    const chips = screen.getByTestId("gs-u-status-stack-chips").querySelectorAll("[data-status]");
    expect(chips).toHaveLength(17);
    expect(Array.from(chips).every((chip) => chip.textContent?.endsWith("0"))).toBe(true);
    expect(screen.queryByTestId("gs-u-status-stack-bar")).not.toBeInTheDocument();
  });

  it("renders unavailable as em-dash chips, not zeros", () => {
    render(
      <StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={liveCounts} available={false} />,
    );
    expect(screen.getByTestId("gs-u-status-stack")).toHaveAttribute("data-mode", "unavailable");
    expect(screen.getByTestId("gs-u-status-stack-unavailable")).toBeInTheDocument();
    const chips = screen.getByTestId("gs-u-status-stack-chips").querySelectorAll("[data-status]");
    expect(chips).toHaveLength(17);
    expect(Array.from(chips).every((chip) => chip.textContent?.includes("—"))).toBe(true);
    expect(screen.queryByTestId("gs-u-status-stack-bar")).not.toBeInTheDocument();
  });

  it("shows FISCAL_FAILED as 1 after an increment", () => {
    const counts = incrementOrderStatusCount(emptyOrderStatusCounts(), "FISCAL_FAILED");
    render(<StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={counts} source="live" />);
    const chip = screen.getByTestId("gs-u-status-stack-chips").querySelector('[data-status="FISCAL_FAILED"]');
    expect(chip?.textContent).toMatch(/1/);
    expect(screen.getByTestId("gs-u-status-stack-chips").querySelectorAll("[data-status]")).toHaveLength(17);
  });

  it("renders the live 17-key stack with a bar", () => {
    render(<StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={liveCounts} source="live" />);
    expect(screen.getByTestId("gs-u-status-stack")).toHaveAttribute("data-mode", "live");
    expect(screen.getByTestId("gs-u-status-stack-bar")).toBeInTheDocument();
    expect(screen.getByTestId("gs-u-status-stack-chips").querySelectorAll("[data-status]")).toHaveLength(17);
    expect(screen.getByTestId("gs-u-source-chip")).toHaveAttribute("data-source", "live");
  });

  it("previews empty, zero, unavailable, and live 17 together", () => {
    render(<StatusStackPreview />);
    const stacks = screen.getAllByTestId("gs-u-status-stack");
    expect(stacks.map((node) => node.getAttribute("data-mode"))).toEqual([
      "empty",
      "zero",
      "unavailable",
      "live",
    ]);
    expect(screen.getByTestId("gs-u-status-stack-preview")).toBeInTheDocument();
  });
});

describe("GS-U1 KpiStat spark guard", () => {
  it("does not draw a spark for unavailable history", () => {
    render(
      <KpiStat
        label="Completed"
        value="—"
        source="unavailable"
        spark={{ points: [1, 2, 3], source: "unavailable", available: false }}
      />,
    );
    expect(screen.queryByTestId("gs-u-kpi-spark")).not.toBeInTheDocument();
  });

  it("draws a spark only for a live series", () => {
    render(
      <KpiStat
        label="Pending"
        value="4"
        source="live"
        spark={{ points: [1, 3, 2], source: "live", available: true }}
      />,
    );
    expect(screen.getByTestId("gs-u-kpi-spark")).toBeInTheDocument();
  });
});
