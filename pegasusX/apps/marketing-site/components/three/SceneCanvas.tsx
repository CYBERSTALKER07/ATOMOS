"use client";

import { Suspense, type ReactNode } from "react";
import { Canvas } from "@react-three/fiber";

type SceneCanvasProps = {
  children: ReactNode;
  className?: string;
  camera?: { position: [number, number, number]; fov?: number };
};

export function SceneCanvas({
  children,
  className = "",
  camera = { position: [0, 2, 8], fov: 45 },
}: SceneCanvasProps) {
  return (
    <div className={`absolute inset-0 ${className}`.trim()}>
      <Canvas
        camera={camera}
        dpr={[1, 1.5]}
        gl={{ antialias: true, alpha: true }}
        style={{ background: "transparent" }}
      >
        <color attach="background" args={["#0a0a0a"]} />
        <Suspense fallback={null}>{children}</Suspense>
      </Canvas>
    </div>
  );
}

export function SceneFallback() {
  return (
    <div className="absolute inset-0 bg-gradient-to-b from-[var(--mkt-canvas-deep)] to-[var(--mkt-canvas-base)]" />
  );
}
