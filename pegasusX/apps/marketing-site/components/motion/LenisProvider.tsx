"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import Lenis from "lenis";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { useReducedMotion } from "./ReducedMotionProvider";

gsap.registerPlugin(ScrollTrigger);

type LenisContextValue = {
  lenis: Lenis | null;
  scrollProgress: number;
};

const LenisContext = createContext<LenisContextValue>({
  lenis: null,
  scrollProgress: 0,
});

export function LenisProvider({ children }: { children: ReactNode }) {
  const reducedMotion = useReducedMotion();
  const [lenis, setLenis] = useState<Lenis | null>(null);
  const [scrollProgress, setScrollProgress] = useState(0);

  useEffect(() => {
    if (reducedMotion) return;

    const instance = new Lenis({
      lerp: 0.08,
      smoothWheel: true,
      autoRaf: false,
    });

    setLenis(instance);

    ScrollTrigger.scrollerProxy(document.documentElement, {
      scrollTop(value) {
        if (arguments.length && typeof value === "number") {
          instance.scrollTo(value, { immediate: true });
        }
        return instance.scroll;
      },
      getBoundingClientRect() {
        return {
          top: 0,
          left: 0,
          width: window.innerWidth,
          height: window.innerHeight,
        };
      },
      pinType: document.documentElement.style.transform ? "transform" : "fixed",
    });

    instance.on("scroll", ({ progress }: { progress: number }) => {
      setScrollProgress(progress);
      ScrollTrigger.update();
    });

    ScrollTrigger.addEventListener("refresh", () => instance.resize());

    const ticker = (time: number) => {
      instance.raf(time * 1000);
    };
    gsap.ticker.add(ticker);
    gsap.ticker.lagSmoothing(0);

    ScrollTrigger.refresh();

    return () => {
      gsap.ticker.remove(ticker);
      ScrollTrigger.removeEventListener("refresh", () => instance.resize());
      ScrollTrigger.scrollerProxy(document.documentElement, {});
      instance.destroy();
      setLenis(null);
    };
  }, [reducedMotion]);

  return (
    <LenisContext.Provider value={{ lenis, scrollProgress }}>
      {children}
    </LenisContext.Provider>
  );
}

export function useLenis() {
  return useContext(LenisContext);
}
