import { describe, expect, it } from 'vitest';
import {
  attachTruckStatus,
  BOARD_MANIFEST_STATES,
  groupBoardColumns,
  unassignedTrucks,
  type BoardTruck,
} from '../utils/manifestBoard';

describe('GS-U7 payload board', () => {
  it('groups trucks into four manifest-state columns', () => {
    const trucks: BoardTruck[] = [
      { id: 'd', label: 'D', license_plate: '', vehicle_class: 'TRUCK', truck_status: 'DRAFT' },
      { id: 'l', label: 'L', license_plate: '', vehicle_class: 'TRUCK', truck_status: 'LOADING' },
      { id: 's', label: 'S', license_plate: '', vehicle_class: 'TRUCK', truck_status: 'SEALED' },
      { id: 'x', label: 'X', license_plate: '', vehicle_class: 'TRUCK', truck_status: 'DISPATCHED' },
      { id: 'done', label: 'Z', license_plate: '', vehicle_class: 'TRUCK', truck_status: 'COMPLETED' },
      { id: 'none', label: 'N', license_plate: '', vehicle_class: 'TRUCK' },
    ];
    const cols = groupBoardColumns(trucks);
    expect(Object.keys(cols)).toEqual([...BOARD_MANIFEST_STATES]);
    expect(cols.DRAFT.map((t) => t.id)).toEqual(['d']);
    expect(cols.LOADING.map((t) => t.id)).toEqual(['l']);
    expect(cols.SEALED.map((t) => t.id)).toEqual(['s']);
    expect(cols.DISPATCHED.map((t) => t.id)).toEqual(['x']);
    expect(unassignedTrucks(trucks).map((t) => t.id)).toEqual(['done', 'none']);
  });

  it('attaches manifest state and does not invent DRAFT from COMPLETED', () => {
    const attached = attachTruckStatus(
      [
        { id: 'veh-1', label: 'A', license_plate: '', vehicle_class: 'TRUCK' },
        { id: 'veh-2', label: 'B', license_plate: '', vehicle_class: 'TRUCK' },
      ],
      [
        { vehicle_id: 'veh-1', state: 'SEALED', total_volume_vu: 8, max_volume_vu: 40, stop_count: 2 },
        { vehicle_id: 'veh-2', state: 'COMPLETED' },
      ],
    );
    expect(attached[0].truck_status).toBe('SEALED');
    expect(attached[0].used_volume_vu).toBe(8);
    expect(attached[1].truck_status ?? '').toBe('');
    const cols = groupBoardColumns(attached);
    expect(cols.SEALED).toHaveLength(1);
    expect(cols.DRAFT).toHaveLength(0);
  });

  it('empty board keeps four empty columns', () => {
    const cols = groupBoardColumns([]);
    expect(BOARD_MANIFEST_STATES.every((s) => cols[s].length === 0)).toBe(true);
  });
});
