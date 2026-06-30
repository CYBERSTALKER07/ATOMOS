"use client";

import { useEffect, useState } from "react";
import { LANDING_SECTIONS } from "@/lib/constants";

export function useScrollSpy() {
  const [activeSection, setActiveSection] = useState(LANDING_SECTIONS[0].id);

  useEffect(() => {
    const ids = LANDING_SECTIONS.map((s) => s.id);
    const elements = ids
      .map((id) => document.getElementById(id))
      .filter((el): el is HTMLElement => el !== null);

    if (elements.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio);
        const top = visible[0];
        if (top?.target.id) {
          setActiveSection(top.target.id as typeof activeSection);
        }
      },
      { rootMargin: "-40% 0px -45% 0px", threshold: [0, 0.25, 0.5, 0.75, 1] },
    );

    elements.forEach((el) => observer.observe(el));
    return () => observer.disconnect();
  }, []);

  return activeSection;
}
