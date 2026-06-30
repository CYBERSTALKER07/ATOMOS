"use client";

import { useEffect, useRef, useState } from "react";
import { dispatchPipeline, dispatchStats, dispatchSpecs } from "@/content/landing";
import { PinSection } from "@/components/motion/PinSection";
import { TechIconGrid } from "@/components/icons/TechIconGrid";
import { SectionHeader, SpecTable } from "@/components/docs/SpecTable";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

export function DispatchEngineSection() {
  const reducedMotion = useReducedMotion();
  const [progress, setProgress] = useState(0);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const w = canvas.clientWidth;
    const h = canvas.clientHeight;
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);

    const hexSize = 24;
    const cols = Math.ceil(w / (hexSize * 1.5)) + 2;
    const rows = Math.ceil(h / (hexSize * Math.sqrt(3))) + 2;

    for (let row = 0; row < rows; row++) {
      for (let col = 0; col < cols; col++) {
        const x = col * hexSize * 1.5;
        const y = row * hexSize * Math.sqrt(3) + (col % 2 ? hexSize * 0.866 : 0);
        const heat = Math.sin(col * 0.4 + progress * 6) * 0.5 + 0.5;
        if (heat < 0.35) continue;
        drawHex(ctx, x, y, hexSize * 0.9, `rgba(255,255,255,${heat * 0.15})`);
      }
    }
  }, [progress]);

  const activeStep = Math.min(
    dispatchPipeline.length - 1,
    Math.floor(progress * dispatchPipeline.length),
  );

  const inner = (
    <div className="relative min-h-screen">
      <div className="mx-auto grid min-h-screen max-w-7xl grid-cols-1 items-center gap-10 px-4 py-8 md:grid-cols-2 md:px-6">
        <div>
          <SectionHeader
            platformFrame
            label="Dispatch"
            title="Match orders to trucks — fast and accurate."
            description="Warehouse teams pick trucks and orders on a visual board. Smart suggestions help when you want them. Every load is sealed before it leaves the yard."
            titleId="dispatch-engine-title"
          />
          <div className="mt-8">
            <SpecTable rows={dispatchSpecs} />
          </div>
          <div className="mt-8 flex flex-wrap gap-3">
            {dispatchStats.map((stat) => (
              <div key={stat.label} className="mkt-card px-4 py-3">
                <p className="text-xl font-bold tabular-nums">{stat.value}</p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-[var(--mkt-subtle)]">
                  {stat.label}
                </p>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          <div className="relative h-48 overflow-hidden rounded-lg border border-[var(--mkt-border)]">
            <canvas ref={canvasRef} className="h-full w-full" aria-hidden />
          </div>
          <TechIconGrid icons={["h3", "go", "kafka", "websocket"]} columns={4} />
          <div className="mkt-card p-5">
            <p className="font-mono text-xs uppercase tracking-wider text-[var(--mkt-subtle)]">
              Step {activeStep + 1} / {dispatchPipeline.length}
            </p>
            <p className="mt-2 font-medium">{dispatchPipeline[activeStep]}</p>
            <ol className="mt-4 space-y-1 text-xs text-[var(--mkt-subtle)]">
              {dispatchPipeline.map((step, i) => (
                <li key={step} className={i === activeStep ? "text-[var(--mkt-text)]" : ""}>
                  {i + 1}. {step}
                </li>
              ))}
            </ol>
          </div>
        </div>
      </div>
    </div>
  );

  if (reducedMotion) {
    return (
      <section id="dispatch-engine" aria-labelledby="dispatch-engine-title" className="py-24">
        <div className="mx-auto max-w-7xl px-4 md:px-6">{inner}</div>
      </section>
    );
  }

  return (
    <PinSection id="dispatch-engine" end="+=250%" onProgress={setProgress}>
      {inner}
    </PinSection>
  );
}

function drawHex(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  size: number,
  fill: string,
) {
  ctx.beginPath();
  for (let i = 0; i < 6; i++) {
    const angle = (Math.PI / 3) * i - Math.PI / 6;
    const px = x + size * Math.cos(angle);
    const py = y + size * Math.sin(angle);
    if (i === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  }
  ctx.closePath();
  ctx.fillStyle = fill;
  ctx.fill();
  ctx.strokeStyle = "rgba(255,255,255,0.08)";
  ctx.stroke();
}
