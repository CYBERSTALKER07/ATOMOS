'use client';

import React, { useRef, useEffect } from 'react';
import {
  WebGLRenderer,
  Scene,
  OrthographicCamera,
  ShaderMaterial,
  PlaneGeometry,
  Mesh,
  Color,
  Vector2
} from 'three';
import { cn } from '@/lib/utils';

export interface DitherWaveProps {
  width?: number | string;
  height?: number | string;
  speed?: number;
  intensity?: number;
  scale?: number;
  downScale?: number;
  primaryColor?: string;
  secondaryColor?: string;
  tertiaryColor?: string;
  opacity?: number;
  quality?: 'low' | 'medium' | 'high';
  maxFPS?: number;
  pauseWhenOffscreen?: boolean;
  className?: string;
  children?: React.ReactNode;
}

const vertexShader = /* glsl */ `
  void main() {
    gl_Position = vec4(position, 1.0);
  }
`;

const fragmentShader = /* glsl */ `
  #define COLOR_COUNT 3

  uniform float iTime;
  uniform vec2 iResolution;
  uniform float uSpeed;
  uniform float uIntensity;
  uniform float uScale;
  uniform float uDownScale;
  uniform float uOpacity;

  uniform vec3 uColor1;
  uniform vec3 uColor2;
  uniform vec3 uColor3;

  vec3 colors[COLOR_COUNT];

  void setupColorPalette() {
    colors[0] = uColor1;
    colors[1] = uColor2;
    colors[2] = uColor3;
  }

  float Bayer2(vec2 a) {
    a = floor(a);
    return fract(a.x / 2.0 + a.y * a.y * 0.75);
  }

  #define Bayer4(a)   (Bayer2(0.5 * (a)) * 0.25 + Bayer2(a))
  #define Bayer8(a)   (Bayer4(0.5 * (a)) * 0.25 + Bayer2(a))
  #define Bayer16(a)  (Bayer8(0.5 * (a)) * 0.25 + Bayer2(a))
  #define Bayer32(a)  (Bayer16(0.5 * (a)) * 0.25 + Bayer2(a))
  #define Bayer64(a)  (Bayer32(0.5 * (a)) * 0.25 + Bayer2(a))

  vec3 applyDitheredColor(float value, vec2 pixelCoord) {
    float paletteIndex = clamp(value, 0.0, 1.0) * float(COLOR_COUNT - 1);

    vec3 colorA = vec3(0.0);
    vec3 colorB = vec3(0.0);

    for (int i = 0; i < COLOR_COUNT; i++) {
      if (float(i) == floor(paletteIndex)) {
        colorA = colors[i];
        if (i < COLOR_COUNT - 1) {
          colorB = colors[i + 1];
        } else {
          colorB = colorA;
        }
        break;
      }
    }

    float ditherValue = Bayer64(pixelCoord * 0.25);
    float blendAmount = float(fract(paletteIndex) > ditherValue);
    return mix(colorA, colorB, blendAmount);
  }

  float flowField(vec2 p, float t) {
    return sin(p.x + sin(p.y + t * 0.1)) * sin(p.y * p.x * 0.1 + t * 0.2);
  }

  vec2 computeField(vec2 p, float t) {
    vec2 ep = vec2(0.05, 0.0);
    vec2 result = vec2(0.0);

    for (int i = 0; i < 20; i++) {
      float t0 = flowField(p, t);
      float t1 = flowField(p + ep.xy, t);
      float t2 = flowField(p + ep.yx, t);
      vec2 gradient = vec2((t1 - t0), (t2 - t0)) / ep.xx;
      vec2 tangent = vec2(-gradient.y, gradient.x);

      p += tangent * 0.5 + gradient * 0.005;
      p.x += sin(t * 0.25) * 0.1;
      p.y += cos(t * 0.25) * 0.1;
      result = gradient;
    }

    return result;
  }

  void main() {
    setupColorPalette();

    vec2 uv = gl_FragCoord.xy / iResolution.xy - 0.5;
    uv.x *= iResolution.x / iResolution.y;
    float animTime = iTime * uSpeed;

    vec2 p = uv * uScale;
    vec2 field = computeField(p, animTime);

    float colorValue = length(field) * uIntensity;
    colorValue = clamp(colorValue, 0.0, 1.0);

    vec3 finalColor = applyDitheredColor(colorValue, gl_FragCoord.xy / uDownScale);
    gl_FragColor = vec4(finalColor, uOpacity);
  }
`;

function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const match = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return match
    ? {
        r: parseInt(match[1], 16) / 255,
        g: parseInt(match[2], 16) / 255,
        b: parseInt(match[3], 16) / 255
      }
    : { r: 0, g: 0, b: 0 };
}

