"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

export function GsapRouteCleanup() {
  const pathname = usePathname();

  useEffect(() => {
    ScrollTrigger.getAll().forEach((trigger) => trigger.kill());
    gsap.globalTimeline.clear();
    ScrollTrigger.refresh();
  }, [pathname]);

  return null;
}
