import { z } from "zod";

export const supplierLoginSchema = z.object({
  phone: z.string().trim().min(8).max(32),
  password: z.string().min(6).max(128),
});

export const supplierRegisterAccountSchema = z.object({
  phone: z.string().trim().min(8).max(32),
  password: z.string().min(8).max(128),
  contact_name: z.string().trim().min(1).max(120),
  email: z.string().trim().email().max(254),
});

export type SupplierLoginInput = z.infer<typeof supplierLoginSchema>;
export type SupplierRegisterAccountInput = z.infer<typeof supplierRegisterAccountSchema>;

export type NormalizeEanResult =
  | { ok: true; code: string }
  | { ok: false; error: string };

function validGtinChecksum(code: string): boolean {
  const n = code.length;
  if (n < 8) return false;
  let sum = 0;
  for (let i = 0; i < n - 1; i++) {
    const d = Number(code[i]);
    const posFromRight = n - 1 - i;
    sum += posFromRight % 2 === 1 ? d * 3 : d;
  }
  const check = Number(code[n - 1]);
  return (10 - (sum % 10)) % 10 === check;
}

/** Strip non-digits and validate EAN-8/12/13/14 GTIN checksum (matches backend returns.NormalizeBarcode). */
export function normalizeEanBarcode(raw: string): NormalizeEanResult {
  const digits = raw.replace(/\D/g, "");
  if (!digits) return { ok: false, error: "barcode_required" };
  if (![8, 12, 13, 14].includes(digits.length)) {
    return { ok: false, error: "unsupported_barcode_length" };
  }
  if (!validGtinChecksum(digits)) {
    return { ok: false, error: "invalid_barcode_checksum" };
  }
  return { ok: true, code: digits };
}

// ---------------------------------------------------------------------------
// Geospatial Polygon Validation & Simplification
// ---------------------------------------------------------------------------

export type Position = [number, number] | number[];
export type CoordinateRing = Position[];
export type PolygonCoordinates = CoordinateRing[];

export interface GeoJSONPolygon {
  type: "Polygon";
  coordinates: number[][][];
}

export interface GeoJSONFeature<G = GeoJSONPolygon> {
  type: "Feature";
  geometry: G;
  properties?: Record<string, any> | null;
}

export type PolygonInput =
  | GeoJSONPolygon
  | GeoJSONFeature<GeoJSONPolygon>
  | number[][][]
  | number[][]
  | { type: string; coordinates?: any; geometry?: any }
  | null
  | undefined;

export interface ValidationOptions {
  /** Maximum number of vertices allowed (default: 50) */
  maxVertices?: number;
  /** Whether to automatically simplify polygons exceeding maxVertices (default: true) */
  autoSimplify?: boolean;
  /** Initial RDP simplification tolerance in coordinate units (degrees, default: 0.0001) */
  tolerance?: number;
  /** High quality simplification flag (default: true) */
  highQuality?: boolean;
}

export interface PolygonValidationResult {
  valid: boolean;
  error?: string;
  hasKinks?: boolean;
  vertexCount: number;
  simplified: boolean;
  geojson?: GeoJSONPolygon;
}

/**
 * Remove duplicate consecutive coordinates in a ring.
 */
export function cleanRingCoordinates(ring: CoordinateRing, eps = 1e-9): [number, number][] {
  if (!Array.isArray(ring) || ring.length === 0) return [];
  const cleaned: [number, number][] = [];
  for (let i = 0; i < ring.length; i++) {
    const pt = ring[i];
    if (!Array.isArray(pt) || pt.length < 2 || !Number.isFinite(pt[0]) || !Number.isFinite(pt[1])) {
      continue;
    }
    const curr: [number, number] = [Number(pt[0]), Number(pt[1])];
    if (cleaned.length === 0) {
      cleaned.push(curr);
    } else {
      const prev = cleaned[cleaned.length - 1];
      if (Math.abs(prev[0] - curr[0]) > eps || Math.abs(prev[1] - curr[1]) > eps) {
        cleaned.push(curr);
      }
    }
  }
  return cleaned;
}

/**
 * Normalizes input polygon/feature/coordinates into standard GeoJSON Polygon coordinates.
 */
