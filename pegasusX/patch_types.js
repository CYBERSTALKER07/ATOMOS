const fs = require('fs');
let code = fs.readFileSync('packages/ui-maps/src/GenericFleetLiveMap.tsx', 'utf8');

code = code.replace(
  /export interface GenericRouteType \{/,
  \`export interface GenericRouteType {
  driver_location?: { lat: number; lng: number };
  live_location_available?: boolean;
  location_stale?: boolean;\`
);

code = code.replace(
  /const source = geometry\.source;/g,
  'const source = (geometry as any).source;'
);

code = code.replace(
  /origin: \[source\.lng, source\.lat\]/g,
  'origin: [(source as any).lng, (source as any).lat]'
);

code = code.replace(
  /void import\('maplibre-gl\\/dist\\/maplibre-gl.css'\);/g,
  '// @ts-ignore\n    void import(\'maplibre-gl/dist/maplibre-gl.css\');'
);

fs.writeFileSync('packages/ui-maps/src/GenericFleetLiveMap.tsx', code);
