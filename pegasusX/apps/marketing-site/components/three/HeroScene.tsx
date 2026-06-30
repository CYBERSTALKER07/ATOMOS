"use client";

import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";

const NODES: Array<[number, number, number]> = [
  [-3, 0.5, -1],
  [-1, 1.2, 1],
  [1, 0.8, -0.5],
  [3, 1, 0.5],
  [0, 2, 2],
  [-2, -0.5, 2],
];

const EDGES: Array<[number, number]> = [
  [0, 1],
  [1, 2],
  [2, 3],
  [3, 4],
  [4, 0],
  [1, 5],
  [5, 0],
  [2, 4],
];

function NetworkParticles() {
  const ref = useRef<THREE.Points>(null);
  const count = 40;
  const positions = new Float32Array(count * 3);

  for (let i = 0; i < count; i++) {
    const edge = EDGES[i % EDGES.length];
    const a = NODES[edge[0]];
    const b = NODES[edge[1]];
    const t = (i % 10) / 10;
    positions[i * 3] = a[0] + (b[0] - a[0]) * t;
    positions[i * 3 + 1] = a[1] + (b[1] - a[1]) * t;
    positions[i * 3 + 2] = a[2] + (b[2] - a[2]) * t;
  }

  useFrame((state) => {
    if (!ref.current) return;
    ref.current.rotation.y = state.clock.elapsedTime * 0.05;
  });

  return (
    <points ref={ref}>
      <bufferGeometry>
        <bufferAttribute attach="attributes-position" args={[positions, 3]} />
      </bufferGeometry>
      <pointsMaterial size={0.06} color="#40E0FF" transparent opacity={0.8} />
    </points>
  );
}

export function HeroSceneContent() {
  const groupRef = useRef<THREE.Group>(null);

  useFrame((state) => {
    if (groupRef.current) {
      groupRef.current.rotation.y = Math.sin(state.clock.elapsedTime * 0.2) * 0.15;
    }
  });

  return (
    <>
      <ambientLight intensity={0.3} />
      <directionalLight position={[5, 8, 5]} intensity={1.2} color="#ffffff" />
      <pointLight position={[-4, 2, -2]} intensity={0.6} color="#2F5BFF" />
      <group ref={groupRef}>
        {NODES.map(([x, y, z], i) => (
          <mesh key={i} position={[x, y, z]}>
            <sphereGeometry args={[0.18, 16, 16]} />
            <meshStandardMaterial
              color={i % 2 === 0 ? "#2F5BFF" : "#00C96B"}
              emissive={i % 2 === 0 ? "#2F5BFF" : "#00C96B"}
              emissiveIntensity={0.3}
            />
          </mesh>
        ))}
        {EDGES.map(([a, b], i) => {
          const start = new THREE.Vector3(...NODES[a]);
          const end = new THREE.Vector3(...NODES[b]);
          const mid = start.clone().lerp(end, 0.5);
          const len = start.distanceTo(end);
          const dir = end.clone().sub(start).normalize();
          const quat = new THREE.Quaternion().setFromUnitVectors(
            new THREE.Vector3(0, 1, 0),
            dir,
          );
          return (
            <mesh key={`edge-${i}`} position={mid.toArray()} quaternion={quat}>
              <cylinderGeometry args={[0.015, 0.015, len, 6]} />
              <meshStandardMaterial color="#ffffff" transparent opacity={0.15} />
            </mesh>
          );
        })}
        <NetworkParticles />
      </group>
    </>
  );
}
