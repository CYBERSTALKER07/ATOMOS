"use client";

import { SceneCanvas } from "./SceneCanvas";
import { HeroSceneContent } from "./HeroScene";

export default function HeroSceneCanvas() {
  return (
    <SceneCanvas camera={{ position: [0, 2, 8], fov: 45 }}>
      <HeroSceneContent />
    </SceneCanvas>
  );
}
