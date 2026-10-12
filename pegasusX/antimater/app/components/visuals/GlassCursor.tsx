'use client';

import React, { useRef, useMemo, useEffect, useCallback } from 'react';
import { Canvas, useThree, useFrame } from '@react-three/fiber';
import {
  Vector2,
  Texture,
  TextureLoader,
  LinearFilter,
  ClampToEdgeWrapping,
  Mesh,
  ShaderMaterial as ThreeShaderMaterial
} from 'three';
import { cn } from '@/lib/utils';

const glassVertexShader = /* glsl */ `
  varying vec2 vPos;
  void main() {
    vPos = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const glassFragmentShader = /* glsl */ `
  precision highp float;
  varying vec2 vPos;

  #define MAX_TRAIL 40
  #define INV_81 0.01234567901

  uniform sampler2D uImage;
  uniform vec2 uViewport;
  uniform vec2 uInvViewport;
  uniform float uImgRatio;
  uniform vec2 uTrail[MAX_TRAIL];
  uniform float uRadii[MAX_TRAIL];
  uniform int uTrailCount;
  uniform float uPresence;
  uniform float uThreshold;
  uniform float uRefract;
  uniform float uSpread;
  uniform float uRim;
  uniform float uSpec;
  uniform vec2 uBlobCenter;
  uniform float uWarp;
  uniform float uWarpScale;
  uniform float uTime;
  uniform float uAlpha;
  uniform vec2 uBoundsMin;
  uniform vec2 uBoundsMax;

  vec2 coverScale;

  void initCoverScale() {
    float vr = uViewport.x * uInvViewport.y;
    coverScale = vr > uImgRatio
      ? vec2(1.0, uImgRatio / vr)
      : vec2(vr / uImgRatio, 1.0);
  }

  vec2 coverFit(vec2 c) {
    return (c - 0.5) * coverScale + 0.5;
  }

  float capsule(vec2 p, vec2 a, vec2 b, float r) {
    vec2 pa = p - a;
    vec2 ba = b - a;
    float h = clamp(dot(pa, ba) / (dot(ba, ba) + 1e-8), 0.0, 1.0);
    float d = length(pa - ba * h);
    return (r * r) / (d * d + 1e-6);
  }

  float metafield(vec2 p) {
    float f = 0.0;
    float ar = uViewport.x * uInvViewport.y;
    vec2 asp = vec2(ar, 1.0);
    int n = min(uTrailCount, MAX_TRAIL);

    vec2 q = p;
    if (uWarp > 0.0) {
      float wk = uWarp * 0.0008;
      float px = p.x * uWarpScale;
      float py = p.y * uWarpScale;
      float t1 = uTime * 2.3;
      float t2 = uTime * 1.7;
      q.x += (sin(py + t1) * 0.6 + sin(py * 2.3 + px * 0.7 + t2) * 0.4) * wk;
      q.y += (cos(px + t2) * 0.6 + cos(px * 2.1 + py * 0.9 + t1) * 0.4) * wk;
    }
    q *= asp;

    for (int i = 0; i < MAX_TRAIL; i++) {
      if (i >= n) break;

      float r0 = uRadii[i];
      vec2 a = (uTrail[i] * uInvViewport) * asp;

      if (i < n - 1) {
        float r1 = uRadii[i + 1];
        vec2 b = (uTrail[i + 1] * uInvViewport) * asp;
        float rm = (r0 + r1) * 0.5;
        f += capsule(q, a, b, rm);
      } else {
        vec2 d = q - a;
        f += (r0 * r0) / (dot(d, d) + 1e-6);
      }
    }
    return f;
  }

  void main() {
    initCoverScale();

    vec2 st = vPos;

    if (st.x < uBoundsMin.x || st.x > uBoundsMax.x ||
        st.y < uBoundsMin.y || st.y > uBoundsMax.y) {
      gl_FragColor = vec4(texture2D(uImage, coverFit(st)).rgb, uAlpha);
      return;
    }

    float f = metafield(st);
    float nf = f / uThreshold;

    float fill = smoothstep(0.5, 1.0, nf);
    float edge = smoothstep(0.35, 0.55, nf) - smoothstep(0.7, 0.95, nf);
    float mask = smoothstep(0.0, 1.0, clamp(fill + edge * 0.5, 0.0, 1.0)) * uPresence;

    if (mask < 0.001) {
      gl_FragColor = vec4(texture2D(uImage, coverFit(st)).rgb, uAlpha);
      return;
    }

    vec2 bent = (st - uBlobCenter) * (1.0 - uRefract * fill) + uBlobCenter;
    vec2 baseFit = coverFit(bent);
    vec2 tapStep = coverScale * uSpread * uInvViewport;

    vec4 px = vec4(0.0);
    for (float i = -4.0; i <= 4.0; i++) {
      for (float j = -4.0; j <= 4.0; j++) {
        px += texture2D(uImage, baseFit + vec2(i, j) * tapStep);
      }
    }
    px *= INV_81;

    vec2 head = uTrail[0] * uInvViewport;
    vec2 dh = st - head;
    float glow = clamp((clamp(dh.y, 0.0, 0.15) + 0.06) * 2.5, 0.0, 1.0);

    vec4 lit = clamp(px + vec4(fill) * glow * uSpec + vec4(edge) * uRim, 0.0, 1.0);
    vec3 base = texture2D(uImage, coverFit(st)).rgb;

    gl_FragColor = vec4(mix(base, lit.rgb, mask), uAlpha);
  }
