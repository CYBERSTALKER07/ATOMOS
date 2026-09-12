import { describe, expect, it } from "vitest";
import {
  cleanRingCoordinates,
  detectPolygonKinks,
  segmentsIntersect,
  simplifyRingRDP,
  validateAndSimplifyPolygon,
  type GeoJSONPolygon,
  type GeoJSONFeature,
} from "../index";

describe("Geospatial Polygon Validation & Simplification", () => {
  describe("Segment Intersection & Kink Detection", () => {
    it("detects intersection between crossing segments", () => {
      const a1: [number, number] = [0, 0];
      const a2: [number, number] = [2, 2];
      const b1: [number, number] = [0, 2];
      const b2: [number, number] = [2, 0];
      expect(segmentsIntersect(a1, a2, b1, b2)).toBe(true);
    });

    it("does not detect intersection between parallel non-overlapping segments", () => {
      const a1: [number, number] = [0, 0];
      const a2: [number, number] = [2, 0];
      const b1: [number, number] = [0, 2];
      const b2: [number, number] = [2, 2];
      expect(segmentsIntersect(a1, a2, b1, b2)).toBe(false);
    });

    it("detects kinks in a bowtie / hourglass self-intersecting polygon", () => {
      const bowtie: number[][][] = [
        [
          [0, 0],
          [2, 2],
          [0, 2],
          [2, 0],
          [0, 0],
        ],
      ];
      expect(detectPolygonKinks(bowtie)).toBe(true);
    });

    it("does not detect kinks in a simple valid rectangle", () => {
      const rect: number[][][] = [
        [
          [0, 0],
          [2, 0],
          [2, 2],
          [0, 2],
          [0, 0],
        ],
      ];
      expect(detectPolygonKinks(rect)).toBe(false);
    });
  });

  describe("Validation: Valid Simple Polygons", () => {
    it("passes a valid simple polygon without simplification", () => {
      const polygon: GeoJSONPolygon = {
        type: "Polygon",
        coordinates: [
          [
            [10.0, 50.0],
            [10.1, 50.0],
            [10.1, 50.1],
            [10.0, 50.1],
            [10.0, 50.0],
          ],
        ],
      };

      const result = validateAndSimplifyPolygon(polygon);
      expect(result.valid).toBe(true);
      expect(result.simplified).toBe(false);
      expect(result.vertexCount).toBe(5);
      expect(result.geojson).toBeDefined();
      expect(result.geojson?.type).toBe("Polygon");
      expect(result.geojson?.coordinates[0].length).toBe(5);
    });

    it("accepts GeoJSON Feature wrapped polygon", () => {
      const feature: GeoJSONFeature = {
        type: "Feature",
        geometry: {
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
        },
      };

      const result = validateAndSimplifyPolygon(feature);
      expect(result.valid).toBe(true);
      expect(result.simplified).toBe(false);
      expect(result.vertexCount).toBe(5);
    });

    it("accepts raw 3D coordinate array", () => {
      const coords = [
        [
          [0, 0],
          [4, 0],
          [4, 4],
          [0, 4],
          [0, 0],
        ],
      ];

      const result = validateAndSimplifyPolygon(coords);
      expect(result.valid).toBe(true);
      expect(result.vertexCount).toBe(5);
    });
  });

  describe("Auto-Simplification for Complex Polygons (>50 Vertices)", () => {
    it("auto-simplifies a 64+ vertex circle down to <50 vertices preserving closed invariant", () => {
      // Create a 64-vertex circle polygon
      const numPoints = 64;
      const center = [37.7749, -122.4194];
      const radius = 0.05;
      const coordinates: [number, number][] = [];

      for (let i = 0; i <= numPoints; i++) {
        const angle = (i / numPoints) * 2 * Math.PI;
        coordinates.push([
          center[0] + radius * Math.cos(angle),
          center[1] + radius * Math.sin(angle),
        ]);
      }

      const complexPolygon: GeoJSONPolygon = {
        type: "Polygon",
        coordinates: [coordinates],
      };

      // Ensure test input has >50 vertices
      expect(complexPolygon.coordinates[0].length).toBe(65);

      const result = validateAndSimplifyPolygon(complexPolygon);
      expect(result.valid).toBe(true);
      expect(result.simplified).toBe(true);
      expect(result.vertexCount).toBeLessThan(50);
      expect(result.vertexCount).toBeGreaterThanOrEqual(4);

      const simplifiedRing = result.geojson?.coordinates[0];
      expect(simplifiedRing).toBeDefined();
      expect(simplifiedRing!.length).toBeLessThan(50);

      // Verify ring is closed: first point equals last point
      const first = simplifiedRing![0];
      const last = simplifiedRing![simplifiedRing!.length - 1];
      expect(first[0]).toBeCloseTo(last[0], 6);
      expect(first[1]).toBeCloseTo(last[1], 6);
    });

    it("rejects complex polygon when autoSimplify is false", () => {
      const numPoints = 60;
      const coordinates: [number, number][] = [];
      for (let i = 0; i <= numPoints; i++) {
        const angle = (i / numPoints) * 2 * Math.PI;
        coordinates.push([Math.cos(angle), Math.sin(angle)]);
      }

      const complexPolygon: GeoJSONPolygon = {
        type: "Polygon",
        coordinates: [coordinates],
      };

      const result = validateAndSimplifyPolygon(complexPolygon, { autoSimplify: false });
      expect(result.valid).toBe(false);
      expect(result.simplified).toBe(false);
      expect(result.error).toContain("exceeding maximum allowed of 50");
    });
  });

  describe("Kink / Self-Intersection Rejection", () => {
    it("rejects bowtie / self-intersecting polygon with descriptive error", () => {
      const bowtiePolygon: GeoJSONPolygon = {
        type: "Polygon",
        coordinates: [
          [
            [0, 0],
            [2, 2],
            [0, 2],
            [2, 0],
            [0, 0],
          ],
        ],
      };

      const result = validateAndSimplifyPolygon(bowtiePolygon);
      expect(result.valid).toBe(false);
      expect(result.hasKinks).toBe(true);
      expect(result.error).toBe(
        "Self-intersecting polygon detected (kinks). Polygons must not cross themselves."
      );
    });

    it("rejects complex figure-8 crossing polygon", () => {
      const figure8: GeoJSONPolygon = {
        type: "Polygon",
        coordinates: [
          [
            [0, 0],
            [1, 1],
            [0, 2],
            [2, 2],
            [1, 1],
            [2, 0],
            [0, 0],
          ],
        ],
      };

      const result = validateAndSimplifyPolygon(figure8);
      expect(result.valid).toBe(false);
      expect(result.hasKinks).toBe(true);
      expect(result.error).toContain("Self-intersecting polygon detected (kinks)");
    });
  });

  describe("Invalid Geometry & Unclosed Polygons", () => {
    it("auto-closes an open polygon ring with >= 3 distinct points", () => {
      const unclosedTriangle = [
        [
          [0, 0],
          [2, 0],
          [1, 2],
        ],
      ];

      const result = validateAndSimplifyPolygon(unclosedTriangle);
      expect(result.valid).toBe(true);
      expect(result.vertexCount).toBe(4);
      const ring = result.geojson?.coordinates[0];
      expect(ring?.length).toBe(4);
      expect(ring![0]).toEqual(ring![3]);
    });

    it("rejects degenerate ring with fewer than 3 distinct points", () => {
      const lineSegment = [
        [
          [0, 0],
          [1, 1],
        ],
      ];

      const result = validateAndSimplifyPolygon(lineSegment);
      expect(result.valid).toBe(false);
      expect(result.error).toContain("at least 3 distinct vertices");
    });

    it("rejects null or undefined input", () => {
      const resultNull = validateAndSimplifyPolygon(null as any);
      expect(resultNull.valid).toBe(false);
      expect(resultNull.error).toContain("Invalid polygon geometry");

      const resultUndefined = validateAndSimplifyPolygon(undefined as any);
      expect(resultUndefined.valid).toBe(false);
      expect(resultUndefined.error).toContain("Invalid polygon geometry");
    });

    it("rejects empty coordinates or invalid object types", () => {
      const emptyPolygon: GeoJSONPolygon = {
        type: "Polygon",
        coordinates: [],
      };

      const result = validateAndSimplifyPolygon(emptyPolygon);
      expect(result.valid).toBe(false);
      expect(result.error).toContain("Invalid polygon geometry");
    });
  });

  describe("Collinear Coordinate Cleaning & RDP Algorithm", () => {
    it("cleans duplicate consecutive coordinates", () => {
      const rawRing = [
        [0, 0],
        [0, 0],
        [1, 1],
        [1, 1],
        [2, 2],
      ];
      const cleaned = cleanRingCoordinates(rawRing);
      expect(cleaned.length).toBe(3);
      expect(cleaned).toEqual([
        [0, 0],
        [1, 1],
        [2, 2],
      ]);
    });

    it("simplifies closed ring preserving closure", () => {
      const ring: [number, number][] = [
        [0, 0],
        [1, 0.001],
        [2, 0],
        [2, 2],
        [0, 2],
        [0, 0],
      ];
      const simplified = simplifyRingRDP(ring, 0.01);
      expect(simplified.length).toBe(5); // [0,0], [2,0], [2,2], [0,2], [0,0]
      expect(simplified[0]).toEqual(simplified[simplified.length - 1]);
    });
  });
});
