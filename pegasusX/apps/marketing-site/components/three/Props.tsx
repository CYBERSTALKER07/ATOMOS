"use client";

import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";

export function Truck({ position = [0, 0, 0] as [number, number, number], scrollVelocity = 0 }) {
  const wheelsRef = useRef<THREE.Group>(null);

  useFrame(() => {
    if (wheelsRef.current) {
      wheelsRef.current.rotation.x += scrollVelocity * 0.05 + 0.01;
    }
  });

  return (
    <group position={position}>
      <mesh position={[0, 0.4, 0]}>
        <boxGeometry args={[1.2, 0.5, 0.6]} />
        <meshStandardMaterial color="#2F5BFF" />
      </mesh>
      <mesh position={[0.7, 0.65, 0]}>
        <boxGeometry args={[0.5, 0.4, 0.55]} />
        <meshStandardMaterial color="#ffffff" />
      </mesh>
      <group ref={wheelsRef}>
        {[
          [-0.4, 0.1, 0.25],
          [-0.4, 0.1, -0.25],
          [0.5, 0.1, 0.25],
          [0.5, 0.1, -0.25],
        ].map((pos, i) => (
          <mesh key={i} position={pos as [number, number, number]} rotation={[0, 0, Math.PI / 2]}>
            <cylinderGeometry args={[0.12, 0.12, 0.08, 12]} />
            <meshStandardMaterial color="#111" />
          </mesh>
        ))}
      </group>
    </group>
  );
}

export function Payloader({ position = [0, 0, 0] as [number, number, number] }) {
  return (
    <group position={position}>
      <mesh position={[0, 0.3, 0]}>
        <boxGeometry args={[0.8, 0.3, 0.5]} />
        <meshStandardMaterial color="#FF7A18" />
      </mesh>
      <mesh position={[0, 0.55, 0.2]} rotation={[-0.3, 0, 0]}>
        <boxGeometry args={[0.15, 0.6, 0.08]} />
        <meshStandardMaterial color="#737373" />
      </mesh>
      <mesh position={[0, 0.1, 0]}>
        <boxGeometry args={[0.6, 0.15, 0.4]} />
        <meshStandardMaterial color="#333" />
      </mesh>
    </group>
  );
}

export function Depot({ position = [0, 0, 0] as [number, number, number] }) {
  return (
    <group position={position}>
      <mesh position={[0, 0.5, 0]}>
        <boxGeometry args={[1.5, 1, 1]} />
        <meshStandardMaterial color="#00C96B" transparent opacity={0.7} />
      </mesh>
      <mesh position={[0.6, 0.2, 0.6]}>
        <boxGeometry args={[0.4, 0.4, 0.2]} />
        <meshStandardMaterial color="#40E0FF" />
      </mesh>
    </group>
  );
}

export function Storefront({ position = [0, 0, 0] as [number, number, number] }) {
  return (
    <group position={position}>
      <mesh position={[0, 0.6, 0]}>
        <boxGeometry args={[1, 1.2, 0.6]} />
        <meshStandardMaterial color="#ffffff" transparent opacity={0.85} />
      </mesh>
      <mesh position={[0, 0.3, 0.31]}>
        <planeGeometry args={[0.6, 0.5]} />
        <meshBasicMaterial color="#2F5BFF" transparent opacity={0.5} />
      </mesh>
    </group>
  );
}

export function LaneBoard({ position = [0, 0, 0] as [number, number, number] }) {
  return (
    <group position={position}>
      <mesh position={[0, 0.8, 0]}>
        <boxGeometry args={[1.2, 1.6, 0.1]} />
        <meshStandardMaterial color="#FF7A18" />
      </mesh>
      {[0.5, 0.2, -0.1].map((y, i) => (
        <mesh key={i} position={[0, y, 0.06]}>
          <boxGeometry args={[0.8, 0.15, 0.02]} />
          <meshStandardMaterial color="#fff" transparent opacity={0.4} />
        </mesh>
      ))}
    </group>
  );
}

export function GlobeHologram({ position = [0, 0, 0] as [number, number, number] }) {
  const ref = useRef<THREE.Mesh>(null);
  useFrame((state) => {
    if (ref.current) ref.current.rotation.y = state.clock.elapsedTime * 0.3;
  });
  return (
    <mesh ref={ref} position={position}>
      <icosahedronGeometry args={[0.7, 1]} />
      <meshStandardMaterial color="#2F5BFF" wireframe transparent opacity={0.6} />
    </mesh>
  );
}
