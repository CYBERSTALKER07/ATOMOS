"use client";

import { type ReactNode } from "react";

type AssetSlotProps = {
  slotId: "object-a-hero" | "object-b-layers" | "object-c-roles";
  assetPath?: string;
  videoPath?: string;
  children?: ReactNode;
  className?: string;
  minHeight?: string;
};

export function AssetSlot({
  slotId,
  assetPath,
  videoPath,
  children,
  className = "",
  minHeight = "min-h-[280px]",
}: AssetSlotProps) {
  if (children) {
    return (
      <div className={`asset-slot ${minHeight} ${className}`.trim()}>
        {children}
      </div>
    );
  }

  if (videoPath) {
    return (
      <div className={`asset-slot ${minHeight} ${className}`.trim()}>
        <video
          autoPlay
          muted
          loop
          playsInline
          className="h-full w-full object-cover"
          poster={`/images/${slotId}-poster.jpg`}
        >
          <source src={videoPath} type="video/mp4" />
        </video>
      </div>
    );
  }

  return (
    <div className={`asset-slot ${minHeight} ${className}`.trim()}>
      <div className="asset-slot__placeholder">
        Slot {slotId}
        {assetPath ? ` · ${assetPath}` : ""}
      </div>
    </div>
  );
}
