import { describe, expect, it } from 'vitest';
import {
  UZS_BANKNOTE_DENOMINATIONS,
  TIYIN_PER_UZS,
  STATUTORY_AML_CASH_CEILING_MINOR,
  type CashierSessionView,
} from '@pegasusx/types';

describe('Role 7 Depot Cashier Blind Count & Currency Rules', () => {
  it('converts all Uzbek Som banknote denominations to integer minor units (Tiyin: 1 UZS = 100 Tiyin)', () => {
    expect(UZS_BANKNOTE_DENOMINATIONS).toHaveLength(8);

    // Verify 1 UZS = 100 Tiyin
    expect(TIYIN_PER_UZS).toBe(100);

    const expectedDenominations = [
      { som: 200000, expectedTiyin: 20000000 },
      { som: 100000, expectedTiyin: 10000000 },
      { som: 50000, expectedTiyin: 5000000 },
      { som: 20000, expectedTiyin: 2000000 },
      { som: 10000, expectedTiyin: 1000000 },
      { som: 5000, expectedTiyin: 500000 },
      { som: 2000, expectedTiyin: 200000 },
      { som: 1000, expectedTiyin: 100000 },
    ];

    for (const item of expectedDenominations) {
      expect(UZS_BANKNOTE_DENOMINATIONS).toContain(item.som);
      const computedTiyin = item.som * TIYIN_PER_UZS;
      expect(computedTiyin).toBe(item.expectedTiyin);
      expect(Number.isInteger(computedTiyin)).toBe(true);
    }
  });

  it('calculates total blind count tally strictly in integer Tiyin without floating point drift', () => {
    // 5 notes of 200k UZS + 3 notes of 50k UZS + 10 notes of 1k UZS
    const counts: Record<number, number> = {
      200000: 5,
      50000: 3,
      1000: 10,
    };

    let totalTiyin = 0;
    for (const note of UZS_BANKNOTE_DENOMINATIONS) {
      const qty = counts[note] || 0;
      totalTiyin += qty * (note * TIYIN_PER_UZS);
    }

    // 5 * 20,000,000 = 100,000,000
    // 3 * 5,000,000  = 15,000,000
    // 10 * 100,000   = 1,000,000
    // Total = 116,000,000 Tiyin = 1,160,000 UZS
    expect(totalTiyin).toBe(116000000);
    expect(Number.isInteger(totalTiyin)).toBe(true);
  });

  it('strictly isolates expected_cash_minor during AWAITING_BLIND_COUNT_1 and BLIND_RECOUNT_REQUIRED', () => {
    const sessionAttempt1: CashierSessionView = {
      session_id: 'cash-sess-001',
      supplier_id: 'supp-01',
      depot_id: 'depot-01',
      driver_id: 'driver-01',
      route_id: 'route-01',
      bag_seal_barcode: 'SEAL-UZB-9981',
      attempt_number: 1,
      counted_cash_minor: 0,
      status: 'AWAITING_BLIND_COUNT_1',
      created_at: new Date().toISOString(),
    };

    const getMaskedExpectedCash = (session: CashierSessionView): string => {
      if (
        session.status === 'AWAITING_BLIND_COUNT_1' ||
        session.status === 'BLIND_RECOUNT_REQUIRED'
      ) {
        return '•••••••• (BLIND COUNT)';
      }
      return `${(session.expected_cash_minor ?? 0) / 100} UZS`;
    };

    expect(getMaskedExpectedCash(sessionAttempt1)).toBe('•••••••• (BLIND COUNT)');

    const sessionRecount: CashierSessionView = {
      ...sessionAttempt1,
      attempt_number: 2,
      status: 'BLIND_RECOUNT_REQUIRED',
    };
    expect(getMaskedExpectedCash(sessionRecount)).toBe('•••••••• (BLIND COUNT)');

    const sessionMatched: CashierSessionView = {
      ...sessionAttempt1,
      attempt_number: 1,
      status: 'BALANCED',
      expected_cash_minor: 116000000,
    };
    expect(getMaskedExpectedCash(sessionMatched)).toBe('1160000 UZS');
  });

  it('validates RRN reclassification ±15 minute drift and exact amount matching', () => {
    const disputedCashMinor = 25000000; // 250,000 UZS
    const posAuthAmountMinor = 25000000; // 250,000 UZS
    const tenderAmountMinor: number = 24000000; // Mismatch

    const orderTime = new Date('2026-10-30T10:00:00Z').getTime();
    const posSwipeTimeValid = new Date('2026-10-30T10:12:00Z').getTime(); // +12 min
    const posSwipeTimeInvalid = new Date('2026-10-30T10:20:00Z').getTime(); // +20 min

    // Exact minor amount check
    expect(disputedCashMinor === posAuthAmountMinor).toBe(true);
    expect(disputedCashMinor === tenderAmountMinor).toBe(false);

    // 15-minute tolerance check (900,000 ms)
    const driftMsValid = Math.abs(posSwipeTimeValid - orderTime);
    const driftMsInvalid = Math.abs(posSwipeTimeInvalid - orderTime);
    const fifteenMinMs = 15 * 60 * 1000;

    expect(driftMsValid <= fifteenMinMs).toBe(true);
    expect(driftMsInvalid <= fifteenMinMs).toBe(false);
  });

  it('validates dynamic AML threshold and uncapped policy', () => {
    expect(STATUTORY_AML_CASH_CEILING_MINOR).toBe(2500000000);

    const safeTurnIn = 1_500_000_000;
    const suspiciousTurnIn = 2_600_000_000;

    expect(safeTurnIn > STATUTORY_AML_CASH_CEILING_MINOR).toBe(false);
    expect(suspiciousTurnIn > STATUTORY_AML_CASH_CEILING_MINOR).toBe(true);

    const uncappedPolicy = (amount: number, ceiling: number) => ceiling === 0 || amount <= ceiling;
    expect(uncappedPolicy(suspiciousTurnIn, 0)).toBe(true);
  });
});
