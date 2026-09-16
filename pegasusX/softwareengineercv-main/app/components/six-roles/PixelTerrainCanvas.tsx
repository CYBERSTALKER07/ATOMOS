'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { usePerfProfile } from '@/app/hooks/useDevice';

interface PixelTerrainCanvasProps {
  className?: string;
  onCellHover?: (col: number, row: number, type: 'light' | 'dark') => void;
  activeRoleIndex?: number | null;
}

export default function PixelTerrainCanvas({
  className = '',
  onCellHover,
  activeRoleIndex,
}: PixelTerrainCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mouseRef = useRef<{ col: number; row: number; active: boolean }>({
    col: -1,
    row: -1,
    active: false,
  });

  const { prefersReducedMotion, isLowEnd } = usePerfProfile();

  useEffect(() => {
    const canvas = canvasRef.current;
    const container = containerRef.current;
    if (!canvas || !container) return;

    const ctx = canvas.getContext('2d', { alpha: false });
    if (!ctx) return;

    let width = 0;
    let height = 0;
    let dpr = 1;
    let animationFrameId = 0;

    const CELL_SIZE = 28; // 28px square cells matching reference aesthetic

    const updateSize = () => {
      const rect = container.getBoundingClientRect();
      width = Math.max(rect.width, 300);
      height = Math.max(rect.height, 350);
      dpr = Math.min(window.devicePixelRatio || 1, 2);

      canvas.width = Math.floor(width * dpr);
      canvas.height = Math.floor(height * dpr);
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;

      ctx.scale(dpr, dpr);
      draw();
    };

    // Deterministic terrain boundary generator matching user reference image
    const isDarkCell = (c: number, r: number, cols: number, rows: number): boolean => {
      const nx = c / cols;
      const ny = r / rows;

      // Base stepped contour curve
      let boundY = 0.54 - 0.22 * Math.sin(nx * Math.PI * 1.35);

      // Distinct peninsulas & bays from reference screenshot
      if (nx >= 0.42 && nx <= 0.52) {
        boundY -= 0.12; // central stepped rise
      } else if (nx >= 0.64 && nx <= 0.74) {
        boundY += 0.08; // hover bay
      } else if (nx > 0.74) {
        boundY -= 0.25 * ((nx - 0.74) / 0.26); // eastern ridge climbing up
      }

      // Discrete grid snapping
      const boundRow = Math.floor(boundY * rows);
      let isDark = r >= boundRow;

      // Floating dark pixels in light zone (top-left)
      const colDistFromCenter = Math.abs(c - Math.floor(cols * 0.25));
      const rowDistFromTop = Math.abs(r - Math.floor(rows * 0.22));
      if (colDistFromCenter <= 1 && rowDistFromTop <= 1) isDark = true;

      if (c === Math.floor(cols * 0.08) && r === Math.floor(rows * 0.48)) isDark = true;
      if (c === Math.floor(cols * 0.38) && r === Math.floor(rows * 0.44)) isDark = true;
      if (c === Math.floor(cols * 0.58) && (r === Math.floor(rows * 0.28) || r === Math.floor(rows * 0.32))) isDark = true;

      // Floating light pixels in dark zone (bottom-right)
      if (
        (c === Math.floor(cols * 0.48) && r === Math.floor(rows * 0.75)) ||
        (c === Math.floor(cols * 0.49) && r === Math.floor(rows * 0.75)) ||
        (c === Math.floor(cols * 0.54) && r === Math.floor(rows * 0.82)) ||
        (c === Math.floor(cols * 0.62) && (r === Math.floor(rows * 0.78) || r === Math.floor(rows * 0.82))) ||
        (c === Math.floor(cols * 0.82) && r === Math.floor(rows * 0.62)) ||
        (c === Math.floor(cols * 0.86) && (r === Math.floor(rows * 0.68) || r === Math.floor(rows * 0.72)))
      ) {
        isDark = false;
      }

      return isDark;
    };

    const draw = () => {
      const cols = Math.ceil(width / CELL_SIZE);
      const rows = Math.ceil(height / CELL_SIZE);

      // Precompute terrain matrix
      const grid: boolean[][] = [];
      for (let c = 0; c < cols; c++) {
        grid[c] = [];
        for (let r = 0; r < rows; r++) {
          grid[c][r] = isDarkCell(c, r, cols, rows);
        }
      }

      // Base background colors from reference: Crisp Bone Light (#ECECED) vs Dark Obsidian (#121216)
      const colorLight = '#EDEDED';
      const colorDark = '#121216';
      const colorGridLine = 'rgba(0, 0, 0, 0.03)';

      // 1. Fill entire canvas with light
      ctx.fillStyle = colorLight;
      ctx.fillRect(0, 0, width, height);

      // 2. Draw dark cells
      ctx.fillStyle = colorDark;
      for (let c = 0; c < cols; c++) {
        for (let r = 0; r < rows; r++) {
          if (grid[c][r]) {
            ctx.fillRect(c * CELL_SIZE, r * CELL_SIZE, CELL_SIZE, CELL_SIZE);
          }
        }
      }

      // 3. Draw cellular grid lines for crisp technical structure
      ctx.strokeStyle = colorGridLine;
      ctx.lineWidth = 1;
      ctx.beginPath();
      for (let c = 0; c <= cols; c++) {
        ctx.moveTo(c * CELL_SIZE, 0);
        ctx.lineTo(c * CELL_SIZE, height);
      }
      for (let r = 0; r <= rows; r++) {
        ctx.moveTo(0, r * CELL_SIZE);
        ctx.lineTo(width, r * CELL_SIZE);
      }
      ctx.stroke();

      // 4. Compute and render Minesweeper distance numbers along borders
      ctx.font = '500 9px monospace, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      for (let c = 0; c < cols; c++) {
        for (let r = 0; r < rows; r++) {
          const isDark = grid[c][r];

          // Count dark neighbors if cell is light
          if (!isDark) {
            let darkNeighbors = 0;
            for (let dc = -1; dc <= 1; dc++) {
              for (let dr = -1; dr <= 1; dr++) {
                if (dc === 0 && dr === 0) continue;
                const nc = c + dc;
                const nr = r + dr;
                if (nc >= 0 && nc < cols && nr >= 0 && nr < rows) {
                  if (grid[nc][nr]) darkNeighbors++;
                }
              }
            }

            if (darkNeighbors > 0 && darkNeighbors <= 4) {
              ctx.fillStyle = darkNeighbors >= 3 ? '#71717A' : '#A1A1AA';
              ctx.fillText(
                darkNeighbors.toString(),
                c * CELL_SIZE + CELL_SIZE / 2,
                r * CELL_SIZE + CELL_SIZE / 2
              );
            }
          } else {
            // In dark zone, count light neighbors near floating light blocks
            let lightNeighbors = 0;
            for (let dc = -1; dc <= 1; dc++) {
              for (let dr = -1; dr <= 1; dr++) {
                if (dc === 0 && dr === 0) continue;
                const nc = c + dc;
                const nr = r + dr;
                if (nc >= 0 && nc < cols && nr >= 0 && nr < rows) {
                  if (!grid[nc][nr]) lightNeighbors++;
                }
              }
            }

            // Only show numbers immediately surrounding the floating white clusters
            if (lightNeighbors > 0 && lightNeighbors <= 3) {
              ctx.fillStyle = '#52525B';
              ctx.fillText(
                lightNeighbors.toString(),
                c * CELL_SIZE + CELL_SIZE / 2,
                r * CELL_SIZE + CELL_SIZE / 2
              );
            }
          }
        }
      }

      // 5. Render interactive hovered cell highlight (matching blue glow in reference)
      const mouse = mouseRef.current;
      if (mouse.active && mouse.col >= 0 && mouse.col < cols && mouse.row >= 0 && mouse.row < rows) {
        const hx = mouse.col * CELL_SIZE;
        const hy = mouse.row * CELL_SIZE;

        // Electric blue hover box with inner radiance
        ctx.fillStyle = 'rgba(37, 99, 235, 0.38)';
        ctx.fillRect(hx, hy, CELL_SIZE, CELL_SIZE);

        ctx.strokeStyle = 'rgba(96, 165, 250, 0.85)';
        ctx.lineWidth = 1.5;
        ctx.strokeRect(hx + 0.75, hy + 0.75, CELL_SIZE - 1.5, CELL_SIZE - 1.5);

        // Subtle crosshair tick in center
        ctx.strokeStyle = '#FFFFFF';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(hx + CELL_SIZE / 2 - 2, hy + CELL_SIZE / 2);
        ctx.lineTo(hx + CELL_SIZE / 2 + 2, hy + CELL_SIZE / 2);
        ctx.moveTo(hx + CELL_SIZE / 2, hy + CELL_SIZE / 2 - 2);
        ctx.lineTo(hx + CELL_SIZE / 2, hy + CELL_SIZE / 2 + 2);
        ctx.stroke();
      }

      // 6. If a specific role is highlighted externally, highlight its network sector
      if (typeof activeRoleIndex === 'number' && activeRoleIndex >= 0) {
        const targetCol = Math.floor((cols / 7) * (activeRoleIndex + 1));
        const targetRow = Math.floor(rows * 0.65);
        if (targetCol >= 0 && targetCol < cols && targetRow >= 0 && targetRow < rows) {
          ctx.fillStyle = 'rgba(16, 185, 129, 0.3)';
          ctx.fillRect(targetCol * CELL_SIZE, targetRow * CELL_SIZE, CELL_SIZE, CELL_SIZE);
          ctx.strokeStyle = '#34D399';
          ctx.lineWidth = 1.5;
          ctx.strokeRect(targetCol * CELL_SIZE + 0.75, targetRow * CELL_SIZE + 0.75, CELL_SIZE - 1.5, CELL_SIZE - 1.5);
        }
      }
    };

    updateSize();

    const resizeObserver = new ResizeObserver(() => {
      updateSize();
    });
    resizeObserver.observe(container);

    // Mouse movement inside canvas container
    const handleMouseMove = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;

      const col = Math.floor(x / CELL_SIZE);
      const row = Math.floor(y / CELL_SIZE);

      if (mouseRef.current.col !== col || mouseRef.current.row !== row || !mouseRef.current.active) {
        mouseRef.current = { col, row, active: true };
        draw();

        if (onCellHover) {
          const cols = Math.ceil(width / CELL_SIZE);
          const rows = Math.ceil(height / CELL_SIZE);
          const isDark = isDarkCell(col, row, cols, rows);
          onCellHover(col, row, isDark ? 'dark' : 'light');
        }
      }
    };

    const handleMouseLeave = () => {
      mouseRef.current.active = false;
      draw();
    };

    container.addEventListener('mousemove', handleMouseMove);
    container.addEventListener('mouseleave', handleMouseLeave);

    return () => {
      cancelAnimationFrame(animationFrameId);
      resizeObserver.disconnect();
      container.removeEventListener('mousemove', handleMouseMove);
      container.removeEventListener('mouseleave', handleMouseLeave);
    };
  }, [onCellHover, activeRoleIndex]);

  return (
    <div
      ref={containerRef}
      className={`relative w-full h-full select-none cursor-crosshair ${className}`}
    >
      <canvas
        ref={canvasRef}
        className="block w-full h-full pointer-events-none"
      />
    </div>
  );
}