export default function DitherWave({
  width = '100%',
  height = '100%',
  speed = 1,
  intensity = 1,
  scale = 6,
  downScale = 0.5,
  primaryColor = '#5227FF',
  secondaryColor = '#5227FF',
  tertiaryColor = '#0a0a0a',
  opacity = 1,
  quality = 'medium',
  maxFPS = 60,
  pauseWhenOffscreen = true,
  className,
  children
}: DitherWaveProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const rafId = useRef<number>(0);
  const startTime = useRef<number>(0);
  const lastFrameTime = useRef<number>(0);
  const isVisible = useRef<boolean>(true);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const c1 = hexToRgb(primaryColor);
    const c2 = hexToRgb(secondaryColor);
    const c3 = hexToRgb(tertiaryColor);

    const rect = container.getBoundingClientRect();
    const w = rect.width;
    const h = rect.height;

    const preset = {
      low: { pixelRatio: 1, antialias: false },
      medium: { pixelRatio: Math.min(window.devicePixelRatio || 1, 2), antialias: true },
      high: { pixelRatio: Math.min(window.devicePixelRatio || 1, 3), antialias: true }
    }[quality];

    const renderer = new WebGLRenderer({
      antialias: preset.antialias,
      alpha: true,
      powerPreference: 'high-performance',
      stencil: false,
      depth: false
    });
    renderer.setClearColor(0, 0);

    const dpr = preset.pixelRatio;
    renderer.setSize(w, h, false);
    renderer.setPixelRatio(dpr);
    renderer.domElement.style.width = '100%';
    renderer.domElement.style.height = '100%';
    renderer.domElement.style.display = 'block';
    container.appendChild(renderer.domElement);

    const scene = new Scene();
    const camera = new OrthographicCamera(-1, 1, 1, -1, 0, 1);

    const uniforms = {
      iTime: { value: 0 },
      iResolution: { value: new Vector2(w * dpr, h * dpr) },
      uSpeed: { value: speed },
      uIntensity: { value: intensity },
      uScale: { value: scale },
      uDownScale: { value: downScale },
      uOpacity: { value: opacity },
      uColor1: { value: new Color(c1.r, c1.g, c1.b) },
      uColor2: { value: new Color(c2.r, c2.g, c2.b) },
      uColor3: { value: new Color(c3.r, c3.g, c3.b) }
    };

    const material = new ShaderMaterial({
      uniforms,
      vertexShader,
      fragmentShader,
      transparent: true
    });

    const geometry = new PlaneGeometry(2, 2);
    const quad = new Mesh(geometry, material);
    scene.add(quad);

    let observer: IntersectionObserver | null = null;
    if (pauseWhenOffscreen) {
      observer = new IntersectionObserver(
        entries => {
          isVisible.current = entries[0].isIntersecting;
        },
        { threshold: 0 }
      );
      observer.observe(container);
    }

    const frameInterval = 1000 / maxFPS;

    const animate = (timestamp: number) => {
      rafId.current = requestAnimationFrame(animate);

      if (!startTime.current) {
        startTime.current = timestamp;
        lastFrameTime.current = timestamp;
      }

      const elapsed = timestamp - lastFrameTime.current;
      if (elapsed < frameInterval) return;

      lastFrameTime.current = timestamp - (elapsed % frameInterval);

      if (!pauseWhenOffscreen || isVisible.current) {
        uniforms.iTime.value = (timestamp - startTime.current) * 0.001;
        uniforms.uSpeed.value = speed;
        uniforms.uIntensity.value = intensity;
        uniforms.uScale.value = scale;
        uniforms.uDownScale.value = downScale;
        uniforms.uOpacity.value = opacity;
        renderer.render(scene, camera);
      }
    };

    rafId.current = requestAnimationFrame(animate);

    const handleResize = () => {
      const b = container.getBoundingClientRect();
      const rw = b.width;
      const rh = b.height;
      renderer.setSize(rw, rh, false);
      uniforms.iResolution.value.set(rw * dpr, rh * dpr);
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (rafId.current) cancelAnimationFrame(rafId.current);
      if (observer) observer.disconnect();
      geometry.dispose();
      material.dispose();
      renderer.dispose();
      if (container.contains(renderer.domElement)) {
        container.removeChild(renderer.domElement);
      }
    };
  }, [
    speed,
    intensity,
    scale,
    downScale,
    primaryColor,
    secondaryColor,
    tertiaryColor,
    opacity,
    quality,
    maxFPS,
    pauseWhenOffscreen
  ]);

  const cssWidth = typeof width === 'number' ? `${width}px` : width;
  const cssHeight = typeof height === 'number' ? `${height}px` : height;

  return (
    <div
      ref={containerRef}
      className={cn('relative overflow-hidden', className)}
      style={{ width: cssWidth, height: cssHeight }}
    >
      {children && (
        <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center">
          {children}
        </div>
      )}
    </div>
  );
}
