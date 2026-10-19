'use client';

import React, { useEffect, useRef } from 'react';

// Exact Bayer 8x8 matrix from Dither.tsx
const BAYER_8X8_RAW = [
   0, 48, 12, 60,  3, 51, 15, 63,
  32, 16, 44, 28, 35, 19, 47, 31,
   8, 56,  4, 52, 11, 59,  7, 55,
  40, 24, 36, 20, 43, 27, 39, 23,
   2, 50, 14, 62,  1, 49, 13, 61,
  34, 18, 46, 30, 33, 17, 45, 29,
  10, 58,  6, 54,  9, 57,  5, 53,
  42, 26, 38, 22, 41, 25, 37, 21,
];

const vertexShaderSource = `
attribute vec2 position;
varying vec2 vUv;
void main() {
  vUv = position * 0.5 + 0.5;
  gl_Position = vec4(position, 0.0, 1.0);
}
`;

const fragmentShaderSource = `
precision highp float;

uniform sampler2D uTexture;
uniform sampler2D uBayerTexture;
uniform vec2 uResolution;
uniform vec2 uImgSize;
uniform float uTime;
uniform float uProgress;
uniform vec2 uMouse;
uniform float uMouseActive;
uniform float uPixelSize;
uniform float uColorNum;
uniform float uRippleTime;
uniform vec2 uRippleCenter;

varying vec2 vUv;

vec3 applyBayerDither(vec2 screenCoord, vec3 color) {
  // Sample exact 8x8 Bayer matrix threshold from repeated nearest texture
  float bayerVal = texture2D(uBayerTexture, screenCoord / (uPixelSize * 8.0)).r;
  float threshold = bayerVal - 0.25;
  float stepVal = 1.0 / (uColorNum - 1.0);
  color += threshold * stepVal;
  float luminance = dot(color, vec3(0.2126, 0.7152, 0.0722));
  float bias = mix(0.2, 0.0, smoothstep(0.45, 0.8, luminance));
  color = clamp(color - bias, 0.0, 1.0);
  return floor(color * (uColorNum - 1.0) + 0.5) / (uColorNum - 1.0);
}

void main() {
  vec2 screenPixel = vUv * uResolution;
  
  // Responsive layout: calculate image placement & scale
  float isMobile = step(uResolution.x, 768.0);
  float isTablet = step(uResolution.x, 1024.0) * (1.0 - isMobile);
  
  float scale = (uResolution.y / uImgSize.y) * mix(mix(1.05, 1.0, isTablet), 0.95, isMobile);
  float imgW = uImgSize.x * scale;
  float imgH = uImgSize.y * scale;
  
  float offsetX = mix(mix(-10.0, -15.0, isTablet), -max(20.0, imgW * 0.18), isMobile);
  float offsetY = (uResolution.y - imgH) * 0.5;
  
  // Calculate relative image coordinates [0..1]
  float texX = (screenPixel.x - offsetX) / imgW;
  float texY = (screenPixel.y - offsetY) / imgH;
  
  if (texX < 0.0 || texX > 1.0 || texY < 0.0 || texY > 1.0) {
    gl_FragColor = vec4(0.0);
    return;
  }
  
  // Invert Y for image texture orientation
  vec2 imgUv = vec2(texX, 1.0 - texY);
  
  // Starter cybernetic scanline assembly transition
  float revealSweep = smoothstep(0.0, 0.2, uProgress * 1.3 - texX);
  if (revealSweep < 0.001) {
    gl_FragColor = vec4(0.0);
    return;
  }
  
  // Interactive pointer wave ripple (displacement)
  if (uMouseActive > 0.01) {
    vec2 mouseImgUv = (uMouse - vec2(offsetX, offsetY)) / vec2(imgW, imgH);
    mouseImgUv.y = 1.0 - mouseImgUv.y;
    float dist = length(imgUv - mouseImgUv);
    float wave = sin(dist * 40.0 - uTime * 6.0) * exp(-dist * 12.0) * 0.012 * uMouseActive;
    vec2 dir = normalize(imgUv - mouseImgUv + vec2(0.0001));
    imgUv += dir * wave;
  }
  
  // Click shockwave ripple
  if (uRippleTime > 0.0 && uRippleTime < 1.5) {
    vec2 ripUv = (uRippleCenter - vec2(offsetX, offsetY)) / vec2(imgW, imgH);
    ripUv.y = 1.0 - ripUv.y;
    float ripDist = length(imgUv - ripUv);
    float ripWave = sin((ripDist - uRippleTime * 0.8) * 30.0) * exp(-abs(ripDist - uRippleTime * 0.8) * 10.0) * 0.015;
    imgUv += normalize(imgUv - ripUv + vec2(0.0001)) * ripWave;
  }
  
  // Subtle ambient idle wave shimmer (matching Dither.tsx wave effect)
  float idleShimmer = sin(imgUv.x * 10.0 + imgUv.y * 7.0 + uTime * 1.4) * 0.04;
  
  vec4 rawTex = texture2D(uTexture, clamp(imgUv, 0.0, 1.0));
  
  // Discard empty pitch black background
  float brightness = rawTex.r + rawTex.g + rawTex.b;
  if (brightness < 0.07 || rawTex.a < 0.05) {
    gl_FragColor = vec4(0.0);
    return;
  }
  
  // Vibrant purple/lavender tonal scaling with idle shimmer
  vec3 baseColor = rawTex.rgb * (1.28 + idleShimmer) * revealSweep;
  
  // Cursor proximity glow (highlights face points under mouse)
  if (uMouseActive > 0.01) {
    float mousePxDist = length(screenPixel - uMouse);
    float glow = smoothstep(140.0, 0.0, mousePxDist) * 0.3 * uMouseActive;
    baseColor += vec3(0.75, 0.52, 0.98) * glow;
  }
  
  // Apply retro Bayer 8x8 matrix ordered dither
  vec3 dithered = applyBayerDither(screenPixel, baseColor);
  
  gl_FragColor = vec4(dithered, 1.0);
}
`;

