import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(process.cwd(), 'patent-dossier');

const surfaceTitleOverrides = new Map([
  ['web-auth-login', 'Supplier Login'],
  ['web-auth-register', 'Supplier Register'],
]);

function readJson(relativePath) {
  return JSON.parse(fs.readFileSync(path.join(root, relativePath), 'utf8'));
}

function walkJson(directoryPath) {
  const output = [];
  for (const entry of fs.readdirSync(directoryPath, { withFileTypes: true })) {
    const fullPath = path.join(directoryPath, entry.name);
    if (entry.isDirectory()) {
      output.push(...walkJson(fullPath));
    } else if (entry.name.endsWith('.json')) {
      output.push(fullPath);
    }
  }
  return output;
}

function flattenLabels(items) {
  if (!Array.isArray(items)) {
    return [];
  }
  return items
    .map((item) => {
      if (typeof item === 'string') {
        return item;
      }
      if (item && typeof item === 'object') {
        return item.button || item.icon || item.flowId || item.zoneId || item.label || item.title || null;
      }
      return null;
    })
    .filter(Boolean);
}

function humanizeSurfaceId(surfaceId) {
  if (surfaceTitleOverrides.has(surfaceId)) {
    return surfaceTitleOverrides.get(surfaceId);
  }

  return surfaceId
    .replace(/^fig-/, '')
    .replace(/^(ios|android|web|payload)-/, '')
    .replace(/-/g, ' ')
    .replace(/\b\w/g, (match) => match.toUpperCase());
}

function silhouetteFor(surfaceId) {
  if (surfaceId.startsWith('web-')) {
    return 'desktop or laptop outline';
  }
  if (surfaceId.startsWith('payload-')) {
    return 'landscape tablet outline';
  }
  return 'portrait phone outline';
}

const surfaceMap = new Map();
for (const absoluteFilePath of walkJson(root)) {
  const relativePath = path.relative(root, absoluteFilePath).replaceAll(path.sep, '/');
  const json = JSON.parse(fs.readFileSync(absoluteFilePath, 'utf8'));

  if (json.pageId) {
    surfaceMap.set(json.pageId, { ...json, artifactRef: relativePath });
  }

  if (Array.isArray(json.surfaces)) {
    for (const surface of json.surfaces) {
      const pageId = surface.pageId || surface.surfaceId;
      if (!pageId) {
        continue;
      }
      surfaceMap.set(pageId, {
        ...surface,
        artifactRef: relativePath,
        appId: json.appId,
        platform: json.platform,
        role: json.role,
        bundleId: json.bundleId,
      });
    }
  }
}

const capabilityEntries = [...surfaceMap.entries()]
  .map(([surfaceId, surface]) => {
    const explicitMinifeatures = Array.isArray(surface.minifeatures) ? surface.minifeatures : [];
    const derivedMinifeatures = [
      ...flattenLabels(surface.buttonPlacements),
      ...flattenLabels(surface.iconPlacements),
      ...flattenLabels(surface.interactiveFlows),
      ...flattenLabels(surface.stateVariants),
    ].slice(0, 12);

    return {
      surfaceId,
      artifactRef: surface.artifactRef,
      appId: surface.appId || null,
      platform: surface.platform || null,
      role: surface.role || null,
      purpose: surface.purpose || null,
      minifeatureSource: explicitMinifeatures.length ? 'explicit' : 'derived-from-dossier-structure',
      minifeatureCount: explicitMinifeatures.length || derivedMinifeatures.length,
      minifeatures: explicitMinifeatures.length ? explicitMinifeatures : derivedMinifeatures,
    };
  })
  .sort((left, right) => left.surfaceId.localeCompare(right.surfaceId));

const capabilityIndex = {
  generatedAt: '2026-04-06',
  purpose: 'Surface-level capability and minifeature index derived from page dossiers and bundled mobile dossiers.',
  totalSurfaceCount: capabilityEntries.length,
  entries: capabilityEntries,
};

fs.writeFileSync(
  path.join(root, 'counsel', 'surface-capability-index.json'),
  JSON.stringify(capabilityIndex, null, 2) + '\n',
);

const queue = readJson('figure-shot-queue.json');
const claimFamilies = readJson('counsel/claim-families.json');
const figureGroups = readJson('counsel/figure-groupings.json');
const queueShotIds = new Set(queue.shots.map((shot) => shot.shotId));

const missingShotRefs = [];
for (const group of figureGroups.groups) {
  for (const shotId of group.shotRefs) {
    if (!queueShotIds.has(shotId)) {
      missingShotRefs.push({ groupId: group.groupId, shotId });
    }
  }
}

if (missingShotRefs.length) {
  console.error('Figure grouping references missing queue shots:');
  for (const missingRef of missingShotRefs) {
    console.error(`${missingRef.groupId}: ${missingRef.shotId}`);
  }
  process.exit(1);
}

const figureCatalogShots = queue.shots.map((shot, index) => {
  const surface = surfaceMap.get(shot.surfaceId) || {};
  const claimFamilyHints = claimFamilies.families
    .filter((family) => Array.isArray(family.surfaceRefs) && family.surfaceRefs.includes(shot.surfaceId))
    .map((family) => family.familyId);

  const caption = `FIG. ${index + 1}. ${humanizeSurfaceId(shot.surfaceId)} showing ${shot.focus}.`;
  const prompt = [
    'Create a patent-style monochrome line-art figure.',
    `Subject: ${surface.purpose || humanizeSurfaceId(shot.surfaceId)}.`,
    `Primary focus: ${shot.focus}.`,
    `View type: ${shot.viewType}.`,
    `Use a ${silhouetteFor(shot.surfaceId)}.`,
    'Render the active interface in clean black contour lines on a white background.',
    'If the view contains a sheet, drawer, modal, or overlay, keep the background context lighter or dashed.',
    'Avoid gradients, realistic textures, photographic shading, and human facial detail.',
    'Use simplified interface blocks, icons, labels, and reference numerals suitable for patent drawings.',
  ].join(' ');

  return {
    figureNumber: index + 1,
    shotId: shot.shotId,
    surfaceId: shot.surfaceId,
    artifactRef: shot.artifactRef,
    viewType: shot.viewType,
    claimFamilyHints,
    caption,
    prompt,
  };
});

const figureCatalog = {
  generatedAt: '2026-04-06',
  purpose: 'Per-shot figure generation prompts and captions derived from the live figure queue and dossier corpus.',
  renderingProfileRef: queue.renderingProfileRef,
  renderingMode: queue.renderingMode,
  totalFigures: figureCatalogShots.length,
  shots: figureCatalogShots,
};

fs.writeFileSync(
  path.join(root, 'figure-production-catalog.json'),
  JSON.stringify(figureCatalog, null, 2) + '\n',
);

console.log(`Wrote counsel/surface-capability-index.json with ${capabilityEntries.length} surfaces.`);
console.log(`Wrote figure-production-catalog.json with ${figureCatalogShots.length} figures.`);