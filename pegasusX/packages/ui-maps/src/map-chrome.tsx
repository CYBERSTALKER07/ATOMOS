"use client";

import { useEffect, useRef, useState, type RefObject } from "react";
import type maplibregl from "maplibre-gl";

/** Defer MapLibre mount until the container is near the viewport. */
export function useLazyMapMount(rootMargin = "120px") {
  const containerRef = useRef<HTMLDivElement>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const node = containerRef.current;
    if (!node || mounted) return;

    if (typeof IntersectionObserver === "undefined") {
      setMounted(true);
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) {
          setMounted(true);
          observer.disconnect();
        }
      },
      { rootMargin },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [mounted, rootMargin]);

  return { containerRef, mounted };
}

/** Remove MapLibre WebGL context on unmount (PX-DESK-3B). */
export function useMapLibreTeardown(mapRef: RefObject<maplibregl.Map | null>) {
  useEffect(() => {
    return () => {
      mapRef.current?.remove();
      mapRef.current = null;
    };
  }, [mapRef]);
}

type Map3DViewToggleProps = {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  className?: string;
};

/** Optional pitched map view — off by default for dock-PC GPU savings. */
export function Map3DViewToggle({ enabled, onChange, className = "" }: Map3DViewToggleProps) {
  return (
    <button
      type="button"
      aria-pressed={enabled}
      className={`rounded-lg border border-black/10 bg-white/90 px-2.5 py-1 text-[11px] font-medium uppercase tracking-wide text-gray-800 shadow-sm backdrop-blur hover:bg-white ${className}`.trim()}
      onClick={() => onChange(!enabled)}
    >
      {enabled ? "2D view" : "3D view"}
    </button>
  );
}
