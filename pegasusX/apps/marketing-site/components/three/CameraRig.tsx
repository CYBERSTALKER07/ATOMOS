"use client";

import { useFrame, useThree } from "@react-three/fiber";
import { useRef } from "react";
import * as THREE from "three";

type CameraRigProps = {
  scrollProgress?: number;
  keyframes?: Array<{ z: number; y: number; x?: number }>;
};

export function CameraRig({ scrollProgress = 0, keyframes }: CameraRigProps) {
  const { camera } = useThree();
  const target = useRef(new THREE.Vector3(0, 0, 0));

  useFrame(() => {
    if (keyframes && keyframes.length > 0) {
      const scaled = scrollProgress * (keyframes.length - 1);
      const idx = Math.min(Math.floor(scaled), keyframes.length - 2);
      const t = scaled - idx;
      const a = keyframes[idx];
      const b = keyframes[idx + 1];
      if (a && b) {
        camera.position.z = THREE.MathUtils.lerp(a.z, b.z, t);
        camera.position.y = THREE.MathUtils.lerp(a.y, b.y, t);
        camera.position.x = THREE.MathUtils.lerp(a.x ?? 0, b.x ?? 0, t);
      }
    }
    camera.lookAt(target.current);
  });

  return null;
}
