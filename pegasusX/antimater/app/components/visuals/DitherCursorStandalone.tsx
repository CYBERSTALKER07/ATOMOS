'use client';

import React, { useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

export interface DitherCursorStandaloneProps {
  /** Size of the dither pattern pixels (default: 4) */
  ditherSize?: number;
  /** Radius of cursor influence in normalized UV space (default: 0.1) */
  radius?: number;
  /** Exponent curve for falloff steepness (default: 2.0) */
  exponent?: number;
  /** Trail decay speed per frame (default: 0.01) */
  decay?: number;
  /** Intensity multiplier of the brush (default: 0.5) */
  intensity?: number;
  /** Color of the dithered dots in hex (default: "#000000") */
  color?: string;
  /** Container CSS classes */
  className?: string;
}

const SIM_VS = `#version 300 es
in vec2 aPosition;
out vec2 vUv;
void main() {
  vUv = aPosition * 0.5 + 0.5;
  gl_Position = vec4(aPosition, 0.0, 1.0);
}`;

const SIM_FS = `#version 300 es
precision highp float;
in vec2 vUv;
uniform float uTime;
uniform vec2 uMouse;
uniform sampler2D uPreviousState;
uniform vec2 uResolution;
uniform float uRadius;
uniform float uDecay;
uniform float uIntensity;
uniform float uSpeed;

out vec4 fragColor;

vec3 mod289(vec3 x) { return x - floor(x * (1.0 / 289.0)) * 289.0; }
vec2 mod289(vec2 x) { return x - floor(x * (1.0 / 289.0)) * 289.0; }
vec3 permute(vec3 x) { return mod289(((x*34.0)+1.0)*x); }

float snoise(vec2 v) {
  const vec4 C = vec4(0.211324865405187, 0.366025403784439, -0.577350269189626, 0.024390243902439);
  vec2 i  = floor(v + dot(v, C.yy));
  vec2 x0 = v - i + dot(i, C.xx);
  vec2 i1 = (x0.x > x0.y) ? vec2(1.0, 0.0) : vec2(0.0, 1.0);
  vec4 x12 = x0.xyxy + C.xxzz;
  x12.xy -= i1;
  i = mod289(i);
  vec3 p = permute(permute(i.y + vec3(0.0, i1.y, 1.0)) + i.x + vec3(0.0, i1.x, 1.0));
  vec3 m = max(0.5 - vec3(dot(x0, x0), dot(x12.xy, x12.xy), dot(x12.zw, x12.zw)), 0.0);
  m = m * m;
  m = m * m;
  vec3 x = 2.0 * fract(p * C.www) - 1.0;
  vec3 h = abs(x) - 0.5;
  vec3 ox = floor(x + 0.5);
  vec3 a0 = x - ox;
  m *= 1.79284291400159 - 0.85373472095314 * (a0 * a0 + h * h);
  vec3 g;
  g.x  = a0.x  * x0.x  + h.x  * x0.y;
  g.yz = a0.yz * x12.xz + h.yz * x12.yw;
  return 130.0 * dot(m, g);
}

vec2 curl(vec2 p) {
  float eps = 0.1;
  float n1 = snoise(p + vec2(0.0, eps));
  float n2 = snoise(p - vec2(0.0, eps));
  float n3 = snoise(p + vec2(eps, 0.0));
  float n4 = snoise(p - vec2(eps, 0.0));
  float x = (n1 - n2) / (2.0 * eps);
  float y = (n3 - n4) / (2.0 * eps);
  return vec2(y, -x);
}

void main() {
  vec2 uv = vUv;
  vec2 texel = 1.0 / uResolution;

  vec2 noiseUV = uv * 0.5;
  vec2 velocity = curl(noiseUV + uTime * 0.1);

  float advectionStrength = 0.001;
  vec2 advectedUV = uv - velocity * advectionStrength;

  float prev   = texture(uPreviousState, advectedUV).r;
  float top    = texture(uPreviousState, advectedUV + vec2(0.0, texel.y)).r;
  float bottom = texture(uPreviousState, advectedUV - vec2(0.0, texel.y)).r;
  float left   = texture(uPreviousState, advectedUV - vec2(texel.x, 0.0)).r;
  float right  = texture(uPreviousState, advectedUV + vec2(texel.x, 0.0)).r;

  float diffused = (prev + top + bottom + left + right) / 5.0;

  float aspect = uResolution.x / uResolution.y;
  vec2 aspectCorrection = vec2(aspect, 1.0);
  vec2 mouseUV = uMouse;

  float dist = length((uv - mouseUV) * aspectCorrection);
  float brush = exp(-pow(dist / uRadius, 2.0));
  float speedFactor = smoothstep(0.0, 0.01, uSpeed);

  brush *= uIntensity * speedFactor * 0.5;

  float value = min(0.95, diffused + brush);
  value -= uDecay;

  fragColor = vec4(vec3(max(0.0, value)), 1.0);
}`;

const DITHER_VS = `#version 300 es
in vec2 aPosition;
out vec2 vUv;
void main() {
  vUv = aPosition * 0.5 + 0.5;
  gl_Position = vec4(aPosition, 0.0, 1.0);
}`;

const DITHER_FS = `#version 300 es
precision highp float;
in vec2 vUv;
uniform sampler2D uSimulationState;
uniform float uDitherSize;
uniform float uExponent;
uniform vec3 uColor;

out vec4 fragColor;

const float bayer8[64] = float[64](
  0.0 / 64.0, 32.0 / 64.0,  8.0 / 64.0, 40.0 / 64.0,  2.0 / 64.0, 34.0 / 64.0, 10.0 / 64.0, 42.0 / 64.0,
  48.0 / 64.0, 16.0 / 64.0, 56.0 / 64.0, 24.0 / 64.0, 50.0 / 64.0, 18.0 / 64.0, 58.0 / 64.0, 26.0 / 64.0,
  12.0 / 64.0, 44.0 / 64.0,  4.0 / 64.0, 36.0 / 64.0, 14.0 / 64.0, 46.0 / 64.0,  6.0 / 64.0, 38.0 / 64.0,
  60.0 / 64.0, 28.0 / 64.0, 52.0 / 64.0, 20.0 / 64.0, 62.0 / 64.0, 30.0 / 64.0, 54.0 / 64.0, 22.0 / 64.0,
  3.0 / 64.0, 35.0 / 64.0, 11.0 / 64.0, 43.0 / 64.0,  1.0 / 64.0, 33.0 / 64.0,  9.0 / 64.0, 41.0 / 64.0,
  51.0 / 64.0, 19.0 / 64.0, 59.0 / 64.0, 27.0 / 64.0, 49.0 / 64.0, 17.0 / 64.0, 57.0 / 64.0, 25.0 / 64.0,
  15.0 / 64.0, 47.0 / 64.0,  7.0 / 64.0, 39.0 / 64.0, 13.0 / 64.0, 45.0 / 64.0,  5.0 / 64.0, 37.0 / 64.0,
  63.0 / 64.0, 31.0 / 64.0, 55.0 / 64.0, 23.0 / 64.0, 61.0 / 64.0, 29.0 / 64.0, 53.0 / 64.0, 21.0 / 64.0
);

void main() {
  vec2 uv = vUv;
  float signal = texture(uSimulationState, uv).r;
  signal = pow(signal, uExponent);

  ivec2 cell = ivec2(mod(gl_FragCoord.xy / uDitherSize, 8.0));
  float threshold = bayer8[cell.y * 8 + cell.x];
  float mask = step(threshold, signal);

  if (signal < 0.01) mask = 0.0;
  if (mask < 0.5) discard;

  fragColor = vec4(uColor, 1.0);
}`;

function hexToRgb(hex: string): [number, number, number] {
  const clean = hex.replace('#', '');
  return [
    parseInt(clean.substring(0, 2), 16) / 255,
    parseInt(clean.substring(2, 4), 16) / 255,
    parseInt(clean.substring(4, 6), 16) / 255
  ];
}

export default function DitherCursorStandalone({
  ditherSize = 4,
  radius = 0.1,
  exponent = 2.0,
  decay = 0.01,
  intensity = 0.5,
  color = '#000000',
  className
}: DitherCursorStandaloneProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    const container = containerRef.current;
    if (!canvas || !container) return;

    const gl = canvas.getContext('webgl2', { alpha: true, antialias: false });
    if (!gl) return;

    const createShader = (type: number, src: string) => {
      const sh = gl.createShader(type);
      if (!sh) return null;
      gl.shaderSource(sh, src);
      gl.compileShader(sh);
      return sh;
    };

    const simProg = gl.createProgram();
    const simVs = createShader(gl.VERTEX_SHADER, SIM_VS);
    const simFs = createShader(gl.FRAGMENT_SHADER, SIM_FS);
    if (!simProg || !simVs || !simFs) return;
    gl.attachShader(simProg, simVs);
    gl.attachShader(simProg, simFs);
    gl.linkProgram(simProg);

    const ditherProg = gl.createProgram();
    const ditherVs = createShader(gl.VERTEX_SHADER, DITHER_VS);
    const ditherFs = createShader(gl.FRAGMENT_SHADER, DITHER_FS);
    if (!ditherProg || !ditherVs || !ditherFs) return;
    gl.attachShader(ditherProg, ditherVs);
    gl.attachShader(ditherProg, ditherFs);
    gl.linkProgram(ditherProg);

    // Quad geometry
    const quad = new Float32Array([-1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1]);
    const vao = gl.createVertexArray();
    gl.bindVertexArray(vao);
    const buf = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, quad, gl.STATIC_DRAW);
    gl.enableVertexAttribArray(0);
    gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0);

    // Ping-pong FBOs
    let width = container.clientWidth || 300;
    let height = container.clientHeight || 300;

    const createFBO = (w: number, h: number) => {
      const tex = gl.createTexture();
      gl.bindTexture(gl.TEXTURE_2D, tex);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, w, h, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
      gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);

      const fb = gl.createFramebuffer();
      gl.bindFramebuffer(gl.FRAMEBUFFER, fb);
      gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, tex, 0);
      return { tex, fb };
    };

    let fboA = createFBO(width, height);
    let fboB = createFBO(width, height);

    let mouseX = 0;
    let mouseY = 0;
    let prevMouseX = 0;
    let prevMouseY = 0;
    let speed = 0;
    let frame = 0;

    const onMouseMove = (e: MouseEvent) => {
      const rect = container.getBoundingClientRect();
      mouseX = (e.clientX - rect.left) / rect.width;
      mouseY = 1.0 - (e.clientY - rect.top) / rect.height;
    };
    window.addEventListener('mousemove', onMouseMove, { passive: true });

    const resize = () => {
      width = container.clientWidth || 300;
      height = container.clientHeight || 300;
      canvas.width = width;
      canvas.height = height;
      fboA = createFBO(width, height);
      fboB = createFBO(width, height);
    };
    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(container);

    let raf: number;
    let startTime = performance.now();

    const render = () => {
      raf = requestAnimationFrame(render);
      const time = (performance.now() - startTime) * 0.001;

      const dist = Math.hypot(mouseX - prevMouseX, mouseY - prevMouseY);
      speed += (dist - speed) * 0.1;
      prevMouseX = mouseX;
      prevMouseY = mouseY;

      const currentFBO = frame % 2 === 0 ? fboA : fboB;
      const prevFBO = frame % 2 === 0 ? fboB : fboA;

      // PASS 1: Simulation
      gl.bindFramebuffer(gl.FRAMEBUFFER, currentFBO.fb);
      gl.viewport(0, 0, width, height);
      gl.useProgram(simProg);

      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, prevFBO.tex);
      gl.uniform1i(gl.getUniformLocation(simProg, 'uPreviousState'), 0);
      gl.uniform1f(gl.getUniformLocation(simProg, 'uTime'), time);
      gl.uniform2f(gl.getUniformLocation(simProg, 'uMouse'), mouseX, mouseY);
      gl.uniform2f(gl.getUniformLocation(simProg, 'uResolution'), width, height);
      gl.uniform1f(gl.getUniformLocation(simProg, 'uRadius'), radius);
      gl.uniform1f(gl.getUniformLocation(simProg, 'uDecay'), decay);
      gl.uniform1f(gl.getUniformLocation(simProg, 'uIntensity'), intensity);
      gl.uniform1f(gl.getUniformLocation(simProg, 'uSpeed'), speed);

      gl.bindVertexArray(vao);
      gl.drawArrays(gl.TRIANGLES, 0, 6);

      // PASS 2: Dither to Screen
      gl.bindFramebuffer(gl.FRAMEBUFFER, null);
      gl.viewport(0, 0, width, height);
      gl.clearColor(0, 0, 0, 0);
      gl.clear(gl.COLOR_BUFFER_BIT);

      gl.useProgram(ditherProg);
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, currentFBO.tex);
      gl.uniform1i(gl.getUniformLocation(ditherProg, 'uSimulationState'), 0);
      gl.uniform1f(gl.getUniformLocation(ditherProg, 'uDitherSize'), ditherSize);
      gl.uniform1f(gl.getUniformLocation(ditherProg, 'uExponent'), exponent);

      const rgb = hexToRgb(color);
      gl.uniform3f(gl.getUniformLocation(ditherProg, 'uColor'), rgb[0], rgb[1], rgb[2]);

      gl.drawArrays(gl.TRIANGLES, 0, 6);

      frame++;
    };

    raf = requestAnimationFrame(render);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('mousemove', onMouseMove);
      ro.disconnect();
      gl.deleteBuffer(buf);
      gl.deleteVertexArray(vao);
      gl.deleteProgram(simProg);
      gl.deleteProgram(ditherProg);
    };
  }, [ditherSize, radius, exponent, decay, intensity, color]);

  return (
    <div
      ref={containerRef}
      className={cn('absolute inset-0 w-full h-full pointer-events-none z-10 overflow-hidden', className)}
    >
      <canvas ref={canvasRef} className="w-full h-full block" />
    </div>
  );
}