`;

interface GlassMeshProps {
  src: string;
  pointerRef: React.MutableRefObject<{ target: [number, number]; smooth: [number, number] }>;
  trailPool: Vector2[];
  trailCountRef: React.MutableRefObject<number>;
  presenceRef: React.MutableRefObject<number>;
  activeRef: React.MutableRefObject<boolean>;
  dampening: number;
  blobSize: number;
  trailLength: number;
  tailFade: number;
  threshold: number;
  refraction: number;
  blurSpread: number;
  borderGlow: number;
  specularGain: number;
  blobWarpAmount: number;
  blobWarpScale: number;
  opacity: number;
}

function GlassMesh({
  src,
  pointerRef,
  trailPool,
  trailCountRef,
  presenceRef,
  activeRef,
  dampening,
  blobSize,
  trailLength,
  tailFade,
  threshold,
  refraction,
  blurSpread,
  borderGlow,
  specularGain,
  blobWarpAmount,
  blobWarpScale,
  opacity
}: GlassMeshProps) {
  const meshRef = useRef<Mesh>(null);
  const textureRef = useRef<Texture | null>(null);
  const imgRatioRef = useRef<number>(1);
  const { size } = useThree();
  const radiiRef = useRef(new Float32Array(40));

  useEffect(() => {
    const loader = new TextureLoader();
    loader.setCrossOrigin('anonymous');
    loader.load(src, tex => {
      tex.minFilter = LinearFilter;
      tex.magFilter = LinearFilter;
      tex.wrapS = ClampToEdgeWrapping;
      tex.wrapT = ClampToEdgeWrapping;
      textureRef.current = tex;
      if (tex.image) {
        imgRatioRef.current = tex.image.width / tex.image.height;
      }
    });
  }, [src]);

  const uniforms = useMemo(
    () => ({
      uImage: { value: null as Texture | null },
      uViewport: { value: new Vector2(1, 1) },
      uInvViewport: { value: new Vector2(1, 1) },
      uImgRatio: { value: 1 },
      uTrail: { value: trailPool },
      uRadii: { value: new Float32Array(40) },
      uTrailCount: { value: 0 },
      uPresence: { value: 0 },
      uThreshold: { value: 1 },
      uRefract: { value: 0.015 },
      uSpread: { value: 0.5 },
      uRim: { value: 0.3 },
      uSpec: { value: 1 },
      uBlobCenter: { value: new Vector2(0.5, 0.5) },
      uWarp: { value: 0 },
      uWarpScale: { value: 12 },
      uTime: { value: 0 },
      uAlpha: { value: 1 },
      uBoundsMin: { value: new Vector2(0, 0) },
      uBoundsMax: { value: new Vector2(1, 1) }
    }),
    [trailPool]
  );

  useFrame((state, delta) => {
    if (!meshRef.current || !textureRef.current) return;
    const mat = meshRef.current.material as ThreeShaderMaterial;
    const ptr = pointerRef.current;

    if (ptr.target[0] > -999) {
      activeRef.current = true;
      if (ptr.smooth[0] < -999) {
        ptr.smooth[0] = ptr.target[0];
        ptr.smooth[1] = ptr.target[1];
        const sx = ptr.smooth[0];
        const sy = size.height - ptr.smooth[1];
        for (let i = 0; i < 40; i++) trailPool[i].set(sx, sy);
        trailCountRef.current = Math.min(trailLength, 40);
      } else {
        const factor = 1 - Math.pow(dampening, Math.min(60 * delta, 4));
        ptr.smooth[0] += (ptr.target[0] - ptr.smooth[0]) * factor;
        ptr.smooth[1] += (ptr.target[1] - ptr.smooth[1]) * factor;
        for (let i = 39; i >= 1; i--) trailPool[i].copy(trailPool[i - 1]);
        trailPool[0].set(ptr.smooth[0], size.height - ptr.smooth[1]);
        trailCountRef.current = Math.min((trailCountRef.current ?? 0) + 1, Math.min(trailLength, 40));
      }
    } else {
      activeRef.current = false;
    }

    const targetPresence = activeRef.current ? 1 : 0;
    const presenceSpeed = activeRef.current ? 8 : 3;
    presenceRef.current += (targetPresence - presenceRef.current) * (1 - Math.exp(-presenceSpeed * delta));
    if (presenceRef.current < 0.002) presenceRef.current = 0;
    if (presenceRef.current > 0.998) presenceRef.current = 1;

    if (!activeRef.current && presenceRef.current <= 0) {
      ptr.smooth[0] = -9999;
      ptr.smooth[1] = -9999;
      trailCountRef.current = 0;
      for (let i = 0; i < 40; i++) trailPool[i].set(-9999, -9999);
    }

    const rad = radiiRef.current;
    const count = trailCountRef.current ?? 0;
    const presence = presenceRef.current;
    const invCount = 1 / Math.max(count - 1, 1);
    const invW = 1 / size.width;
    const invH = 1 / size.height;

    let sumX = 0, sumY = 0, sumW = 0;
    let minX = 1e9, minY = 1e9, maxX = -1e9, maxY = -1e9;
    let maxR = 0;

    for (let i = 0; i < 40; i++) {
      if (i < count) {
        const tNorm = i * invCount;
        const curR = blobSize * presence * Math.pow(Math.max(1 - tNorm, 0.001), tailFade);
        rad[i] = curR;
        const px = trailPool[i].x * invW;
        const py = trailPool[i].y * invH;
        const weight = Math.pow(Math.max(1 - tNorm, 0.001), tailFade);
        sumX += px * weight;
        sumY += py * weight;
        sumW += weight;
        if (px < minX) minX = px;
        if (px > maxX) maxX = px;
        if (py < minY) minY = py;
        if (py > maxY) maxY = py;
        if (curR > maxR) maxR = curR;
      } else {
        rad[i] = 0;
      }
    }

    if (sumW > 0) {
      sumX /= sumW;
      sumY /= sumW;
    } else {
      sumX = 0.5;
      sumY = 0.5;
    }

    const ar = size.width * invH;
    const warpPad = 0.06 * (blobWarpAmount > 0 ? 1 : 0);
    const blurPad = blurSpread * Math.max(invW, invH) * 5;
    const boundRadius = 12 * maxR;
    const boundPadX = boundRadius / ar + warpPad + blurPad;
    const boundPadY = boundRadius + warpPad + blurPad;

    const bMin = mat.uniforms.uBoundsMin.value as Vector2;
    const bMax = mat.uniforms.uBoundsMax.value as Vector2;

    if (count > 0 && presence > 0) {
      bMin.set(Math.max(0, minX - boundPadX), Math.max(0, minY - boundPadY));
      bMax.set(Math.min(1, maxX + boundPadX), Math.min(1, maxY + boundPadY));
    } else {
      bMin.set(2, 2);
      bMax.set(-1, -1);
    }

    mat.uniforms.uImage.value = textureRef.current;
    mat.uniforms.uViewport.value.set(size.width, size.height);
    mat.uniforms.uInvViewport.value.set(invW, invH);
    mat.uniforms.uImgRatio.value = imgRatioRef.current;
    mat.uniforms.uTrail.value = trailPool;
    mat.uniforms.uRadii.value = rad;
    mat.uniforms.uTrailCount.value = count;
    mat.uniforms.uPresence.value = presence;
    mat.uniforms.uThreshold.value = threshold;
    mat.uniforms.uRefract.value = refraction;
    mat.uniforms.uSpread.value = blurSpread;
    mat.uniforms.uRim.value = borderGlow;
    mat.uniforms.uSpec.value = specularGain;
    mat.uniforms.uBlobCenter.value.set(sumX, sumY);
    mat.uniforms.uWarp.value = blobWarpAmount;
    mat.uniforms.uWarpScale.value = blobWarpScale;
    mat.uniforms.uTime.value = state.clock.elapsedTime;
    mat.uniforms.uAlpha.value = opacity;
  });

  return (
    <mesh ref={meshRef}>
      <planeGeometry args={[2, 2]} />
      <shaderMaterial
        vertexShader={glassVertexShader}
        fragmentShader={glassFragmentShader}
        uniforms={uniforms}
        transparent
      />
    </mesh>
  );
}

export interface GlassCursorProps {
  width?: number | string;
  height?: number | string;
  className?: string;
  children?: React.ReactNode;
  src?: string;
  dampening?: number;
  blobSize?: number;
  trailLength?: number;
  tailFade?: number;
  threshold?: number;
  refraction?: number;
  blurSpread?: number;
  borderGlow?: number;
  specularGain?: number;
  blobWarpAmount?: number;
  blobWarpScale?: number;
  backgroundColor?: string;
  opacity?: number;
}

export default function GlassCursor({
  width = '100%',
  height = '100%',
  className,
  children,
  src = 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=1920&q=80',
  dampening = 0,
  blobSize = 0.04,
  trailLength = 40,
  tailFade = 0.3,
  threshold = 0.6,
  refraction = 0.3,
  blurSpread = 0.3,
  borderGlow = 0.15,
  specularGain = 0.2,
  blobWarpAmount = 25,
  blobWarpScale = 5,
  backgroundColor = '#000000',
  opacity = 1
}: GlassCursorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const pointerRef = useRef({
    target: [-9999, -9999] as [number, number],
    smooth: [-9999, -9999] as [number, number]
  });
  const trailCountRef = useRef(0);
  const presenceRef = useRef(0);
  const activeRef = useRef(false);
  const rectRef = useRef<DOMRect | null>(null);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => {
      rectRef.current = el.getBoundingClientRect();
    };
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    window.addEventListener('scroll', update, { passive: true });
    return () => {
      ro.disconnect();
      window.removeEventListener('scroll', update);
    };
  }, []);

  const trailPool = useMemo(
    () => Array.from({ length: 40 }, () => new Vector2(-9999, -9999)),
    []
  );

  const onPointerMove = useCallback((e: React.PointerEvent) => {
    const rect = rectRef.current;
    if (rect) {
      pointerRef.current.target = [e.clientX - rect.left, e.clientY - rect.top];
    }
  }, []);

  const onPointerLeave = useCallback(() => {
    pointerRef.current.target = [-9999, -9999];
  }, []);

  const cssW = typeof width === 'number' ? `${width}px` : width;
  const cssH = typeof height === 'number' ? `${height}px` : height;

  return (
    <div
      ref={containerRef}
      className={cn('relative overflow-hidden', className)}
      style={{ width: cssW, height: cssH, background: backgroundColor }}
      onPointerMove={onPointerMove}
      onPointerLeave={onPointerLeave}
    >
      <Canvas
        className="absolute inset-0 h-full w-full"
        dpr={[1, 2]}
        gl={{ antialias: false, alpha: true, powerPreference: 'high-performance' }}
        orthographic
        camera={{ position: [0, 0, 1], zoom: 1, left: -1, right: 1, top: 1, bottom: -1 }}
      >
        <GlassMesh
          src={src}
          pointerRef={pointerRef}
          trailPool={trailPool}
          trailCountRef={trailCountRef}
          presenceRef={presenceRef}
          activeRef={activeRef}
          dampening={dampening}
          blobSize={blobSize}
          trailLength={trailLength}
          tailFade={tailFade}
          threshold={threshold}
          refraction={refraction}
          blurSpread={blurSpread}
          borderGlow={borderGlow}
          specularGain={specularGain}
          blobWarpAmount={blobWarpAmount}
          blobWarpScale={blobWarpScale}
          opacity={opacity}
        />
      </Canvas>
      {children && <div className="pointer-events-none relative z-10">{children}</div>}
    </div>
  );
}
