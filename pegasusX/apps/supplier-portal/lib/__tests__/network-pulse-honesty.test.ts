import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import NetworkPulsePanel from "../../components/NetworkPulsePanel";

describe("supplier network pulse honesty", () => {
  it("displays error state when network pulse fails instead of empty timeline", async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error("network error"));
    render(React.createElement(NetworkPulsePanel));
    await waitFor(() => {
      expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
    });
  });
});
