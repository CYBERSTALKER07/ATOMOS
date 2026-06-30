"use client";

import { SceneCanvas } from "./SceneCanvas";
import { RolesParadeSceneContent } from "./RolesParadeScene";

type Props = { progress: number };

export default function RolesParadeSceneCanvas({ progress }: Props) {
  return (
    <SceneCanvas camera={{ position: [0, 2, 6], fov: 50 }}>
      <RolesParadeSceneContent scrollProgress={progress} />
    </SceneCanvas>
  );
}
