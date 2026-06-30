"use client";

import { SceneCanvas } from "@/components/three/SceneCanvas";
import { Truck } from "@/components/three/Props";

type DispatchCanvasProps = {
  progress: number;
};

export default function DispatchCanvas({ progress }: DispatchCanvasProps) {
  return (
    <SceneCanvas camera={{ position: [0, 3, 8], fov: 45 }} className="pointer-events-none">
      <ambientLight intensity={0.4} />
      <directionalLight position={[4, 6, 4]} intensity={1} />
      <Truck position={[-1 + progress * 2, 0, 0]} scrollVelocity={progress} />
      <Truck position={[2 - progress * 1.5, 0, -1]} scrollVelocity={progress * 0.5} />
    </SceneCanvas>
  );
}