export default function DitheredFaceCanvas() {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    // Prefer WebGL2 if available for optimal performance, fallback to WebGL
    const gl = (canvas.getContext('webgl2', { alpha: true, antialias: false }) ||
      canvas.getContext('webgl', { alpha: true, antialias: false })) as WebGLRenderingContext | WebGL2RenderingContext | null;

    if (!gl) {
      console.warn('WebGL not supported for DitheredFaceCanvas');
      return;
    }

    // Helper to compile shader
    const compileShader = (type: number, source: string) => {
      const shader = gl.createShader(type);
      if (!shader) return null;
      gl.shaderSource(shader, source);
      gl.compileShader(shader);
      if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
        console.error('Shader compilation error:', gl.getShaderInfoLog(shader));
        gl.deleteShader(shader);
        return null;
      }
      return shader;
    };

    const vertShader = compileShader(gl.VERTEX_SHADER, vertexShaderSource);
    const fragShader = compileShader(gl.FRAGMENT_SHADER, fragmentShaderSource);
    if (!vertShader || !fragShader) return;

    const program = gl.createProgram();
    if (!program) return;
    gl.attachShader(program, vertShader);
    gl.attachShader(program, fragShader);
    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      console.error('Program link error:', gl.getProgramInfoLog(program));
      return;
    }
    gl.useProgram(program);

    // Quad geometry (2 triangles covering clip space)
    const positionBuffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, positionBuffer);
    const positions = new Float32Array([
      -1, -1,
       1, -1,
      -1,  1,
      -1,  1,
       1, -1,
       1,  1,
    ]);
    gl.bufferData(gl.ARRAY_BUFFER, positions, gl.STATIC_DRAW);

    const posAttr = gl.getAttribLocation(program, 'position');
    gl.enableVertexAttribArray(posAttr);
    gl.vertexAttribPointer(posAttr, 2, gl.FLOAT, false, 0, 0);

    // Uniform locations
    const uTextureLoc = gl.getUniformLocation(program, 'uTexture');
    const uBayerTextureLoc = gl.getUniformLocation(program, 'uBayerTexture');
    const uResolutionLoc = gl.getUniformLocation(program, 'uResolution');
    const uImgSizeLoc = gl.getUniformLocation(program, 'uImgSize');
    const uTimeLoc = gl.getUniformLocation(program, 'uTime');
    const uProgressLoc = gl.getUniformLocation(program, 'uProgress');
    const uMouseLoc = gl.getUniformLocation(program, 'uMouse');
    const uMouseActiveLoc = gl.getUniformLocation(program, 'uMouseActive');
    const uPixelSizeLoc = gl.getUniformLocation(program, 'uPixelSize');
    const uColorNumLoc = gl.getUniformLocation(program, 'uColorNum');
    const uRippleTimeLoc = gl.getUniformLocation(program, 'uRippleTime');
    const uRippleCenterLoc = gl.getUniformLocation(program, 'uRippleCenter');

    // Create and upload 8x8 Bayer matrix texture
    const bayerTexture = gl.createTexture();
    gl.activeTexture(gl.TEXTURE1);
    gl.bindTexture(gl.TEXTURE_2D, bayerTexture);
    const bayerBytes = new Uint8Array(64 * 4);
    for (let i = 0; i < 64; i++) {
      const val = Math.round((BAYER_8X8_RAW[i] / 64.0) * 255);
      bayerBytes[i * 4 + 0] = val;
      bayerBytes[i * 4 + 1] = val;
      bayerBytes[i * 4 + 2] = val;
      bayerBytes[i * 4 + 3] = 255;
    }
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 8, 8, 0, gl.RGBA, gl.UNSIGNED_BYTE, bayerBytes);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);

    // Image texture setup
    const texture = gl.createTexture();
    gl.activeTexture(gl.TEXTURE0);
    gl.bindTexture(gl.TEXTURE_2D, texture);
    gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, new Uint8Array([0, 0, 0, 0]));
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);

    let isTextureLoaded = false;
    const img = new Image();
    img.src = '/images/early-access-mesh.png';
    img.onload = () => {
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, texture);
      gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, false);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
      isTextureLoaded = true;
    };

    // State variables
    let animationFrameId: number | null = null;
    const startTime = performance.now();
    let isVisible = false;
    let width = 0;
    let height = 0;
    let dpr = 1;

    // Entrance scanline animation
    let hasTriggeredEntrance = false;
    let entranceStart = 0;
    const entranceDuration = 1200; // ms

    // Interactive pointer state
    const pointer = {
      targetX: -9999,
      targetY: -9999,
      smoothX: -9999,
      smoothY: -9999,
      active: 0,
      targetActive: 0,
    };

    // Shockwave ripple
    let rippleStart = -9999;
    const rippleCenter = { x: 0, y: 0 };

    const handleResize = () => {
      const rect = canvas.getBoundingClientRect();
      width = rect.width;
      height = rect.height;
      dpr = Math.min(window.devicePixelRatio || 1, 2);

      const targetW = Math.floor(width * dpr);
      const targetH = Math.floor(height * dpr);

      if (canvas.width !== targetW || canvas.height !== targetH) {
        canvas.width = targetW;
        canvas.height = targetH;
      }
    };

    const section = canvas.closest('section') || canvas.parentElement;

    const onPointerMove = (e: MouseEvent | TouchEvent) => {
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      const clientX = 'touches' in e ? e.touches[0].clientX : (e as MouseEvent).clientX;
      const clientY = 'touches' in e ? e.touches[0].clientY : (e as MouseEvent).clientY;

      pointer.targetX = (clientX - rect.left) * dpr;
      pointer.targetY = (rect.bottom - clientY) * dpr;
      pointer.targetActive = 1.0;
    };

    const onPointerLeave = () => {
      pointer.targetActive = 0.0;
    };

    const onPointerDown = (e: MouseEvent | TouchEvent) => {
      if (!canvas) return;
      const rect = canvas.getBoundingClientRect();
      const clientX = 'touches' in e ? e.touches[0].clientX : (e as MouseEvent).clientX;
      const clientY = 'touches' in e ? e.touches[0].clientY : (e as MouseEvent).clientY;

      rippleCenter.x = (clientX - rect.left) * dpr;
      rippleCenter.y = (rect.bottom - clientY) * dpr;
      rippleStart = performance.now();
    };

    if (section) {
      section.addEventListener('mousemove', onPointerMove, { passive: true });
      section.addEventListener('mouseleave', onPointerLeave, { passive: true });
      section.addEventListener('mousedown', onPointerDown, { passive: true });
      section.addEventListener('touchmove', onPointerMove, { passive: true });
      section.addEventListener('touchstart', onPointerDown, { passive: true });
    }

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          isVisible = entry.isIntersecting;
          if (isVisible && !hasTriggeredEntrance) {
            hasTriggeredEntrance = true;
            entranceStart = performance.now();
          }
        });
      },
      { threshold: 0.1 }
    );
    observer.observe(canvas);

    // Initial sizing
    handleResize();
    const resizeObserver = new ResizeObserver(() => handleResize());
    resizeObserver.observe(canvas);

    // Render loop
    const render = (now: number) => {
      animationFrameId = requestAnimationFrame(render);
      if (!isVisible || !isTextureLoaded) return;

      const elapsed = (now - startTime) * 0.001;

      // Smooth pointer lerp
      pointer.smoothX += (pointer.targetX - pointer.smoothX) * 0.1;
      pointer.smoothY += (pointer.targetY - pointer.smoothY) * 0.1;
      pointer.active += (pointer.targetActive - pointer.active) * 0.08;

      // Entrance transition progress (0..1)
      let progress = 1.0;
      if (hasTriggeredEntrance) {
        const p = (now - entranceStart) / entranceDuration;
        progress = Math.min(1.0, Math.max(0.0, p));
      }

      // Ripple shockwave time
      let rippleElapsed = -1.0;
      if (rippleStart > 0) {
        rippleElapsed = (now - rippleStart) * 0.001;
      }

      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.clearColor(0.0, 0.0, 0.0, 0.0);
      gl.clear(gl.COLOR_BUFFER_BIT);

      gl.useProgram(program);

      // Texture unit 0: Face image
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, texture);
      gl.uniform1i(uTextureLoc, 0);

      // Texture unit 1: 8x8 Bayer matrix
      gl.activeTexture(gl.TEXTURE1);
      gl.bindTexture(gl.TEXTURE_2D, bayerTexture);
      gl.uniform1i(uBayerTextureLoc, 1);

      gl.uniform2f(uResolutionLoc, canvas.width, canvas.height);
      gl.uniform2f(uImgSizeLoc, 1024.0, 612.0);
      gl.uniform1f(uTimeLoc, elapsed);
      gl.uniform1f(uProgressLoc, progress);
      gl.uniform2f(uMouseLoc, pointer.smoothX, pointer.smoothY);
      gl.uniform1f(uMouseActiveLoc, pointer.active);
      gl.uniform1f(uPixelSizeLoc, 2.0 * dpr);
      gl.uniform1f(uColorNumLoc, 4.0);
      gl.uniform1f(uRippleTimeLoc, rippleElapsed);
      gl.uniform2f(uRippleCenterLoc, rippleCenter.x, rippleCenter.y);

      gl.drawArrays(gl.TRIANGLES, 0, 6);
    };

    animationFrameId = requestAnimationFrame(render);

    return () => {
      if (animationFrameId !== null) cancelAnimationFrame(animationFrameId);
      observer.disconnect();
      resizeObserver.disconnect();
      if (section) {
        section.removeEventListener('mousemove', onPointerMove);
        section.removeEventListener('mouseleave', onPointerLeave);
        section.removeEventListener('mousedown', onPointerDown);
        section.removeEventListener('touchmove', onPointerMove);
        section.removeEventListener('touchstart', onPointerDown);
      }
      gl.deleteProgram(program);
      gl.deleteShader(vertShader);
      gl.deleteShader(fragShader);
      gl.deleteTexture(texture);
      gl.deleteTexture(bayerTexture);
      gl.deleteBuffer(positionBuffer);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="w-full h-full block select-none pointer-events-none"
      style={{
        imageRendering: 'pixelated',
      }}
      aria-hidden="true"
    />
  );
}
