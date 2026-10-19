'use client';

import React, { useRef, useMemo, useEffect } from 'react';
import { Canvas, useThree, useFrame, extend, createPortal } from '@react-three/fiber';
import { shaderMaterial, useFBO } from '@react-three/drei';
import { EffectComposer, Bloom, ChromaticAberration } from '@react-three/postprocessing';
import {
  Vector2,
  Color,
  Scene,
  NearestFilter,
  CanvasTexture,
  MathUtils,
  ShaderMaterial as ThreeShaderMaterial,
  Texture
} from 'three';
import { cn } from '@/lib/utils';

/* -------------------------------------------------------------------------- */
/*                               SHADER CODES                                 */
/* -------------------------------------------------------------------------- */

const asciiSimulationVertexShader = /* glsl */ `
  varying vec2 vUv;
  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const asciiSimulationFragmentShader = /* glsl */ `
  uniform float uTime;
  uniform vec2 uMouse;
  uniform sampler2D uPreviousState;
  uniform vec2 uResolution;
  uniform float uDecay;
  uniform float uSpeed;
  uniform float uRadius;

  varying vec2 vUv;

  void main() {
    vec2 uv = vUv;
    float prev = texture2D(uPreviousState, uv).r;

    float aspect = uResolution.x / uResolution.y;
    vec2 aspectCorrection = vec2(aspect, 1.0);
    vec2 mouseUV = uMouse;

    float dist = length((uv - mouseUV) * aspectCorrection);
    float brush = smoothstep(uRadius, 0.0, dist);

    float speedFactor = smoothstep(0.0, 0.1, uSpeed);
    brush *= speedFactor;

    float value = max(prev, brush);
    value -= uDecay;

    gl_FragColor = vec4(vec3(max(0.0, value)), 1.0);
  }
`;

const asciiDisplayVertexShader = /* glsl */ `
  varying vec2 vUv;
  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const asciiDisplayFragmentShader = /* glsl */ `
  uniform sampler2D uSimulationState;
  uniform sampler2D uFontTexture;
  uniform vec2 uResolution;
  uniform vec3 uColor;
  uniform vec3 uBackgroundColor;
  uniform float uCharCount;
  uniform float uGridSize;
  uniform bool uEnableFade;
  uniform float uOpacity;

  varying vec2 vUv;

  float random(vec2 st) {
    return fract(sin(dot(st.xy, vec2(12.9898, 78.233))) * 43758.5453123);
  }

  void main() {
    vec2 uv = vUv;

    vec2 gridDims = uResolution / uGridSize;
    vec2 cellUV = floor(uv * gridDims) / gridDims;

    float signal = texture2D(uSimulationState, cellUV + (0.5 / gridDims)).r;
    if (signal < 0.01) discard;

    float rnd = random(cellUV);
    float charIndex = floor(rnd * uCharCount);

    vec2 localUV = fract(uv * gridDims);
    vec2 atlasUV = vec2((localUV.x + charIndex) / uCharCount, localUV.y);

    float charMask = texture2D(uFontTexture, atlasUV).a;

    if (random(cellUV + 10.0) > 0.7) {
      charMask = 0.0;
    }

    float opacity = uEnableFade ? signal : 1.0;
    vec3 finalColor = mix(uBackgroundColor, uColor, charMask);

    gl_FragColor = vec4(finalColor, opacity * uOpacity);
  }
`;

/* -------------------------------------------------------------------------- */
/*                         CUSTOM SHADER MATERIALS                            */
/* -------------------------------------------------------------------------- */

const AsciiSimulationMaterial = shaderMaterial(
  {
    uTime: 0,
    uMouse: new Vector2(0, 0),
    uPreviousState: null as Texture | null,
    uResolution: new Vector2(0, 0),
    uDecay: 0.02,
    uSpeed: 0,
    uRadius: 0.05
  },
  asciiSimulationVertexShader,
  asciiSimulationFragmentShader
);