export function normalizePolygonInput(input: PolygonInput): number[][][] | null {
  if (!input) return null;

  // Raw array of coordinate rings (3D array)
  if (Array.isArray(input)) {
    if (input.length === 0) return null;
    if (Array.isArray(input[0]) && Array.isArray(input[0][0])) {
      return input as number[][][];
    }
    if (Array.isArray(input[0]) && typeof input[0][0] === "number") {
      // Single ring (2D array) passed -> wrap in array of rings
      return [input as unknown as number[][]];
    }
    return null;
  }

  // Feature wrapper
  if (typeof input === "object" && input.type === "Feature" && input.geometry) {
    if (input.geometry.type === "Polygon" && Array.isArray(input.geometry.coordinates)) {
      return input.geometry.coordinates;
    }
    return null;
  }

  // Polygon geometry
  if (typeof input === "object" && input.type === "Polygon" && Array.isArray(input.coordinates)) {
    return input.coordinates;
  }

  return null;
}

/**
 * 2D cross product for segment orientation.
 */
function crossProduct(ax: number, ay: number, bx: number, by: number, cx: number, cy: number): number {
  return (bx - ax) * (cy - ay) - (by - ay) * (cx - ax);
}

/**
 * Checks if point (px, py) lies on line segment (ax, ay) -> (bx, by).
 */
function isPointOnSegment(
  px: number,
  py: number,
  ax: number,
  ay: number,
  bx: number,
  by: number,
  eps = 1e-9
): boolean {
  const minX = Math.min(ax, bx) - eps;
  const maxX = Math.max(ax, bx) + eps;
  const minY = Math.min(ay, by) - eps;
  const maxY = Math.max(ay, by) + eps;
  if (px < minX || px > maxX || py < minY || py > maxY) return false;
  const cross = crossProduct(ax, ay, bx, by, px, py);
  return Math.abs(cross) <= eps;
}

/**
 * Determines if two line segments (a1-a2) and (b1-b2) intersect.
 */
export function segmentsIntersect(
  a1: [number, number],
  a2: [number, number],
  b1: [number, number],
  b2: [number, number],
  eps = 1e-9
): boolean {
  const [x1, y1] = a1;
  const [x2, y2] = a2;
  const [x3, y3] = b1;
  const [x4, y4] = b2;

  const cp1 = crossProduct(x1, y1, x2, y2, x3, y3);
  const cp2 = crossProduct(x1, y1, x2, y2, x4, y4);
  const cp3 = crossProduct(x3, y3, x4, y4, x1, y1);
  const cp4 = crossProduct(x3, y3, x4, y4, x2, y2);

  // Proper intersection (straddle)
  if (
    ((cp1 > eps && cp2 < -eps) || (cp1 < -eps && cp2 > eps)) &&
    ((cp3 > eps && cp4 < -eps) || (cp3 < -eps && cp4 > eps))
  ) {
    return true;
  }

  // Touching / collinear overlap cases
  if (Math.abs(cp1) <= eps && isPointOnSegment(x3, y3, x1, y1, x2, y2, eps)) return true;
  if (Math.abs(cp2) <= eps && isPointOnSegment(x4, y4, x1, y1, x2, y2, eps)) return true;
  if (Math.abs(cp3) <= eps && isPointOnSegment(x1, y1, x3, y3, x4, y4, eps)) return true;
  if (Math.abs(cp4) <= eps && isPointOnSegment(x2, y2, x3, y3, x4, y4, eps)) return true;

  return false;
}

/**
 * Detects self-intersections (kinks) in polygon rings.
 */
export function detectPolygonKinks(rings: number[][][], eps = 1e-9): boolean {
  for (let r = 0; r < rings.length; r++) {
    const ring = rings[r];
    const n = ring.length - 1; // number of unique segments (since ring[0] === ring[n])
    if (n < 3) continue;

    // Check non-adjacent segment intersections within this ring
    for (let i = 0; i < n; i++) {
      const a1 = ring[i] as [number, number];
      const a2 = ring[i + 1] as [number, number];

      for (let j = i + 1; j < n; j++) {
        // Skip adjacent segments sharing an endpoint
        if (j === i + 1) continue;
        if (i === 0 && j === n - 1) continue;

        const b1 = ring[j] as [number, number];
        const b2 = ring[j + 1] as [number, number];

        if (segmentsIntersect(a1, a2, b1, b2, eps)) {
          return true;
        }
      }
    }

    // Check intersections between different rings (e.g. exterior and holes)
    for (let r2 = r + 1; r2 < rings.length; r2++) {
      const ring2 = rings[r2];
      const n2 = ring2.length - 1;
      for (let i = 0; i < n; i++) {
        const a1 = ring[i] as [number, number];
        const a2 = ring[i + 1] as [number, number];
        for (let j = 0; j < n2; j++) {
          const b1 = ring2[j] as [number, number];
          const b2 = ring2[j + 1] as [number, number];
          if (segmentsIntersect(a1, a2, b1, b2, eps)) {
            return true;
          }
        }
      }
    }
  }

  return false;
}

