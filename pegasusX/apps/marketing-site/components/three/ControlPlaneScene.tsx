"use client";

import { CameraRig } from "./CameraRig";

const LAYERS = [
  { z: -2, color: "#2F5BFF", label: "Clients" },
  { z: -0.5, color: "#40E0FF", label: "Maglev" },
  { z: 1, color: "#ffffff", label: "API" },
  { z: 2.5, color: "#00C96B", label: "Spanner" },
  { z: 4, color: "#FF7A18", label: "Redis/Kafka" },
  { z: 5.5, color: "#2F5BFF", label: "WS Hubs" },
];

const KEYFRAMES = LAYERS.map((l, i) => ({
  z: 8 - i * 1.2,
  y: 1.5 + i * 0.1,
  x: 0,
}));

type ControlPlaneSceneContentProps = {
  scrollProgress?: number;
};

export function ControlPlaneSceneContent({
  scrollProgress = 0,
}: ControlPlaneSceneContentProps) {
  return (
    <>
      <ambientLight intensity={0.25} />
      <directionalLight position={[4, 6, 8]} intensity={1} />
      <pointLight position={[-3, 2, 4]} intensity={0.5} color="#2F5BFF" />
      <CameraRig scrollProgress={scrollProgress} keyframes={KEYFRAMES} />
      {LAYERS.map((layer, i) => (
        <group key={layer.label} position={[0, 0, -i * 1.2]}>
          <mesh>
            <boxGeometry args={[4, 2.5, 0.08]} />
            <meshStandardMaterial
              color={layer.color}
              transparent
              opacity={0.35}
              metalness={0.4}
              roughness={0.3}
            />
          </mesh>
          <mesh position={[0, 0, 0.05]}>
            <planeGeometry args={[3.6, 0.4]} />
            <meshBasicMaterial color="#ffffff" transparent opacity={0.08} />
          </mesh>
        </group>
      ))}
    </>
  );
}
