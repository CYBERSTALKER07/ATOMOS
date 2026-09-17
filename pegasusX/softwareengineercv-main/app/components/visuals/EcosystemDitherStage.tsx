'use client';

import React, { useEffect, useRef } from 'react';
import * as THREE from 'three';

export interface EcosystemDitherStageProps {
  activeStep: number;
  className?: string;
  theme?: 'dark' | 'light';
}

interface StepCameraTarget {
  azimuth: number;
  elevation: number;
  focus: [number, number, number];
  zoom: number;
  label: string;
}

const STEP_CAMERA_TARGETS: StepCameraTarget[] = [
  // 0: Supplier Operations (Factory Weighbridge & Pallet Staging)
  { azimuth: 0.785, elevation: 0.58, focus: [-1.8, 0.35, -1.2], zoom: 1.15, label: 'SUPPLIER_OUTBOUND' },
  // 1: Warehouse Control (Dock Bays & High-Bay Storage)
  { azimuth: 0.92, elevation: 0.64, focus: [-0.2, 0.45, 0.1], zoom: 1.25, label: 'CROSS_DOCK_WMS' },
  // 2: Retailer Network (Storefront Intake & Register Queue)
  { azimuth: 0.65, elevation: 0.54, focus: [1.7, 0.35, 1.1], zoom: 1.2, label: 'RETAILER_RECEPTION' },
  // 3: Global Fleet (City Transit Grid, Multi-Lane Trucks)
  { azimuth: 0.785, elevation: 0.32, focus: [0.0, 0.2, 0.0], zoom: 0.58, label: 'GLOBAL_FLEET_MESH' },
];

const DITHER_VERTEX_SHADER = `
  varying vec2 vUv;
  void main() {
    vUv = uv;
    gl_Position = vec4(position.xy, 0.0, 1.0);
  }
`;

