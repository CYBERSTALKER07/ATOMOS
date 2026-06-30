"use client";

import { useRef } from "react";
import { useFrame } from "@react-three/fiber";
import type * as THREE from "three";
import {
  Depot,
  GlobeHologram,
  LaneBoard,
  Payloader,
  Storefront,
  Truck,
} from "./Props";

const ROLE_PROPS = [
  GlobeHologram,
  Depot,
  LaneBoard,
  Truck,
  Storefront,
  Payloader,
] as const;

type RolesParadeSceneContentProps = {
  scrollProgress?: number;
};

export function RolesParadeSceneContent({
  scrollProgress = 0,
}: RolesParadeSceneContentProps) {
  const groupRef = useRef<THREE.Group>(null);

  useFrame(() => {
    if (groupRef.current) {
      const offset = scrollProgress * 10 - 2.5;
      groupRef.current.position.x = -offset;
    }
  });

  return (
    <>
      <ambientLight intensity={0.35} />
      <directionalLight position={[5, 8, 5]} intensity={1} />
      <pointLight position={[-2, 3, 2]} intensity={0.4} color="#40E0FF" />
      <group ref={groupRef}>
        {ROLE_PROPS.map((Prop, i) => (
          <group key={i} position={[i * 3.5, 0, 0]}>
            <Prop position={[0, 0, 0]} />
          </group>
        ))}
      </group>
    </>
  );
}
