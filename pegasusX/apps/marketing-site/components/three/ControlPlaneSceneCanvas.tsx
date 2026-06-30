"use client";

import { SceneCanvas } from "./SceneCanvas";
import { ControlPlaneSceneContent } from "./ControlPlaneScene";

type Props = { progress: number };

export default function ControlPlaneSceneCanvas({ progress }: Props) {
  return (
    <SceneCanvas camera={{ position: [0, 1.5, 8], fov: 50 }}>
      <ControlPlaneSceneContent scrollProgress={progress} />
    </SceneCanvas>
  );
}