const DITHER_FRAGMENT_SHADER = `
  uniform sampler2D uScene;
  uniform sampler2D uInk;
  uniform sampler2D uDepth;
  uniform vec2 uGrid;
  uniform vec2 uCellPx;
  uniform float uExposure;
  uniform float uLevels;
  uniform float uMinBlock;
  uniform float uMaxBlock;
  uniform float uBlack;
  uniform float uWhite;
  uniform float uPen;
  uniform vec3 uLight; // Background / Canvas color
  uniform vec3 uDark;  // Ink / Dot color
  uniform float uInvert;

  varying vec2 vUv;

  vec3 neutralToneMapping(vec3 color) {
    const float startCompression = 0.8 - 0.04;
    const float desaturation = 0.15;
    float x = min(color.r, min(color.g, color.b));
    float offset = x < 0.08 ? x - 6.25 * x * x : 0.04;
    color -= offset;
    float peak = max(color.r, max(color.g, color.b));
    if (peak < startCompression) return color;
    float d = 1.0 - startCompression;
    float newPeak = 1.0 - d * d / (peak + d - startCompression);
    color *= newPeak / peak;
    float g = 1.0 - 1.0 / (desaturation * (peak - newPeak) + 1.0);
    return mix(color, vec3(newPeak), g);
  }

  vec3 unmix(vec4 c) {
    return c.rgb / max(c.a, 0.02);
  }

  vec3 linearToSRGB(vec3 c) {
    return mix(
      pow(max(c, vec3(0.0)), vec3(0.41666)) * 1.055 - 0.055,
      c * 12.92,
      vec3(lessThanEqual(c, vec3(0.0031308)))
    );
  }

  void main() {
    vec2 cellId = floor(gl_FragCoord.xy / uCellPx);
    vec2 cellUv = (cellId + 0.5) / uGrid;
    vec4 scene = texture2D(uScene, cellUv);

    vec3 color = linearToSRGB(neutralToneMapping(unmix(scene) * uExposure));
    float lum = dot(color, vec3(0.2126, 0.7152, 0.0722));

    vec4 ink = texture2D(uInk, cellUv);
    float inkTone = unmix(ink).r;
    float maskCov = step(0.5, ink.a);
    float washAmt = smoothstep(0.93, 0.965, inkTone) * maskCov;
    lum = max(lum, inkTone * 0.99 * maskCov);
    lum = mix(lum, 0.97, washAmt * 0.9);

    float t = clamp((uWhite - lum) / max(uWhite - uBlack, 1e-3), 0.0, 1.0);
    float level = floor(t * uLevels + 0.5) / uLevels;

    vec2 texel = 1.0 / uGrid;
    float centerCov = smoothstep(0.3, 0.72, scene.a);
    float depth0 = texture2D(uDepth, cellUv).r;
    float minCov = 1.0;
    float maxLumStep = 0.0;
    float depthEdge = 0.0;

    for (int k = 0; k < 2; k++) {
      vec2 off = k == 0 ? vec2(texel.x, 0.0) : vec2(0.0, texel.y);
      for (int s = 0; s < 2; s++) {
        vec4 n = texture2D(uScene, cellUv + (s == 0 ? off : -off));
        float nCov = smoothstep(0.3, 0.72, n.a);
        minCov = min(minCov, nCov);
        vec3 nColor = linearToSRGB(neutralToneMapping(unmix(n) * uExposure));
        float nLum = dot(nColor, vec3(0.2126, 0.7152, 0.0722));
        maxLumStep = max(maxLumStep, abs(lum - nLum) * step(0.5, nCov));
      }
      float dA = texture2D(uDepth, cellUv + off).r - depth0;
      float dB = texture2D(uDepth, cellUv - off).r - depth0;
      float grade = min(abs(dA), abs(dB));
      depthEdge = max(depthEdge, step(0.0022 + grade * 2.5, max(dA, dB)));
    }

    float silhouette = step(0.5, centerCov) * (1.0 - step(0.5, minCov));
    float feature = step(0.18, maxLumStep) * step(0.5, centerCov);
    depthEdge *= step(0.5, centerCov);

    float line = max(silhouette, max(depthEdge * 0.85, feature * 0.5)) * (1.0 - washAmt * 0.45);
    float shade = max(level, line * uPen);
    float weight = max(level, line);

    if (uInvert > 0.5) {
      shade = 1.0 - shade;
      weight = 1.0 - weight;
    }

    vec3 mono = mix(uLight, uDark, shade);

    float peak = max(color.r, max(color.g, color.b));
    float sat = (peak - min(color.r, min(color.g, color.b))) / max(peak, 1e-3);
    vec3 outColor = mix(mono, color * mix(1.0, 0.82, shade), smoothstep(0.14, 0.42, sat));

    vec2 local = fract(gl_FragCoord.xy / uCellPx) - 0.5;
    float blockHalf = mix(uMinBlock, uMaxBlock, clamp(weight, 0.05, 0.95)) * centerCov;
    float d = max(abs(local.x), abs(local.y));
    float aa = fwidth(d) + 1e-5;
    float block = 1.0 - smoothstep(blockHalf - aa, blockHalf + aa, d);
    float blockAlpha = block * step(0.001, centerCov);

    // Subtle background graph-paper idle dots
    float h = fract(sin(dot(cellId, vec2(12.9898, 78.233))) * 43758.5453);
    float idleDot = 1.0 - smoothstep(0.04 - aa, 0.04 + aa, d);
    float idleAlpha = idleDot * step(0.65, h) * 0.4 * (1.0 - centerCov);
    vec3 idleColor = mix(uLight, uDark, 0.25);

    vec3 finalColor = mix(idleColor, outColor, step(0.001, centerCov));
    float finalAlpha = max(blockAlpha, idleAlpha);

    gl_FragColor = vec4(finalColor, finalAlpha);
  }
`;