const AsciiDisplayMaterial = shaderMaterial(
  {
    uSimulationState: null as Texture | null,
    uFontTexture: null as Texture | null,
    uResolution: new Vector2(0, 0),
    uColor: new Color(0, 1, 0),
    uBackgroundColor: new Color(0, 0, 0),
    uCharCount: 10,
    uGridSize: 10,
    uEnableFade: true,
    uOpacity: 1.0
  },
  asciiDisplayVertexShader,
  asciiDisplayFragmentShader
);

extend({ AsciiSimulationMaterial, AsciiDisplayMaterial });

type AsciiSimulationMaterialImpl = ThreeShaderMaterial & {
  uTime: number;
  uMouse: Vector2;
  uPreviousState: Texture | null;
  uResolution: Vector2;
  uDecay: number;
  uSpeed: number;
  uRadius: number;
};

type AsciiDisplayMaterialImpl = ThreeShaderMaterial & {
  uSimulationState: Texture | null;
  uFontTexture: Texture | null;
  uResolution: Vector2;
  uColor: Color;
  uBackgroundColor: Color;
  uCharCount: number;
  uGridSize: number;
  uEnableFade: boolean;
  uOpacity: number;
};

declare module '@react-three/fiber' {
  interface ThreeElements {
    asciiSimulationMaterial: React.ComponentProps<'shaderMaterial'> & {
      ref?: React.Ref<AsciiSimulationMaterialImpl>;
    };
    asciiDisplayMaterial: React.ComponentProps<'shaderMaterial'> & {
      ref?: React.Ref<AsciiDisplayMaterialImpl>;
      transparent?: boolean;
    };
  }
}

/* -------------------------------------------------------------------------- */
/*                               SCENE INNER                                  */
/* -------------------------------------------------------------------------- */

function createFontTexture(chars: string, cellSize = 64): CanvasTexture {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  const count = chars.length;
  if (!ctx) return new CanvasTexture(canvas);

  canvas.width = count * cellSize;
  canvas.height = cellSize;

  ctx.font = `bold ${cellSize}px monospace`;
  ctx.fillStyle = 'white';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';

  for (let i = 0; i < count; i++) {
    const char = chars[i];
    const x = i * cellSize + cellSize / 2;
    const y = cellSize / 2;
    ctx.fillText(char, x, y);
  }

  const texture = new CanvasTexture(canvas);
  texture.minFilter = NearestFilter;
  texture.magFilter = NearestFilter;
  texture.needsUpdate = true;
  return texture;
}

interface AsciiCursorInnerProps {
  simulationScene: Scene;
  characters: string;
  size: number;
  color: string;
  backgroundColor: string;
  enableFade: boolean;
  spread: number;
  persistence: number;
  opacity: number;
  enableBloom: boolean;
  bloomStrength: number;
  bloomRadius: number;
  enableChromaticAberration: boolean;
  chromaticAberrationOffset: number;
}

