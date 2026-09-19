'use client';

import React, { useEffect, useRef, useState } from 'react';
import { useTheme } from '@/app/context/ThemeContext';
import { usePerfProfile } from '@/app/hooks/useDevice';

// 4x4 Bayer dithering matrix for authentic ordered halftoning
const BAYER_4X4 = [
 [ 0 / 16, 8 / 16, 2 / 16, 10 / 16],
 [12 / 16, 4 / 16, 14 / 16, 6 / 16],
 [ 3 / 16, 11 / 16, 1 / 16, 9 / 16],
 [15 / 16, 7 / 16, 13 / 16, 5 / 16],
];

interface AnimatedDitherFieldProps {
 className?: string;
 dotSpacing?: number;
 speed?: number;
}

export default function AnimatedDitherField({
 className = '',
 dotSpacing = 6,
 speed = 1.0,
}: AnimatedDitherFieldProps) {
 const canvasRef = useRef<HTMLCanvasElement | null>(null);
 const containerRef = useRef<HTMLDivElement | null>(null);
 const { resolvedTheme } = useTheme();
 const { prefersReducedMotion, isLowEnd } = usePerfProfile();
 const isLight = resolvedTheme === 'light';

 // Mouse interaction coordinates (relative to canvas)
 const mouseRef = useRef<{ x: number; y: number; active: boolean }>({
 x: -1000,
 y: -1000,
 active: false,
 });

 useEffect(() => {
 const canvas = canvasRef.current;
 const container = containerRef.current;
 if (!canvas || !container) return;

 const ctx = canvas.getContext('2d', { alpha: true });
 if (!ctx) return;

 let animationFrameId: number;
 let isVisible = true;
 let width = 0;
 let height = 0;
 let dpr = 1;

 // Responsive canvas sizing with device pixel ratio
 const updateSize = () => {
 const rect = container.getBoundingClientRect();
 width = Math.max(rect.width, 100);
 height = Math.max(rect.height, 80);
 dpr = Math.min(window.devicePixelRatio || 1, 2);

 canvas.width = Math.floor(width * dpr);
 canvas.height = Math.floor(height * dpr);
 canvas.style.width = `${width}px`;
 canvas.style.height = `${height}px`;

 ctx.scale(dpr, dpr);
 };

 updateSize();

 const resizeObserver = new ResizeObserver(() => {
 updateSize();
 if (prefersReducedMotion || isLowEnd) {
 drawFrame(0);
 }
 });
 resizeObserver.observe(container);

 // Pause animation when scrolled out of view to preserve battery & CPU
 const intersectionObserver = new IntersectionObserver(
 ([entry]) => {
 isVisible = entry.isIntersecting;
 },
 { threshold: 0.1 }
 );
 intersectionObserver.observe(container);

 // Render single frame of animated dither
 const drawFrame = (timeSeconds: number) => {
 ctx.clearRect(0, 0, width, height);

 const t = timeSeconds * speed;
 const mouse = mouseRef.current;
 const cols = Math.ceil(width / dotSpacing);
 const rows = Math.ceil(height / dotSpacing);

 // Theme-specific color parameters
 // Dark mode: electric violet, lilac, glowing lavender
 // Light mode: deep tactical violet, indigo, slate purple
 const baseR = isLight ? 124 : 167;
 const baseG = isLight ? 58 : 139;
 const baseB = isLight ? 237 : 250;

 const peakR = isLight ? 99 : 224;
 const peakG = isLight ? 102 : 231;
 const peakB = isLight ? 241 : 255;

 for (let c = 0; c < cols; c++) {
 const x = c * dotSpacing;
 const normX = x / width;

 for (let r = 0; r < rows; r++) {
 const y = r * dotSpacing;
 const normY = y / height;

 // 1. Primary data flow wave: travels left to right (from SQL to Chart)
 const primaryWave = Math.sin(x * 0.045 - t * 2.6 + y * 0.018);

 // 2. Secondary diagonal harmonic shimmer for organic cybernetic texture
 const crossWave = Math.cos(x * 0.025 + y * 0.05 - t * 1.5);

 // 3. Subtle micro-scintillation (individual dot twinkle)
 const twinkle = Math.sin(x * 12.9898 + y * 78.233 + t * 4.2) * 0.12;

 // 4. Mouse proximity ripple
 let mouseFactor = 0;
 if (mouse.active) {
 const dx = x - mouse.x;
 const dy = y - mouse.y;
 const distSq = dx * dx + dy * dy;
 if (distSq < 10000) { // 100px radius
 mouseFactor = Math.exp(-distSq / 3200) * 0.55;
 }
 }

 // 5. Edge vignette: smooth fade out towards borders
 const edgeFadeX = Math.sin(Math.PI * Math.min(Math.max(normX, 0), 1));
 const edgeFadeY = Math.sin(Math.PI * Math.min(Math.max(normY, 0), 1));
 const vignette = Math.pow(edgeFadeX * edgeFadeY, 0.45);

 // Combine wave intensities (0.0 to 1.0)
 const rawIntensity = 0.42 + 0.32 * primaryWave + 0.18 * crossWave + twinkle + mouseFactor;
 const intensity = Math.max(0, Math.min(1, rawIntensity * vignette));

 // 6. Ordered Bayer dither quantization
 const bayerThreshold = BAYER_4X4[c % 4][r % 4];

 if (intensity > bayerThreshold * 0.7) {
 // Normalized brightness above threshold
 const level = Math.min(1, (intensity - bayerThreshold * 0.7) / 0.6);
 const alpha = Math.max(0.12, Math.min(0.95, 0.2 + level * 0.75));

 // Radius scales slightly with intensity for optical depth
 const radius = 0.75 + level * 0.85;

 // Interpolate color between base violet and luminous peak
 const red = Math.round(baseR + (peakR - baseR) * level);
 const green = Math.round(baseG + (peakG - baseG) * level);
 const blue = Math.round(baseB + (peakB - baseB) * level);

 ctx.fillStyle = `rgba(${red}, ${green}, ${blue}, ${alpha.toFixed(3)})`;
 ctx.beginPath();
 ctx.arc(x, y, radius, 0, Math.PI * 2);
 ctx.fill();
 }
 }
 }
 };

 // Static render for users who prefer reduced motion
 if (prefersReducedMotion || isLowEnd) {
 drawFrame(1.5);
 return () => {
 resizeObserver.disconnect();
 intersectionObserver.disconnect();
 };
 }

 // Continuous 60fps render loop
 let startTime: number | null = null;

 const loop = (timestamp: number) => {
 if (!startTime) startTime = timestamp;
 const elapsed = (timestamp - startTime) / 1000;

 if (isVisible) {
 drawFrame(elapsed);
 }

 animationFrameId = requestAnimationFrame(loop);
 };

 animationFrameId = requestAnimationFrame(loop);

 return () => {
 cancelAnimationFrame(animationFrameId);
 resizeObserver.disconnect();
 intersectionObserver.disconnect();
 };
 }, [dotSpacing, speed, prefersReducedMotion, isLowEnd, isLight]);

 // Pointer event listeners for interactive cursor ripple
 const handlePointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
 const container = containerRef.current;
 if (!container) return;
 const rect = container.getBoundingClientRect();
 mouseRef.current = {
 x: e.clientX - rect.left,
 y: e.clientY - rect.top,
 active: true,
 };
 };

 const handlePointerLeave = () => {
 mouseRef.current.active = false;
 };

 return (
 <div
 ref={containerRef}
 onPointerMove={handlePointerMove}
 onPointerLeave={handlePointerLeave}
 className={`relative w-full h-full overflow-hidden pointer-events-auto ${className}`}
 aria-hidden="true"
 >
 <canvas
 ref={canvasRef}
 className="block w-full h-full pointer-events-none"
 />
 </div>
 );
}