/**
 * Calculates squared perpendicular distance from point p to segment p1-p2.
 */
function getSqSegDist(p: [number, number], p1: [number, number], p2: [number, number]): number {
  let x = p1[0];
  let y = p1[1];
  let dx = p2[0] - x;
  let dy = p2[1] - y;

  if (dx !== 0 || dy !== 0) {
    const t = ((p[0] - x) * dx + (p[1] - y) * dy) / (dx * dx + dy * dy);
    if (t > 1) {
      x = p2[0];
      y = p2[1];
    } else if (t > 0) {
      x += dx * t;
      y += dy * t;
    }
  }

  dx = p[0] - x;
  dy = p[1] - y;
  return dx * dx + dy * dy;
}

/**
 * Recursive Ramer-Douglas-Peucker simplification on a polyline slice.
 */
function simplifyRDPStep(
  points: [number, number][],
  first: number,
  last: number,
  sqTolerance: number,
  simplified: [number, number][]
) {
  let maxSqDist = sqTolerance;
  let index = -1;

  for (let i = first + 1; i < last; i++) {
    const sqDist = getSqSegDist(points[i], points[first], points[last]);
    if (sqDist > maxSqDist) {
      index = i;
      maxSqDist = sqDist;
    }
  }

  if (index !== -1) {
    if (index - first > 1) simplifyRDPStep(points, first, index, sqTolerance, simplified);
    simplified.push(points[index]);
    if (last - index > 1) simplifyRDPStep(points, index, last, sqTolerance, simplified);
  }
}

/**
 * Simplifies an open polyline using RDP algorithm.
 */
function simplifyPolyline(points: [number, number][], tolerance: number): [number, number][] {
  if (points.length <= 2) return [...points];
  const sqTolerance = tolerance * tolerance;
  const simplified: [number, number][] = [points[0]];
  simplifyRDPStep(points, 0, points.length - 1, sqTolerance, simplified);
  simplified.push(points[points.length - 1]);
  return simplified;
}

/**
 * Simplifies a closed polygon ring while preserving the closed ring invariant.
 */
export function simplifyRingRDP(ring: [number, number][], tolerance: number): [number, number][] {
  if (ring.length <= 4) return [...ring];

  // For a closed ring (ring[0] === ring[n-1]), find the point farthest from ring[0] to split the loop
  const p0 = ring[0];
  let maxDist = -1;
  let splitIndex = Math.floor(ring.length / 2);

  for (let i = 1; i < ring.length - 1; i++) {
    const dx = ring[i][0] - p0[0];
    const dy = ring[i][1] - p0[1];
    const distSq = dx * dx + dy * dy;
    if (distSq > maxDist) {
      maxDist = distSq;
      splitIndex = i;
    }
  }

  const part1 = simplifyPolyline(ring.slice(0, splitIndex + 1), tolerance);
  const part2 = simplifyPolyline(ring.slice(splitIndex), tolerance);

  // Combine both simplified chains
  const combined = [...part1.slice(0, -1), ...part2];

  // Ensure ring is closed and has at least 4 points
  if (combined.length < 4) {
    // If simplified below 4 points (triangle), retain the original or largest 4 points
    return [...ring];
  }

  // Ensure first and last are identical
  combined[combined.length - 1] = [combined[0][0], combined[0][1]];
  return combined;
}

/**
 * Validates polygon geometry, checks for self-intersections (kinks), and simplifies
 * vertices if count exceeds maxVertices threshold.
 */
