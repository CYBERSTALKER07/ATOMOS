'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import type { SupplierFleetLiveRoute } from '@pegasusx/types';

export const FLEET_ROUTE_COLORS = [
  '#1b6ef3',
  '#0f9d58',
  '#db4437',
  '#f4b400',
  '#ab47bc',
  '#00838f',
];

type DriverAnimState = {
  driverId: string;
  fromLng: number;
  fromLat: number;
  toLng: number;
  toLat: number;
  startMs: number;
  color: string;
  label: string;
  stale: boolean;
};

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t;
}

function easeOut(t: number): number {
  return t * (2 - t);
}

function driverTarget(route: SupplierFleetLiveRoute): { lat: number; lng: number } | null {
  const location = route.driver_location;
  if (!route.live_location_available || !location) {
    return null;
  }
  const lat = location.lat ?? location.latitude;
  const lng = location.lng ?? location.longitude;
  if (!Number.isFinite(lat) || !Number.isFinite(lng)) {
    return null;
  }
  return { lat, lng };
}

/**
 * Smoothly interpolates driver GPS markers between poll / websocket refreshes.
 */
export function useAnimatedDriverMarkers(
  routes: SupplierFleetLiveRoute[],
  durationMs = 1200,
): GeoJSON.FeatureCollection<GeoJSON.Point> {
  const animRef = useRef<Map<string, DriverAnimState>>(new Map());
  const [collection, setCollection] = useState<GeoJSON.FeatureCollection<GeoJSON.Point>>({
    type: 'FeatureCollection',
    features: [],
  });

  const routesSignature = useMemo(
    () =>
      routes
        .map((route) => {
          const target = driverTarget(route);
          return `${route.driver_id}:${target?.lat ?? ''}:${target?.lng ?? ''}:${route.location_stale ? 1 : 0}`;
        })
        .join('|'),
    [routes],
  );

  useEffect(() => {
    const now = performance.now();
    const activeDrivers = new Set<string>();

    routes.forEach((route, index) => {
      const target = driverTarget(route);
      if (!target) {
        return;
      }
      activeDrivers.add(route.driver_id);
      const existing = animRef.current.get(route.driver_id);
      const progress = existing
        ? Math.min(1, (now - existing.startMs) / durationMs)
        : 1;
      const eased = easeOut(progress);
      const currentLng = existing ? lerp(existing.fromLng, existing.toLng, eased) : target.lng;
      const currentLat = existing ? lerp(existing.fromLat, existing.toLat, eased) : target.lat;

      animRef.current.set(route.driver_id, {
        driverId: route.driver_id,
        fromLng: currentLng,
        fromLat: currentLat,
        toLng: target.lng,
        toLat: target.lat,
        startMs: now,
        color: FLEET_ROUTE_COLORS[index % FLEET_ROUTE_COLORS.length],
        label: route.driver_name || route.driver_id,
        stale: route.location_stale ?? false,
      });
    });

    for (const driverId of [...animRef.current.keys()]) {
      if (!activeDrivers.has(driverId)) {
        animRef.current.delete(driverId);
      }
    }

    let frameId = 0;
    const tick = () => {
      const tickNow = performance.now();
      const features: GeoJSON.Feature<GeoJSON.Point>[] = [];
      for (const anim of animRef.current.values()) {
        const t = Math.min(1, (tickNow - anim.startMs) / durationMs);
        const eased = easeOut(t);
        features.push({
          type: 'Feature',
          properties: {
            color: anim.color,
            label: anim.label,
            stale: anim.stale ? 1 : 0,
          },
          geometry: {
            type: 'Point',
            coordinates: [
              lerp(anim.fromLng, anim.toLng, eased),
              lerp(anim.fromLat, anim.toLat, eased),
            ],
          },
        });
      }
      setCollection({ type: 'FeatureCollection', features });
      frameId = requestAnimationFrame(tick);
    };

    frameId = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frameId);
  }, [durationMs, routesSignature]);

  return collection;
}
