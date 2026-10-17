'use client';

import React, { useRef, useMemo, useEffect, useState } from 'react';
import { Canvas, useThree, useFrame, extend, createPortal } from '@react-three/fiber';
import { shaderMaterial, useFBO } from '@react-three/drei';
import {
  Vector2,
  Color,
  Scene,
  NearestFilter,
  MathUtils,
  SRGBColorSpace,
  ShaderMaterial as ThreeShaderMaterial,
  Texture
} from 'three';
import { cn } from '@/lib/utils';

/* -------------------------------------------------------------------------- */
/*                               SHADER CODES                                 */
/* -------------------------------------------------------------------------- */

const simulationVertexShader = /* glsl */ `
  varying vec2 vUv;
  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const simulationFragmentShader = /* glsl */ `
  uniform float uTime;
  uniform vec2 uMouse;
  uniform sampler2D uPreviousState;
  uniform vec2 uResolution;
  uniform float uRadius;
  uniform float uDecay;
  uniform float uIntensity;
  uniform float uSpeed;
  uniform vec2 uDirection;

  varying vec2 vUv;

  vec3 mod289(vec3 x) { return x - floor(x * (1.0 / 289.0)) * 289.0; }
  vec2 mod289(vec2 x) { return x - floor(x * (1.0 / 289.0)) * 289.0; }
  vec3 permute(vec3 x) { return mod289(((x * 34.0) + 1.0) * x); }

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

    float prev   = texture2D(uPreviousState, advectedUV).r;
    float top    = texture2D(uPreviousState, advectedUV + vec2(0.0, texel.y)).r;
    float bottom = texture2D(uPreviousState, advectedUV - vec2(0.0, texel.y)).r;
    float left   = texture2D(uPreviousState, advectedUV - vec2(texel.x, 0.0)).r;
    float right  = texture2D(uPreviousState, advectedUV + vec2(texel.x, 0.0)).r;

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

    gl_FragColor = vec4(vec3(max(0.0, value)), 1.0);
  }
`;

const ditherVertexShader = /* glsl */ `
  varying vec2 vUv;
  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const ditherFragmentShader = /* glsl */ `
  uniform sampler2D uSimulationState;
  uniform float uDitherSize;
  uniform float uExponent;
  uniform vec2 uResolution;
  uniform vec3 uColor;

  varying vec2 vUv;

  float bayer8(vec2 uv) {
    int x = int(mod(uv.x, 8.0));
    int y = int(mod(uv.y, 8.0));

    int M[64];
    M[0]=0;  M[1]=32; M[2]=8;  M[3]=40; M[4]=2;  M[5]=34; M[6]=10; M[7]=42;
    M[8]=48; M[9]=16; M[10]=56;M[11]=24;M[12]=50;M[13]=18;M[14]=58;M[15]=26;
    M[16]=12;M[17]=44;M[18]=4; M[19]=36;M[20]=14;M[21]=46;M[22]=6; M[23]=38;
    M[24]=60;M[25]=28;M[26]=52;M[27]=20;M[28]=62;M[29]=30;M[30]=54;M[31]=22;
    M[32]=3; M[33]=35;M[34]=11;M[35]=43;M[36]=1; M[37]=33;M[38]=9; M[39]=41;
    M[40]=51;M[41]=19;M[42]=59;M[43]=27;M[44]=49;M[45]=17;M[46]=57;M[47]=25;
    M[48]=15;M[49]=47;M[50]=7; M[51]=39;M[52]=13;M[53]=45;M[54]=5; M[55]=37;
    M[56]=63;M[57]=31;M[58]=55;M[59]=23;M[60]=61;M[61]=29;M[62]=53;M[63]=21;

    int idx = y * 8 + x;
    return float(M[idx]) / 64.0;
  }

  void main() {
    vec2 uv = vUv;

    float signal = texture2D(uSimulationState, uv).r;
    signal = pow(signal, uExponent);

    float threshold = bayer8(gl_FragCoord.xy / uDitherSize);
    float mask = step(threshold, signal);

    if (signal < 0.01) mask = 0.0;
    if (mask < 0.5) discard;

    vec3 finalColor = uColor;
    gl_FragColor = vec4(finalColor, 1.0);
  }
`;

/* -------------------------------------------------------------------------- */
/*                        THREE.JS SHADER MATERIALS                           */
/* -------------------------------------------------------------------------- */

const SimulationMaterial = shaderMaterial(
  {
    uTime: 0,
    uMouse: new Vector2(0, 0),
    uPreviousState: null as Texture | null,
    uResolution: new Vector2(0, 0),
    uRadius: 0.1,
    uDecay: 0.01,
    uIntensity: 1.0,
    uSpeed: 0.0,
    uDirection: new Vector2(0, 0)
  },
  simulationVertexShader,
  simulationFragmentShader
);

const DitherMaterial = shaderMaterial(
  {
    uSimulationState: null as Texture | null,
    uDitherSize: 8,
    uExponent: 2.0,
    uResolution: new Vector2(0, 0),
    uColor: new Color(0, 0, 0)
  },
  ditherVertexShader,
  ditherFragmentShader
);

extend({ SimulationMaterial, DitherMaterial });

type SimulationMaterialImpl = ThreeShaderMaterial & {
  uTime: number;
  uMouse: Vector2;
  uPreviousState: Texture | null;
  uResolution: Vector2;
  uRadius: number;
  uDecay: number;
  uIntensity: number;
  uSpeed: number;
  uDirection: Vector2;
};

type DitherMaterialImpl = ThreeShaderMaterial & {
  uSimulationState: Texture | null;
  uDitherSize: number;
  uExponent: number;
  uResolution: Vector2;
  uColor: Color;
};

