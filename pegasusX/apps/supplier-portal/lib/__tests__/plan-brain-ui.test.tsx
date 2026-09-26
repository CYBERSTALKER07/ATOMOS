import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import React from "react";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";

describe("GS-U3 Brain honesty UI", () => {
  it("renders sparsity blocked_reason and no invented band", () => {
    render(
      <ForecastConfidenceCard
        confidence={{ label: "insufficient_history", blocked_reason: "sparsity_blocked" }}
      />,
    );
    expect(screen.getByTestId("gs-u-forecast-blocked-reason")).toHaveTextContent("sparsity_blocked");
    expect(screen.queryByText(/units/)).not.toBeInTheDocument();
  });
});
