export const BOARD_MANIFEST_STATES = ['DRAFT', 'LOADING', 'SEALED', 'DISPATCHED'] as const;
export type BoardManifestState = (typeof BOARD_MANIFEST_STATES)[number];

export type BoardTruck = {
  id: string;
  label: string;
  license_plate: string;
  vehicle_class: string;
  truck_status?: string;
  used_volume_vu?: number;
  max_volume_vu?: number;
  stop_count?: number;
};

export type BoardManifest = {
  vehicle_id?: string;
  truck_id?: string;
  state?: string;
  total_volume_vu?: number;
  max_volume_vu?: number;
  stop_count?: number;
  created_at?: string;
};

export function canonicalBoardState(state?: string): BoardManifestState | '' {
  const s = (state ?? '').trim().toUpperCase();
  return (BOARD_MANIFEST_STATES as readonly string[]).includes(s) ? (s as BoardManifestState) : '';
}

export function attachTruckStatus(trucks: BoardTruck[], manifests: BoardManifest[]): BoardTruck[] {
  return trucks.map((truck) => {
    if (canonicalBoardState(truck.truck_status)) return truck;
    const match = manifests
      .filter((m) => (m.vehicle_id === truck.id || m.truck_id === truck.id) && canonicalBoardState(m.state))
      .sort((a, b) => (a.created_at ?? '').localeCompare(b.created_at ?? ''))
      .at(-1);
    if (!match) return truck;
    return {
      ...truck,
      truck_status: canonicalBoardState(match.state),
      used_volume_vu: match.total_volume_vu,
      max_volume_vu: match.max_volume_vu,
      stop_count: match.stop_count,
    };
  });
}

export function groupBoardColumns(trucks: BoardTruck[]): Record<BoardManifestState, BoardTruck[]> {
  const cols = {
    DRAFT: [] as BoardTruck[],
    LOADING: [] as BoardTruck[],
    SEALED: [] as BoardTruck[],
    DISPATCHED: [] as BoardTruck[],
  };
  for (const truck of trucks) {
    const state = canonicalBoardState(truck.truck_status);
    if (state) cols[state].push(truck);
  }
  return cols;
}

export function unassignedTrucks(trucks: BoardTruck[]): BoardTruck[] {
  return trucks.filter((t) => !canonicalBoardState(t.truck_status));
}