export function validateAndSimplifyPolygon(
  geojson: PolygonInput,
  options?: ValidationOptions
): PolygonValidationResult {
  const maxVertices = options?.maxVertices ?? 50;
  const autoSimplify = options?.autoSimplify !== false;
  const baseTolerance = options?.tolerance ?? 0.0001;

  const rawRings = normalizePolygonInput(geojson);
  if (!rawRings || rawRings.length === 0) {
    return {
      valid: false,
      error: "Invalid polygon geometry: input must be a valid GeoJSON Polygon, Feature, or coordinate array.",
      vertexCount: 0,
      simplified: false,
    };
  }

  // Validate and normalize rings
  const normalizedRings: [number, number][][] = [];
  let initialVertexCount = 0;

  for (let r = 0; r < rawRings.length; r++) {
    const rawRing = rawRings[r];
    if (!Array.isArray(rawRing) || rawRing.length === 0) {
      return {
        valid: false,
        error: `Invalid polygon geometry: ring ${r} contains no coordinates.`,
        vertexCount: 0,
        simplified: false,
      };
    }

    let cleaned = cleanRingCoordinates(rawRing as CoordinateRing);
    if (cleaned.length < 3) {
      return {
        valid: false,
        error: "Invalid polygon geometry: ring must have at least 3 distinct vertices (minimum 4 coordinates when closed).",
        vertexCount: cleaned.length,
        simplified: false,
      };
    }

    // Check if closed (first point equals last point)
    const first = cleaned[0];
    const last = cleaned[cleaned.length - 1];
    const isClosed = Math.abs(first[0] - last[0]) < 1e-9 && Math.abs(first[1] - last[1]) < 1e-9;

    if (!isClosed) {
      // Auto-close ring
      cleaned.push([first[0], first[1]]);
    }

    if (cleaned.length < 4) {
      return {
        valid: false,
        error: "Invalid polygon geometry: ring must have at least 4 coordinates (3 distinct vertices and a closing coordinate).",
        vertexCount: cleaned.length,
        simplified: false,
      };
    }

    // Check distinct non-closing vertices count
    const distinctPoints = new Set(cleaned.slice(0, -1).map((p) => `${p[0]},${p[1]}`));
    if (distinctPoints.size < 3) {
      return {
        valid: false,
        error: "Invalid polygon geometry: ring must contain at least 3 non-coincident vertices.",
        vertexCount: cleaned.length,
        simplified: false,
      };
    }

    normalizedRings.push(cleaned);
    initialVertexCount += cleaned.length;
  }

  // Kink / Self-intersection detection
  const hasKinks = detectPolygonKinks(normalizedRings);
  if (hasKinks) {
    return {
      valid: false,
      error: "Self-intersecting polygon detected (kinks). Polygons must not cross themselves.",
      hasKinks: true,
      vertexCount: initialVertexCount,
      simplified: false,
    };
  }

  // Vertex threshold check
  if (initialVertexCount <= maxVertices) {
    return {
      valid: true,
      vertexCount: initialVertexCount,
      simplified: false,
      geojson: {
        type: "Polygon",
        coordinates: normalizedRings,
      },
    };
  }

  // Vertex count exceeds maxVertices (>50)
  if (!autoSimplify) {
    return {
      valid: false,
      error: `Polygon has ${initialVertexCount} vertices, exceeding maximum allowed of ${maxVertices}.`,
      vertexCount: initialVertexCount,
      simplified: false,
    };
  }

  // Perform iterative Ramer-Douglas-Peucker simplification
  let currentRings = normalizedRings;
  let currentTolerance = baseTolerance;
  let simplifiedCount = initialVertexCount;
  let simplifiedSuccess = false;

  for (let iter = 0; iter < 40; iter++) {
    const candidateRings: [number, number][][] = [];
    let candidateCount = 0;

    for (const ring of normalizedRings) {
      const simplified = simplifyRingRDP(ring, currentTolerance);
      candidateRings.push(simplified);
      candidateCount += simplified.length;
    }

    // Ensure candidate rings do not have self-intersections and satisfy closed invariant
    const candidateKinks = detectPolygonKinks(candidateRings);
    if (!candidateKinks && candidateCount < initialVertexCount) {
      currentRings = candidateRings;
      simplifiedCount = candidateCount;

      if (candidateCount < maxVertices || candidateCount <= maxVertices) {
        simplifiedSuccess = true;
        break;
      }
    }

    // Increase tolerance exponentially
    currentTolerance *= 1.6;
  }

  if (simplifiedCount <= maxVertices) {
    return {
      valid: true,
      vertexCount: simplifiedCount,
      simplified: true,
      geojson: {
        type: "Polygon",
        coordinates: currentRings,
      },
    };
  }

  return {
    valid: false,
    error: `Polygon exceeds maximum allowed vertices (${maxVertices}) and could not be simplified without losing geometry.`,
    vertexCount: initialVertexCount,
    simplified: false,
  };
}