function AsciiCursorInner({
  simulationScene,
  characters,
  size,
  color,
  backgroundColor,
  enableFade,
  spread,
  persistence,
  opacity,
  enableBloom,
  bloomStrength,
  bloomRadius,
  enableChromaticAberration,
  chromaticAberrationOffset
}: AsciiCursorInnerProps) {
  const { size: viewSize, viewport, gl } = useThree();
  const mouse = useRef(new Vector2(0, 0));
  const prevMouse = useRef(new Vector2(0, 0));
  const speed = useRef(0);

  const fontTexture = useMemo(() => createFontTexture(characters), [characters]);

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
  const simRef = useRef<AsciiSimulationMaterialImpl>(null);
  const displayRef = useRef<AsciiDisplayMaterialImpl>(null);
  const frameCount = useRef(0);

  useFrame(state => {
    const { gl: renderer, clock } = state;
    const currentTarget = frameCount.current % 2 === 0 ? targetA : targetB;
    const prevTarget = frameCount.current % 2 === 0 ? targetB : targetA;

    if (simRef.current) {
      simRef.current.uTime = clock.elapsedTime;
      simRef.current.uMouse = mouse.current;
      simRef.current.uPreviousState = prevTarget.texture;
      simRef.current.uResolution = new Vector2(viewSize.width, viewSize.height);
      simRef.current.uRadius = 0.01 * spread;
      simRef.current.uDecay = (1 / Math.max(0.1, persistence)) * 0.02;

      const m = mouse.current;
      const pm = prevMouse.current;
      const dist = m.distanceTo(pm);
      speed.current = MathUtils.lerp(speed.current, dist, 0.1);
      simRef.current.uSpeed = speed.current;
      prevMouse.current.copy(m);
    }

    renderer.setRenderTarget(currentTarget);
    renderer.render(simulationScene, state.camera);
    renderer.setRenderTarget(null);

    if (displayRef.current) {
      displayRef.current.uSimulationState = currentTarget.texture;
      displayRef.current.uFontTexture = fontTexture;
      displayRef.current.uResolution = new Vector2(viewSize.width, viewSize.height);
      displayRef.current.uColor = new Color(color);
      displayRef.current.uBackgroundColor = new Color(backgroundColor);
      displayRef.current.uCharCount = characters.length;
      displayRef.current.uGridSize = size;
      displayRef.current.uEnableFade = enableFade;
      displayRef.current.uOpacity = opacity;
    }

    frameCount.current++;
  });

  return (
    <>
      {createPortal(
        <mesh>
          <planeGeometry args={[viewport.width, viewport.height]} />
          <asciiSimulationMaterial ref={simRef} />
        </mesh>,
        simulationScene
      )}
      <mesh>
        <planeGeometry args={[viewport.width, viewport.height]} />
        <asciiDisplayMaterial ref={displayRef} transparent />
      </mesh>
      {(enableBloom || enableChromaticAberration) && (
        <EffectComposer>
          {enableBloom && (
            <Bloom
              luminanceThreshold={0}
              mipmapBlur
              intensity={bloomStrength}
              radius={bloomRadius}
            />
          )}
          {enableChromaticAberration && (
            <ChromaticAberration
              offset={new Vector2(chromaticAberrationOffset, chromaticAberrationOffset)}
              radialModulation={false}
              modulationOffset={0}
            />
          )}
        </EffectComposer>
      )}
    </>
  );
}

/* -------------------------------------------------------------------------- */
/*                            PUBLIC EXPORT                                   */
/* -------------------------------------------------------------------------- */

export interface AsciiCursorProps {
  characters?: string;
  size?: number;
  color?: string;
  backgroundColor?: string;
  enableFade?: boolean;
  spread?: number;
  persistence?: number;
  opacity?: number;
  enableBloom?: boolean;
  bloomStrength?: number;
  bloomRadius?: number;
  enableChromaticAberration?: boolean;
  chromaticAberrationOffset?: number;
  className?: string;
}

export default function AsciiCursor({
  characters = '✶✤↣⌧✷＄☺☉ϟ▵𝵏*.;.',
  size = 40,
  color = '#00ff00',
  backgroundColor = '#000000',
  enableFade = false,
  spread = 20,
  persistence = 2,
  opacity = 1,
  enableBloom = false,
  bloomStrength = 2,
  bloomRadius = 0.4,
  enableChromaticAberration = false,
  chromaticAberrationOffset = 0.005,
  className
}: AsciiCursorProps) {
  const simulationScene = useMemo(() => new Scene(), []);

  return (
    <div
      className={cn(
        'ascii-cursor-container absolute inset-0 overflow-hidden pointer-events-none z-50',
        className
      )}
    >
      <Canvas
        orthographic
        camera={{ zoom: 1, position: [0, 0, 1], near: 0.1, far: 1000 }}
        gl={{ alpha: true, antialias: false }}
        style={{ width: '100%', height: '100%' }}
      >
        <AsciiCursorInner
          simulationScene={simulationScene}
          characters={characters}
          size={size}
          color={color}
          backgroundColor={backgroundColor}
          enableFade={enableFade}
          spread={spread}
          persistence={persistence}
          opacity={opacity}
          enableBloom={enableBloom}
          bloomStrength={bloomStrength}
          bloomRadius={bloomRadius}
          enableChromaticAberration={enableChromaticAberration}
          chromaticAberrationOffset={chromaticAberrationOffset}
        />
      </Canvas>
    </div>
  );
}