export default function EcosystemDitherStage({
  activeStep,
  className = '',
  theme = 'dark',
}: EcosystemDitherStageProps) {
  const mountRef = useRef<HTMLDivElement>(null);
  const activeStepRef = useRef(activeStep);
  activeStepRef.current = activeStep;

  useEffect(() => {
    const container = mountRef.current;
    if (!container) return;

    let width = container.clientWidth || 800;
    let height = container.clientHeight || 500;
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    const cellPx = 3.0 * dpr;

    // 1. Renderer
    const renderer = new THREE.WebGLRenderer({
      powerPreference: 'high-performance',
      antialias: false,
      alpha: true,
      stencil: false,
    });
    renderer.setSize(width, height);
    renderer.setPixelRatio(dpr);
    renderer.outputColorSpace = THREE.LinearSRGBColorSpace;
    container.appendChild(renderer.domElement);

    // 2. Render Targets (Dual-Pass)
    const sceneTarget = new THREE.WebGLRenderTarget(width * dpr, height * dpr, {
      type: THREE.HalfFloatType,
      colorSpace: THREE.LinearSRGBColorSpace,
      minFilter: THREE.LinearFilter,
      magFilter: THREE.LinearFilter,
      samples: 4,
    });

    const depthTexture = new THREE.DepthTexture(width * dpr, height * dpr);
    const inkTarget = new THREE.WebGLRenderTarget(width * dpr, height * dpr, {
      colorSpace: THREE.LinearSRGBColorSpace,
      minFilter: THREE.LinearFilter,
      magFilter: THREE.LinearFilter,
      depthTexture: depthTexture,
    });

    // 3. 3D Scene & Perspective Camera
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(36, width / height, 0.1, 50);

    // Camera animation state
    const currentCam = {
      azimuth: STEP_CAMERA_TARGETS[0].azimuth,
      elevation: STEP_CAMERA_TARGETS[0].elevation,
      focus: new THREE.Vector3(...STEP_CAMERA_TARGETS[0].focus),
      zoom: STEP_CAMERA_TARGETS[0].zoom,
    };
    const targetCam = {
      azimuth: STEP_CAMERA_TARGETS[0].azimuth,
      elevation: STEP_CAMERA_TARGETS[0].elevation,
      focus: new THREE.Vector3(...STEP_CAMERA_TARGETS[0].focus),
      zoom: STEP_CAMERA_TARGETS[0].zoom,
    };

    // Lights
    const ambientLight = new THREE.AmbientLight(0xffffff, 1.2);
    scene.add(ambientLight);

    const dirLight = new THREE.DirectionalLight(0xffffff, 2.5);
    dirLight.position.set(5, 10, 7);
    scene.add(dirLight);

    const fillLight = new THREE.DirectionalLight(0xdbeafe, 1.0);
    fillLight.position.set(-5, 6, -5);
    scene.add(fillLight);

    // Shared box geometry
    const boxGeo = new THREE.BoxGeometry(1, 1, 1);

    // Materials
    const defaultLitMat = new THREE.MeshStandardMaterial({
      color: 0xe2e8f0,
      roughness: 0.45,
      metalness: 0.1,
    });
    const darkLitMat = new THREE.MeshStandardMaterial({
      color: 0x334155,
      roughness: 0.5,
    });
    const accentLitMat = new THREE.MeshStandardMaterial({
      color: 0xffffff,
      roughness: 0.2,
    });

    // Ink materials (unlit for Pass 2)
    const inkMatWhite = new THREE.MeshBasicMaterial({ color: 0xffffff });
    const inkMatDark = new THREE.MeshBasicMaterial({ color: 0x555555 });
    const inkMatMid = new THREE.MeshBasicMaterial({ color: 0x999999 });

    const managedMeshes: {
      mesh: THREE.Mesh | THREE.InstancedMesh;
      litMat: THREE.Material | THREE.Material[];
      inkMat: THREE.Material | THREE.Material[];
    }[] = [];

    function registerMesh(
      mesh: THREE.Mesh | THREE.InstancedMesh,
      litMat: THREE.Material | THREE.Material[],
      inkMat: THREE.Material | THREE.Material[]
    ) {
      scene.add(mesh);
      managedMeshes.push({ mesh, litMat, inkMat });
      return mesh;
    }

    // --- Build 3D Logistics Ecosystem ---
    const dummy = new THREE.Object3D();

    // 1. Central Ground Floor Grid
    const ground = new THREE.Mesh(
      new THREE.PlaneGeometry(16, 16),
      new THREE.MeshStandardMaterial({ color: 0x1e293b, roughness: 0.9 })
    );
    ground.rotation.x = -Math.PI / 2;
    ground.position.y = -0.01;
    registerMesh(ground, ground.material, inkMatDark);

    // 2. Road Grid (Transit Lanes)
    const roadX = new THREE.Mesh(
      boxGeo,
      new THREE.MeshStandardMaterial({ color: 0x0f172a, roughness: 0.8 })
    );
    roadX.scale.set(16, 0.02, 1.4);
    roadX.position.set(0, 0.01, 0);
    registerMesh(roadX, roadX.material, inkMatDark);

    const roadZ = new THREE.Mesh(
      boxGeo,
      new THREE.MeshStandardMaterial({ color: 0x0f172a, roughness: 0.8 })
    );
    roadZ.scale.set(1.4, 0.02, 16);
    roadZ.position.set(0, 0.01, 0);
    registerMesh(roadZ, roadZ.material, inkMatDark);

    // 3. Zone 0: Supplier Weighbridge & Platform (Top-Left, x: -2.2, z: -2.0)
    const supplierBase = new THREE.Mesh(boxGeo, defaultLitMat);
    supplierBase.scale.set(2.4, 0.25, 2.0);
    supplierBase.position.set(-2.2, 0.125, -2.0);
    registerMesh(supplierBase, defaultLitMat, inkMatMid);

    // Factory canopy
    const factoryRoof = new THREE.Mesh(boxGeo, darkLitMat);
    factoryRoof.scale.set(2.6, 0.1, 2.2);
    factoryRoof.position.set(-2.2, 0.9, -2.0);
    registerMesh(factoryRoof, darkLitMat, inkMatWhite);

    // Pillars
    [-3.3, -1.1].forEach((px) => {
      [-2.9, -1.1].forEach((pz) => {
        const pillar = new THREE.Mesh(boxGeo, defaultLitMat);
        pillar.scale.set(0.12, 0.8, 0.12);
        pillar.position.set(px, 0.45, pz);
        registerMesh(pillar, defaultLitMat, inkMatWhite);
      });
    });

    // Pallet stacks in Supplier zone (InstancedMesh)
    const palletCount = 12;
    const pallets = new THREE.InstancedMesh(boxGeo, defaultLitMat, palletCount);
    let pIdx = 0;
    for (let r = 0; r < 3; r++) {
      for (let c = 0; c < 4; c++) {
        dummy.position.set(-2.8 + c * 0.4, 0.35, -2.5 + r * 0.4);
        dummy.scale.set(0.32, 0.2, 0.32);
        dummy.rotation.set(0, (r + c) * 0.1, 0);
        dummy.updateMatrix();
        pallets.setMatrixAt(pIdx++, dummy.matrix);
      }
    }
    pallets.instanceMatrix.needsUpdate = true;
    registerMesh(pallets, defaultLitMat, inkMatWhite);

    // 4. Zone 1: Warehouse Cross-Dock & Racking (Center-Left, x: -0.5, z: 1.8)
    const whBuilding = new THREE.Mesh(boxGeo, darkLitMat);
    whBuilding.scale.set(2.8, 0.85, 2.4);
    whBuilding.position.set(-0.6, 0.425, 2.0);
    registerMesh(whBuilding, darkLitMat, inkMatMid);

    // High-bay Racks (InstancedMesh)
    const rackCount = 16;
    const racks = new THREE.InstancedMesh(boxGeo, defaultLitMat, rackCount);
    let rIdx = 0;
    for (let tier = 0; tier < 4; tier++) {
      for (let bay = 0; bay < 4; bay++) {
        dummy.position.set(-1.6 + bay * 0.55, 0.95 + tier * 0.28, 2.0);
        dummy.scale.set(0.48, 0.08, 1.8);
        dummy.rotation.set(0, 0, 0);
        dummy.updateMatrix();
        racks.setMatrixAt(rIdx++, dummy.matrix);
      }
    }
    racks.instanceMatrix.needsUpdate = true;
    registerMesh(racks, defaultLitMat, inkMatWhite);

    // 5. Zone 2: Retailer Network Hub & Storefront (Bottom-Right, x: 2.2, z: 1.8)
    const storeBuilding = new THREE.Mesh(boxGeo, defaultLitMat);
    storeBuilding.scale.set(2.4, 0.65, 2.2);
    storeBuilding.position.set(2.2, 0.325, 1.8);
    registerMesh(storeBuilding, defaultLitMat, inkMatWhite);

    // Retail Storefront Glass Entrance
    const storeEntrance = new THREE.Mesh(boxGeo, accentLitMat);
    storeEntrance.scale.set(0.9, 0.45, 0.1);
    storeEntrance.position.set(2.2, 0.225, 0.7);
    registerMesh(storeEntrance, accentLitMat, inkMatWhite);

    // Checkout counters
    [-0.4, 0.4].forEach((ox) => {
      const reg = new THREE.Mesh(boxGeo, darkLitMat);
      reg.scale.set(0.35, 0.3, 0.6);
      reg.position.set(2.2 + ox, 0.15, 1.4);
      registerMesh(reg, darkLitMat, inkMatMid);
    });

    // 6. Dynamic Fleet Vehicles (Moving along roads)
    interface MovingTruck {
      mesh: THREE.Group;
      axis: 'x' | 'z';
      dir: 1 | -1;
      speed: number;
      offset: number;
      lane: number;
    }

    const trucks: MovingTruck[] = [];
    const TRUCK_LANES = [
      { axis: 'x' as const, lane: 0.35, dir: 1 as const, speed: 1.4, offset: -6 },
      { axis: 'x' as const, lane: -0.35, dir: -1 as const, speed: 1.8, offset: 6 },
      { axis: 'z' as const, lane: 0.35, dir: 1 as const, speed: 1.2, offset: -5 },
      { axis: 'z' as const, lane: -0.35, dir: -1 as const, speed: 1.6, offset: 5 },
    ];

    TRUCK_LANES.forEach((spec) => {
      const truckGroup = new THREE.Group();

      // Truck cabin
      const cab = new THREE.Mesh(boxGeo, accentLitMat);
      cab.scale.set(0.3, 0.28, 0.3);
      cab.position.set(0, 0.18, spec.axis === 'x' ? 0.25 * spec.dir : 0);
      truckGroup.add(cab);

      // Cargo Container
      const cargo = new THREE.Mesh(boxGeo, defaultLitMat);
      cargo.scale.set(0.34, 0.38, 0.75);
      cargo.position.set(0, 0.23, spec.axis === 'x' ? -0.2 * spec.dir : 0);
      truckGroup.add(cargo);

      registerMesh(cab, accentLitMat, inkMatWhite);
      registerMesh(cargo, defaultLitMat, inkMatMid);
      scene.add(truckGroup);

      trucks.push({
        mesh: truckGroup,
        axis: spec.axis,
        dir: spec.dir,
        speed: spec.speed,
        offset: spec.offset,
        lane: spec.lane,
      });
    });

    // 7. Post-Processing Screen Quad
    const postScene = new THREE.Scene();
    const postCamera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);

    const isDark = theme === 'dark';
    const postMaterial = new THREE.ShaderMaterial({
      vertexShader: DITHER_VERTEX_SHADER,
      fragmentShader: DITHER_FRAGMENT_SHADER,
      uniforms: {
        uScene: { value: sceneTarget.texture },
        uInk: { value: inkTarget.texture },
        uDepth: { value: depthTexture },
        uGrid: { value: new THREE.Vector2(width / cellPx, height / cellPx) },
        uCellPx: { value: new THREE.Vector2(cellPx, cellPx) },
        uExposure: { value: 1.15 },
        uLevels: { value: 5.0 },
        uMinBlock: { value: 0.08 },
        uMaxBlock: { value: 0.44 },
        uBlack: { value: 0.22 },
        uWhite: { value: 0.88 },
        uPen: { value: 0.65 },
        uLight: { value: isDark ? new THREE.Color(0x09090b) : new THREE.Color(0xf8fafc) },
        uDark: { value: isDark ? new THREE.Color(0xffffff) : new THREE.Color(0x0b1014) },
        uInvert: { value: isDark ? 1.0 : 0.0 },
      },
      transparent: true,
      depthTest: false,
      depthWrite: false,
    });

    const quad = new THREE.Mesh(new THREE.PlaneGeometry(2, 2), postMaterial);
    postScene.add(quad);

    // 8. Resize Handler
    const handleResize = () => {
      if (!container) return;
      width = container.clientWidth;
      height = container.clientHeight;
      if (width === 0 || height === 0) return;

      renderer.setSize(width, height);
      sceneTarget.setSize(width * dpr, height * dpr);
      inkTarget.setSize(width * dpr, height * dpr);

      camera.aspect = width / height;
      camera.updateProjectionMatrix();

      postMaterial.uniforms.uGrid.value.set(width / cellPx, height / cellPx);
    };

    window.addEventListener('resize', handleResize);
    handleResize();

    // 9. Intersection Observer (Sleep loop when off-screen)
    let isVisible = true;
    const observer = new IntersectionObserver(([entry]) => {
      isVisible = entry.isIntersecting && !document.hidden;
    }, { threshold: 0.05 });
    observer.observe(container);

    const handleVisibility = () => {
      isVisible = !document.hidden;
    };
    document.addEventListener('visibilitychange', handleVisibility);

    // 10. Animation Loop with Exponential Damping
    let rafId: number;
    let lastTime = performance.now();

    const animate = (time: number) => {
      rafId = requestAnimationFrame(animate);
      if (!isVisible) return;

      const delta = Math.min((time - lastTime) / 1000, 0.1);
      lastTime = time;

      // Update camera target based on current step
      const stepIdx = Math.max(0, Math.min(STEP_CAMERA_TARGETS.length - 1, activeStepRef.current));
      const targetConfig = STEP_CAMERA_TARGETS[stepIdx];

      targetCam.azimuth = targetConfig.azimuth;
      targetCam.elevation = targetConfig.elevation;
      targetCam.focus.set(...targetConfig.focus);
      targetCam.zoom = targetConfig.zoom;

      // Exponential damping
      const damp = 1.0 - Math.exp(-delta / 0.22);
      currentCam.azimuth += (targetCam.azimuth - currentCam.azimuth) * damp;
      currentCam.elevation += (targetCam.elevation - currentCam.elevation) * damp;
      currentCam.focus.lerp(targetCam.focus, damp);
      currentCam.zoom += (targetCam.zoom - currentCam.zoom) * damp;

      // Spherical to Cartesian camera positioning
      const distance = 8.5 / Math.max(currentCam.zoom, 0.2);
      const camX =
        currentCam.focus.x +
        distance * Math.cos(currentCam.elevation) * Math.sin(currentCam.azimuth);
      const camY =
        currentCam.focus.y + distance * Math.sin(currentCam.elevation);
      const camZ =
        currentCam.focus.z +
        distance * Math.cos(currentCam.elevation) * Math.cos(currentCam.azimuth);

      camera.position.set(camX, camY, camZ);
      camera.lookAt(currentCam.focus);

      // Advance moving trucks along roads
      trucks.forEach((t) => {
        t.offset += t.speed * delta * t.dir;
        if (t.dir > 0 && t.offset > 7.5) t.offset = -7.5;
        if (t.dir < 0 && t.offset < -7.5) t.offset = 7.5;

        if (t.axis === 'x') {
          t.mesh.position.set(t.offset, 0, t.lane);
          t.mesh.rotation.y = t.dir > 0 ? 0 : Math.PI;
        } else {
          t.mesh.position.set(t.lane, 0, t.offset);
          t.mesh.rotation.y = t.dir > 0 ? Math.PI / 2 : -Math.PI / 2;
        }
      });

      // --- PASS 1: Lit Scene Render ---
      renderer.setRenderTarget(sceneTarget);
      renderer.render(scene, camera);

      // --- PASS 2: Unlit Ink + Depth Render ---
      for (const m of managedMeshes) {
        m.mesh.material = m.inkMat;
      }
      renderer.setRenderTarget(inkTarget);
      renderer.render(scene, camera);

      // Restore lit materials
      for (const m of managedMeshes) {
        m.mesh.material = m.litMat;
      }

      // --- PASS 3: Halftone Post-Processing Screen Quad ---
      renderer.setRenderTarget(null);
      renderer.render(postScene, postCamera);
    };

    rafId = requestAnimationFrame(animate);

    // 11. Cleanup
    return () => {
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', handleResize);
      document.removeEventListener('visibilitychange', handleVisibility);
      observer.disconnect();

      renderer.dispose();
      renderer.forceContextLoss();
      sceneTarget.dispose();
      inkTarget.dispose();
      depthTexture.dispose();
      postMaterial.dispose();
      quad.geometry.dispose();
      boxGeo.dispose();
      renderer.domElement.remove();
    };
  }, [theme]);

  return (
    <div
      ref={mountRef}
      className={`relative w-full h-full overflow-hidden select-none pointer-events-none ${className}`}
      aria-hidden="true"
    />
  );
}
