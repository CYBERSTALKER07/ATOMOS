'use client';

import { useEffect, useRef, useState } from 'react';
import { useReducedMotion, useIsMobile } from '../hooks/useDevice';

// Standard WebGPU bitflags
const GPU_BUFFER_USAGE = {
  COPY_DST: 0x0008,
  VERTEX: 0x0020,
  UNIFORM: 0x0040,
};

export default function IsometricTerrain() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const prefersReducedMotion = useReducedMotion();
  const { isMobile } = useIsMobile();
  const [activeRenderer, setActiveRenderer] = useState<'webgpu' | 'canvas2d'>('canvas2d');

  useEffect(() => {
    const canvas = canvasRef.current;
    const container = containerRef.current;
    if (!canvas || !container) return;

    const currentCanvas = canvas;
    const currentContainer = container;

    let isDestroyed = false;
    let isVisible = true;
    let animationFrameId: number = 0;
    let resumeRender: (() => void) | null = null;

    // Detect hardware capabilities
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const nav = typeof navigator !== 'undefined' ? (navigator as any) : null;
    const concurrency = nav?.hardwareConcurrency ?? 8;
    const deviceMemory = nav?.deviceMemory;
    const isLowEndDevice = isMobile || concurrency <= 4 || (deviceMemory !== undefined && deviceMemory <= 4);

    // Dynamic grid density & resolution settings based on device tier
    const COLS = isLowEndDevice ? 16 : 28;
    const ROWS = isLowEndDevice ? 16 : 28;
    const targetFps = isLowEndDevice ? 30 : 60;
    const frameInterval = 1000 / targetFps;

    // Parallax tilt state
    let targetTiltX = 0;
    let targetTiltY = 0;
    let currentTiltX = 0;
    let currentTiltY = 0;

    const handleMouseMove = (e: MouseEvent) => {
      const rect = currentCanvas.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width - 0.5;
      const y = (e.clientY - rect.top) / rect.height - 0.5;
      targetTiltX = y * 0.15;
      targetTiltY = x * 0.2;
    };

    const handleMouseLeave = () => {
      targetTiltX = 0;
      targetTiltY = 0;
    };

    window.addEventListener('mousemove', handleMouseMove, { passive: true });

    // Precompute static terrain elevation base to eliminate CPU math overhead on low-end devices
    const stepU = 2 / (COLS - 1);
    const stepV = 2 / (ROWS - 1);
    const baseElevations = new Float32Array(COLS * ROWS);

    for (let i = 0; i < COLS; i++) {
      const u = -1 + i * stepU;
      for (let j = 0; j < ROWS; j++) {
        const v = -1 + j * stepV;
        // Peak 1: prominent mountain peak
        const dx1 = u - (-0.22);
        const dy1 = v - (-0.12);
        const d1 = Math.sqrt(dx1 * dx1 + dy1 * dy1);
        const p1 = Math.exp(-d1 * 4.2) * 115;

        // Peak 2: ridge near center-right
        const dx2 = u - (0.18);
        const dy2 = v - (0.08);
        const d2 = Math.sqrt(dx2 * dx2 + dy2 * dy2);
        const p2 = Math.exp(-d2 * 3.6) * 88;

        // Peak 3: lower ridge
        const dx3 = u - (-0.38);
        const dy3 = v - (0.32);
        const d3 = Math.sqrt(dx3 * dx3 + dy3 * dy3);
        const p3 = Math.exp(-d3 * 4.8) * 52;

        baseElevations[i * ROWS + j] = p1 + p2 + p3;
      }
    }

    // -------------------------------------------------------------
    // WEBGPU ENGINE INITIALIZER (Tier 1)
    // -------------------------------------------------------------
    async function startWebGPU(): Promise<boolean> {
      if (!nav || !nav.gpu) return false;

      try {
        const adapter = await nav.gpu.requestAdapter();
        if (!adapter) return false;

        const device = await adapter.requestDevice();
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const ctx = (currentCanvas as any).getContext('webgpu');
        if (!ctx) return false;

        const format = nav.gpu.getPreferredCanvasFormat ? nav.gpu.getPreferredCanvasFormat() : 'bgra8unorm';
        ctx.configure({
          device,
          format,
          alphaMode: 'premultiplied',
        });

        const shaderModule = device.createShaderModule({
          label: 'IsometricTerrainWGSL',
          code: `
            struct Uniforms {
              origin: vec2<f32>,
              resolution: vec2<f32>,
              tilt: vec2<f32>,
              scale: f32,
              time: f32,
            };

            @group(0) @binding(0) var<uniform> u: Uniforms;

            struct VertexInput {
              @location(0) position: vec3<f32>,
              @location(1) color: vec4<f32>,
            };

            struct VertexOutput {
              @builtin(position) clip_pos: vec4<f32>,
              @location(0) color: vec4<f32>,
            };

            @vertex
            fn vs_main(in: VertexInput) -> VertexOutput {
              var out: VertexOutput;
              let cos_a = 0.8660254; // cos(30 deg)
              let sin_a = 0.5;       // sin(30 deg)

              let world_x = in.position.x * u.scale;
              let world_y = in.position.y * u.scale;
              var z = in.position.z;

              if (z > 0.0) {
                let u_coord = in.position.x;
                let v_coord = in.position.y;
                z += sin(u_coord * 5.0 + u.time * 0.0015) * cos(v_coord * 4.0 + u.time * 0.001) * 7.0;
              }

              // Apply 3D tilt rotation
              let rx = world_x * cos(u.tilt.y) - world_y * sin(u.tilt.y);
              let ry = (world_x * sin(u.tilt.y) + world_y * cos(u.tilt.y)) * cos(u.tilt.x) - z * sin(u.tilt.x);
              let rz = z * cos(u.tilt.x) + (world_x * sin(u.tilt.y) + world_y * cos(u.tilt.y)) * sin(u.tilt.x);

              let screen_x = u.origin.x + (rx - ry) * cos_a;
              let screen_y = u.origin.y + (rx + ry) * sin_a - rz;

              // Convert screen pixels to Normalized Device Coordinates [-1, 1]
              let ndc_x = (screen_x / u.resolution.x) * 2.0 - 1.0;
              let ndc_y = 1.0 - (screen_y / u.resolution.y) * 2.0;

              out.clip_pos = vec4<f32>(ndc_x, ndc_y, 0.0, 1.0);
              out.color = in.color;
              return out;
            }

            @fragment
            fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
              return in.color;
            }
          `,
        });

        const pipeline = device.createRenderPipeline({
          label: 'TerrainLinePipeline',
          layout: 'auto',
          vertex: {
            module: shaderModule,
            entryPoint: 'vs_main',
            buffers: [
              {
                arrayStride: 7 * 4, // 3 floats pos + 4 floats color
                attributes: [
                  { shaderLocation: 0, offset: 0, format: 'float32x3' },
                  { shaderLocation: 1, offset: 12, format: 'float32x4' },
                ],
              },
            ],
          },
          fragment: {
            module: shaderModule,
            entryPoint: 'fs_main',
            targets: [
              {
                format,
                blend: {
                  color: {
                    srcFactor: 'src-alpha',
                    dstFactor: 'one-minus-src-alpha',
                    operation: 'add',
                  },
                  alpha: {
                    srcFactor: 'one',
                    dstFactor: 'one-minus-src-alpha',
                    operation: 'add',
                  },
                },
              },
            ],
          },
          primitive: {
            topology: 'line-list',
          },
        });

        // Generate line vertex data for the wireframe grid
        const vertices: number[] = [];
        const pushVertex = (x: number, y: number, z: number, r: number, g: number, b: number, a: number) => {
          vertices.push(x, y, z, r, g, b, a);
        };

        const NUM_BLOCK_LAYERS = isLowEndDevice ? 6 : 10;
        const BLOCK_DEPTH = 110;

        // Base contour block layers
        for (let layer = NUM_BLOCK_LAYERS; layer >= 1; layer--) {
          const lz = -BLOCK_DEPTH * (layer / NUM_BLOCK_LAYERS) - 15;
          const alpha = 0.12 + (1 - layer / NUM_BLOCK_LAYERS) * 0.22;
          const corners = [
            [-0.5, -0.5],
            [0.5, -0.5],
            [0.5, 0.5],
            [-0.5, 0.5],
          ];
          for (let c = 0; c < 4; c++) {
            const next = (c + 1) % 4;
            pushVertex(corners[c][0], corners[c][1], lz, 1, 1, 1, alpha);
            pushVertex(corners[next][0], corners[next][1], lz, 1, 1, 1, alpha);
          }
        }

        // Top terrain wireframe grid lines
        for (let i = 0; i < COLS; i++) {
          const u = -1 + i * stepU;
          for (let j = 0; j < ROWS; j++) {
            const v = -1 + j * stepV;
            const px = u * 0.5;
            const py = v * 0.5;
            const pz = baseElevations[i * ROWS + j];

            // Horizontal connection to next column
            if (i < COLS - 1) {
              const nextPz = baseElevations[(i + 1) * ROWS + j];
              pushVertex(px, py, pz, 1, 1, 1, 0.85);
              pushVertex((u + stepU) * 0.5, py, nextPz, 1, 1, 1, 0.85);
            }
            // Vertical connection to next row
            if (j < ROWS - 1) {
              const nextPz = baseElevations[i * ROWS + (j + 1)];
              pushVertex(px, py, pz, 1, 1, 1, 0.85);
              pushVertex(px, (v + stepV) * 0.5, nextPz, 1, 1, 1, 0.85);
            }
          }
        }

        const vertexBuffer = device.createBuffer({
          size: vertices.length * 4,
          usage: GPU_BUFFER_USAGE.VERTEX | GPU_BUFFER_USAGE.COPY_DST,
        });
        device.queue.writeBuffer(vertexBuffer, 0, new Float32Array(vertices));

        // Uniform buffer: origin(2), resolution(2), tilt(2), scale(1), time(1)
        const uniformBuffer = device.createBuffer({
          size: 8 * 4,
          usage: GPU_BUFFER_USAGE.UNIFORM | GPU_BUFFER_USAGE.COPY_DST,
        });

        const bindGroup = device.createBindGroup({
          layout: pipeline.getBindGroupLayout(0),
          entries: [{ binding: 0, resource: { buffer: uniformBuffer } }],
        });

        setActiveRenderer('webgpu');

        let lastFrameTime = 0;
        let time = 0;

        const renderWebGPU = (now: number) => {
          if (isDestroyed) return;
          if (!isVisible) {
            animationFrameId = requestAnimationFrame(renderWebGPU);
            return;
          }

          if (now - lastFrameTime < frameInterval && !prefersReducedMotion) {
            animationFrameId = requestAnimationFrame(renderWebGPU);
            return;
          }
          lastFrameTime = now;
          time += 16;

          currentTiltX += (targetTiltX - currentTiltX) * 0.05;
          currentTiltY += (targetTiltY - currentTiltY) * 0.05;

          const rect = currentContainer.getBoundingClientRect();
          const dpr = Math.min(window.devicePixelRatio || 1, isLowEndDevice ? 1.0 : 1.75);
          const width = Math.max(1, Math.floor(rect.width * dpr));
          const height = Math.max(1, Math.floor(rect.height * dpr));

          if (currentCanvas.width !== width || currentCanvas.height !== height) {
            currentCanvas.width = width;
            currentCanvas.height = height;
          }

          const scale = Math.min(width, height) * 0.64;
          const originX = width * 0.52;
          const originY = height * 0.44;

          const uniformData = new Float32Array([
            originX, originY,
            width, height,
            currentTiltX, currentTiltY,
            scale,
            prefersReducedMotion ? 0 : time,
          ]);
          device.queue.writeBuffer(uniformBuffer, 0, uniformData);

          const commandEncoder = device.createCommandEncoder();
          const pass = commandEncoder.beginRenderPass({
            colorAttachments: [
              {
                view: ctx.getCurrentTexture().createView(),
                clearValue: { r: 0, g: 0, b: 0, a: 0 },
                loadOp: 'clear',
                storeOp: 'store',
              },
            ],
          });

          pass.setPipeline(pipeline);
          pass.setBindGroup(0, bindGroup);
          pass.setVertexBuffer(0, vertexBuffer);
          pass.draw(vertices.length / 7);
          pass.end();

          device.queue.submit([commandEncoder.finish()]);

          if (!prefersReducedMotion) {
            animationFrameId = requestAnimationFrame(renderWebGPU);
          }
        };

        animationFrameId = requestAnimationFrame(renderWebGPU);
        return true;
      } catch (err) {
        console.warn('WebGPU init failed, falling back to Canvas 2D:', err);
        return false;
      }
    }

    // -------------------------------------------------------------
    // CANVAS 2D ENGINE (Tier 2 - Low-End Hardware Optimized)
    // -------------------------------------------------------------
    function startCanvas2D() {
      const ctx = currentCanvas.getContext('2d', { alpha: true });
      if (!ctx) return;

      setActiveRenderer('canvas2d');

      let width = 0;
      let height = 0;
      let dpr = 1;

      const resize = () => {
        if (!currentCanvas || !currentContainer) return;
        const rect = currentContainer.getBoundingClientRect();
        // Strict DPR clamping: On low-end hardware, clamp to 1.0 to eliminate fillrate choking
        dpr = Math.min(window.devicePixelRatio || 1, isLowEndDevice ? 1.0 : 1.75);
        width = rect.width;
        height = rect.height;
        currentCanvas.width = Math.floor(width * dpr);
        currentCanvas.height = Math.floor(height * dpr);
        currentCanvas.style.width = `${width}px`;
        currentCanvas.style.height = `${height}px`;
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      };

      resize();
      window.addEventListener('resize', resize, { passive: true });

      const ISO_ANGLE = Math.PI / 6; // 30 degrees
      const COS_A = Math.cos(ISO_ANGLE);
      const SIN_A = Math.sin(ISO_ANGLE);

      const project = (
        x: number,
        y: number,
        z: number,
        originX: number,
        originY: number,
        tiltX: number,
        tiltY: number
      ) => {
        const rx = x * Math.cos(tiltY) - y * Math.sin(tiltY);
        const ry = (x * Math.sin(tiltY) + y * Math.cos(tiltY)) * Math.cos(tiltX) - z * Math.sin(tiltX);
        const rz = z * Math.cos(tiltX) + (x * Math.sin(tiltY) + y * Math.cos(tiltY)) * Math.sin(tiltX);

        return {
          x: originX + (rx - ry) * COS_A,
          y: originY + (rx + ry) * SIN_A - rz,
        };
      };

      let time = 0;
      let lastFrameTime = 0;

      // Reusable point buffer to eliminate Garbage Collection churn
      const sxBuffer = new Float32Array(COLS * ROWS);
      const syBuffer = new Float32Array(COLS * ROWS);
      const zBuffer = new Float32Array(COLS * ROWS);

      const renderCanvas = (now: number) => {
        if (isDestroyed) return;
        if (!isVisible) {
          animationFrameId = 0;
          return;
        }

        // Frame rate throttling for low-end devices (30fps)
        if (now - lastFrameTime < frameInterval && !prefersReducedMotion) {
          animationFrameId = requestAnimationFrame(renderCanvas);
          return;
        }
        lastFrameTime = now;
        time += 16;

        currentTiltX += (targetTiltX - currentTiltX) * 0.05;
        currentTiltY += (targetTiltY - currentTiltY) * 0.05;

        ctx.clearRect(0, 0, width, height);

        const originX = width * 0.52;
        const originY = height * 0.44;
        const scale = Math.min(width, height) * 0.64;

        // 1. Lower contour block layers
        const NUM_BLOCK_LAYERS = isLowEndDevice ? 6 : 11;
        const BLOCK_DEPTH = 110;

        ctx.save();
        for (let layer = NUM_BLOCK_LAYERS; layer >= 1; layer--) {
          const layerZ = -BLOCK_DEPTH * (layer / NUM_BLOCK_LAYERS) - 15;
          const alpha = 0.12 + (1 - layer / NUM_BLOCK_LAYERS) * 0.22;

          ctx.strokeStyle = `rgba(255, 255, 255, ${alpha.toFixed(3)})`;
          ctx.lineWidth = 0.75;

          const p00 = project(-scale * 0.5, -scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);
          const p10 = project(scale * 0.5, -scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);
          const p11 = project(scale * 0.5, scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);
          const p01 = project(-scale * 0.5, scale * 0.5, layerZ, originX, originY, currentTiltX, currentTiltY);

          ctx.beginPath();
          ctx.moveTo(p00.x, p00.y);
          ctx.lineTo(p10.x, p10.y);
          ctx.lineTo(p11.x, p11.y);
          ctx.lineTo(p01.x, p01.y);
          ctx.closePath();
          ctx.stroke();

          // Dotted matrix points on top block layer
          if (layer === 1) {
            ctx.fillStyle = 'rgba(255, 255, 255, 0.45)';
            const stride = isLowEndDevice ? 3 : 2;
            for (let i = 0; i < COLS; i += stride) {
              for (let j = 0; j < ROWS; j += stride) {
                const u = -1 + i * stepU;
                const v = -1 + j * stepV;
                const pt = project(u * (scale * 0.5), v * (scale * 0.5), layerZ, originX, originY, currentTiltX, currentTiltY);
                ctx.beginPath();
                ctx.arc(pt.x, pt.y, 0.85, 0, Math.PI * 2);
                ctx.fill();
              }
            }
          }
        }

        // 2. Middle Floating Dot Plane
        const DOT_PLANE_Z = 12;
        ctx.fillStyle = 'rgba(255, 255, 255, 0.55)';
        const dotStride = isLowEndDevice ? 2 : 1;
        for (let i = 0; i < COLS; i += dotStride) {
          for (let j = 0; j < ROWS; j += dotStride) {
            const u = -1 + i * stepU;
            const v = -1 + j * stepV;
            const pt = project(u * (scale * 0.5), v * (scale * 0.5), DOT_PLANE_Z, originX, originY, currentTiltX, currentTiltY);
            ctx.beginPath();
            ctx.arc(pt.x, pt.y, 0.75, 0, Math.PI * 2);
            ctx.fill();
          }
        }

        // 3. Top Wireframe Surface Mesh (using precalculated elevation table)
        for (let i = 0; i < COLS; i++) {
          const u = -1 + i * stepU;
          for (let j = 0; j < ROWS; j++) {
            const v = -1 + j * stepV;
            const px = u * (scale * 0.5);
            const py = v * (scale * 0.5);
            const baseZ = baseElevations[i * ROWS + j];
            const wave = prefersReducedMotion
              ? 0
              : Math.sin(u * 5 + time * 0.0015) * Math.cos(v * 4 + time * 0.001) * 7;
            const pz = baseZ + wave;

            const pt = project(px, py, pz, originX, originY, currentTiltX, currentTiltY);
            const idx = i * ROWS + j;
            sxBuffer[idx] = pt.x;
            syBuffer[idx] = pt.y;
            zBuffer[idx] = pz;
          }
        }

        // Draw wireframe lines along columns
        ctx.lineWidth = 1.0;
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.85)';

        for (let i = 0; i < COLS; i++) {
          ctx.beginPath();
          for (let j = 0; j < ROWS; j++) {
            const idx = i * ROWS + j;
            if (j === 0) ctx.moveTo(sxBuffer[idx], syBuffer[idx]);
            else ctx.lineTo(sxBuffer[idx], syBuffer[idx]);
          }
          ctx.stroke();
        }

        // Draw wireframe lines along rows
        for (let j = 0; j < ROWS; j++) {
          ctx.beginPath();
          for (let i = 0; i < COLS; i++) {
            const idx = i * ROWS + j;
            if (i === 0) ctx.moveTo(sxBuffer[idx], syBuffer[idx]);
            else ctx.lineTo(sxBuffer[idx], syBuffer[idx]);
          }
          ctx.stroke();
        }

        // High peak vertical drop lines
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
        ctx.lineWidth = 0.5;
        const dropStride = isLowEndDevice ? 3 : 2;
        for (let i = 0; i < COLS; i += dropStride) {
          for (let j = 0; j < ROWS; j += dropStride) {
            const idx = i * ROWS + j;
            if (zBuffer[idx] > 35) {
              const u = -1 + i * stepU;
              const v = -1 + j * stepV;
              const basePt = project(u * (scale * 0.5), v * (scale * 0.5), DOT_PLANE_Z, originX, originY, currentTiltX, currentTiltY);
              ctx.beginPath();
              ctx.moveTo(sxBuffer[idx], syBuffer[idx]);
              ctx.lineTo(basePt.x, basePt.y);
              ctx.stroke();
            }
          }
        }

        ctx.restore();

        if (!prefersReducedMotion) {
          animationFrameId = requestAnimationFrame(renderCanvas);
        } else {
          animationFrameId = 0;
        }
      };

      resumeRender = () => {
        if (!animationFrameId && !isDestroyed && isVisible && !prefersReducedMotion) {
          animationFrameId = requestAnimationFrame(renderCanvas);
        }
      };

      animationFrameId = requestAnimationFrame(renderCanvas);
    }

    // -------------------------------------------------------------
    // INTERSECTION OBSERVER & VISIBILITY GUARDS (Zero Cost Offscreen)
    // -------------------------------------------------------------
    const observer = new IntersectionObserver(
      ([entry]) => {
        const wasVisible = isVisible;
        isVisible = entry.isIntersecting;
        if (isVisible && !wasVisible && resumeRender) {
          resumeRender();
        }
      },
      { threshold: 0.05 }
    );
    observer.observe(currentContainer);

    const handleVisibilityChange = () => {
      const wasVisible = isVisible;
      isVisible = document.visibilityState === 'visible';
      if (isVisible && !wasVisible && resumeRender) {
        resumeRender();
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);

    // Boot: Start rock-solid Canvas 2D engine directly.
    // This guarantees full 3D isometric mountain mesh, contour layers, wave animation,
    // and dot planes render flawlessly across Safari, Chrome, Firefox, and mobile.
    startCanvas2D();

    return () => {
      isDestroyed = true;
      observer.disconnect();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('mousemove', handleMouseMove);
      if (animationFrameId) {
        cancelAnimationFrame(animationFrameId);
      }
    };
  }, [prefersReducedMotion, isMobile]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full min-h-[360px] sm:min-h-[440px] lg:min-h-[520px] relative flex items-center justify-center overflow-hidden"
    >
      <canvas
        ref={canvasRef}
        className="block w-full h-full cursor-grab active:cursor-grabbing"
      />
      {/* Subtle tactical hardware badge in footer corner */}
      <div className="absolute bottom-2 right-2 text-[9px] font-mono tracking-widest uppercase text-white/20 select-none pointer-events-none">
        {activeRenderer === 'webgpu' ? 'GPU: WebGPU accelerated' : 'Canvas2D: Battery optimized'}
      </div>
    </div>
  );
}