declare module '@react-three/fiber' {
  interface ThreeElements {
    simulationMaterial: React.ComponentProps<'shaderMaterial'> & {
      ref?: React.Ref<SimulationMaterialImpl>;
    };
    ditherMaterial: React.ComponentProps<'shaderMaterial'> & {
      ref?: React.Ref<DitherMaterialImpl>;
      transparent?: boolean;
    };
  }
}

/* -------------------------------------------------------------------------- */
/*                           SCENE INNER COMPONENT                            */
/* -------------------------------------------------------------------------- */

interface DitherCursorInnerProps {
  simulationScene: Scene;
  ditherSize: number;
  radius: number;
  exponent: number;
  decay: number;
  intensity: number;
  color: string;
}

function DitherCursorInner({
  simulationScene,
  ditherSize,
  radius,
  exponent,
  decay,
  intensity,
  color
}: DitherCursorInnerProps) {
  const { size, viewport, gl } = useThree();
  const mouse = useRef(new Vector2(0, 0));
  const prevMouse = useRef(new Vector2(0, 0));
  const speed = useRef(0);

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      const el = gl.domElement;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width;
      const y = 1 - (e.clientY - rect.top) / rect.height;
      mouse.current.set(x, y);
    };

    window.addEventListener('mousemove', onMouseMove, { passive: true });
    return () => window.removeEventListener('mousemove', onMouseMove);
  }, [gl.domElement]);

  const targetA = useFBO({ minFilter: NearestFilter, magFilter: NearestFilter });
  const targetB = useFBO({ minFilter: NearestFilter, magFilter: NearestFilter });
  const simMatRef = useRef<SimulationMaterialImpl>(null);
  const ditherMatRef = useRef<DitherMaterialImpl>(null);
  const frameCount = useRef(0);

  useFrame(state => {
    const { gl: renderer, clock } = state;
    const currentTarget = frameCount.current % 2 === 0 ? targetA : targetB;
    const prevTarget = frameCount.current % 2 === 0 ? targetB : targetA;

    if (simMatRef.current) {
      simMatRef.current.uTime = clock.elapsedTime;
      simMatRef.current.uMouse = mouse.current;
      simMatRef.current.uPreviousState = prevTarget.texture;
      simMatRef.current.uResolution = new Vector2(size.width, size.height);
      simMatRef.current.uRadius = radius;
      simMatRef.current.uDecay = decay;
      simMatRef.current.uIntensity = intensity;

      const m = mouse.current;
      const pm = prevMouse.current;
      const distance = m.distanceTo(pm);
      speed.current = MathUtils.lerp(speed.current, distance, 0.1);
      simMatRef.current.uSpeed = speed.current;

      const dir = new Vector2().subVectors(m, pm).normalize();
      simMatRef.current.uDirection = dir;
      prevMouse.current.copy(m);
    }

    renderer.setRenderTarget(currentTarget);
    renderer.render(simulationScene, state.camera);
    renderer.setRenderTarget(null);

    if (ditherMatRef.current) {
      ditherMatRef.current.uSimulationState = currentTarget.texture;
      ditherMatRef.current.uDitherSize = ditherSize;
      ditherMatRef.current.uExponent = exponent;
      ditherMatRef.current.uResolution = new Vector2(size.width, size.height);

      const hex = color.replace('#', '');
      const rVal = parseInt(hex.substring(0, 2), 16) / 255;
      const gVal = parseInt(hex.substring(2, 4), 16) / 255;
      const bVal = parseInt(hex.substring(4, 6), 16) / 255;
      ditherMatRef.current.uColor = new Color(rVal, gVal, bVal);
    }

    frameCount.current++;
  });

  return (
    <>
      {createPortal(
        <mesh>
          <planeGeometry args={[viewport.width, viewport.height]} />
          <simulationMaterial ref={simMatRef} />
        </mesh>,
        simulationScene
      )}
      <mesh>
        <planeGeometry args={[viewport.width, viewport.height]} />
        <ditherMaterial ref={ditherMatRef} transparent />
      </mesh>
    </>
  );
}

/* -------------------------------------------------------------------------- */
/*                            PUBLIC EXPORT                                   */
/* -------------------------------------------------------------------------- */

export interface DitherCursorProps {
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
  /** Color of the dithered dots in hex (default: "#ffffff") */
  color?: string;
  /** Container CSS classes */
  className?: string;
}

export default function DitherCursor({
  ditherSize = 4,
  radius = 0.1,
  exponent = 2.0,
  decay = 0.01,
  intensity = 0.5,
  color = '#ffffff',
  className
}: DitherCursorProps) {
  const simulationScene = useMemo(() => new Scene(), []);

  return (
    <div
      className={cn('fixed inset-0 w-full h-full pointer-events-none z-50 overflow-hidden', className)}
      style={{ pointerEvents: 'none' }}
    >
      <Canvas
        orthographic
        camera={{ zoom: 1, position: [0, 0, 1], near: 0.1, far: 1000 }}
        gl={{ alpha: true, antialias: false }}
        events={() => ({
          enabled: false,
          priority: 0,
          handlePointer: () => () => {},
          connected: false as unknown as HTMLElement
        })}
        style={{ pointerEvents: 'none' }}
        onCreated={({ gl }) => {
          gl.outputColorSpace = SRGBColorSpace;
          gl.domElement.style.pointerEvents = 'none';
        }}
        className="w-full h-full pointer-events-none"
      >
        <DitherCursorInner
          simulationScene={simulationScene}
          ditherSize={ditherSize}
          radius={radius}
          exponent={exponent}
          decay={decay}
          intensity={intensity}
          color={color}
        />
      </Canvas>
    </div>
  );
}
