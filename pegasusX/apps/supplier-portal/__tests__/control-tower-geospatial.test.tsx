import { describe, expect, it, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import ControlTowerCommandPanel from "../components/ControlTowerCommandPanel";
import * as authModule from "../lib/auth";
import * as apiCoreModule from "@pegasusx/api-core";
import * as validationModule from "@pegasusx/validation";

// Mock i18n
vi.mock("@/lib/i18n", () => ({
  usePortalT: () => (key: string) => key,
}));

// Mock FleetLiveMapPanel to simplify DOM rendering
vi.mock("@/components/FleetLiveMapPanel", () => ({
  default: () => <div data-testid="fleet-live-map" />,
}));

describe("ControlTowerCommandPanel Geospatial Validation Integration", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("validates and publishes override when polygon is valid", async () => {
    const mockFetch = vi.spyOn(authModule, "supplierFetch").mockResolvedValue({
      ok: true,
      json: async () => ({ override_id: "ovr-123", action: "REROUTE" }),
    } as Response);

    vi.spyOn(apiCoreModule, "sessionMapCenter").mockReturnValue({
      lat: 37.7749,
      lng: -122.4194,
    });

    render(<ControlTowerCommandPanel />);

    const publishButton = screen.getByRole("button", { name: /Publish zone override/i });
    fireEvent.click(publishButton);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const fetchCall = mockFetch.mock.calls[0];
    expect(fetchCall[0]).toBe("/v1/supplier/control-tower/zone-overrides");
    const body = JSON.parse(fetchCall[1]?.body as string);
    expect(body.action).toBe("REROUTE");
    expect(body.polygon_geojson).toBeDefined();
    expect(body.polygon_geojson.type).toBe("Polygon");
    expect(body.polygon_geojson.coordinates[0].length).toBe(5);

    expect(screen.getByText(/Override ovr-123 active/i)).toBeInTheDocument();
  });

  it("intercepts polygon submission and halts with Validation Error when geometry has kinks or is invalid", async () => {
    const mockFetch = vi.spyOn(authModule, "supplierFetch");
    vi.spyOn(apiCoreModule, "sessionMapCenter").mockReturnValue({
      lat: 37.7749,
      lng: -122.4194,
    });

    // Mock validation to simulate self-intersecting polygon detection
    vi.spyOn(validationModule, "validateAndSimplifyPolygon").mockReturnValueOnce({
      valid: false,
      hasKinks: true,
      error: "Self-intersecting polygon detected (kinks). Polygons must not cross themselves.",
      vertexCount: 5,
      simplified: false,
    });

    render(<ControlTowerCommandPanel />);

    const publishButton = screen.getByRole("button", { name: /Publish zone override/i });
    fireEvent.click(publishButton);

    await waitFor(() => {
      expect(
        screen.getByText(/Validation Error: Self-intersecting polygon detected \(kinks\)/i)
      ).toBeInTheDocument();
    });

    // supplierFetch must NOT have been called due to client-side validation abort
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("passes auto-simplified polygon to supplierFetch when vertex count > 50", async () => {
    const mockFetch = vi.spyOn(authModule, "supplierFetch").mockResolvedValue({
      ok: true,
      json: async () => ({ override_id: "ovr-456", action: "REROUTE" }),
    } as Response);

    vi.spyOn(apiCoreModule, "sessionMapCenter").mockReturnValue({
      lat: 37.7749,
      lng: -122.4194,
    });

    const simplifiedPolygon: validationModule.GeoJSONPolygon = {
      type: "Polygon",
      coordinates: [
        [
          [0, 0],
          [1, 0],
          [1, 1],
          [0, 1],
          [0, 0],
        ],
      ],
    };

    vi.spyOn(validationModule, "validateAndSimplifyPolygon").mockReturnValueOnce({
      valid: true,
      simplified: true,
      vertexCount: 5,
      geojson: simplifiedPolygon,
    });

    render(<ControlTowerCommandPanel />);

    const publishButton = screen.getByRole("button", { name: /Publish zone override/i });
    fireEvent.click(publishButton);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    const fetchCall = mockFetch.mock.calls[0];
    const body = JSON.parse(fetchCall[1]?.body as string);
    expect(body.polygon_geojson).toEqual(simplifiedPolygon);
  });
});
