/**
 * Strict integer money formatter for Uzbekistani So'm (UZS).
 * Ledger and wire contracts carry amounts in minor units (tiyin: 100 tiyin = 1 so'm).
 */

const ZERO = BigInt(0);

/** Number of minor units (tiyin) per 1 major unit (so'm). */
export const TIYIN_PER_UZS: bigint = BigInt(100);

function toStrictMinorBigInt(minor: number | bigint): bigint {
  if (typeof minor === "bigint") {
    return minor;
  }
  if (typeof minor === "number") {
    if (!Number.isSafeInteger(minor)) {
      throw new RangeError(
        "formatUZS expects a finite safe-integer minor amount in tiyin (use bigint for values beyond Number.MAX_SAFE_INTEGER)",
      );
    }
    return BigInt(minor === 0 ? 0 : minor);
  }
  throw new TypeError("formatUZS expects minor units as a safe-integer number or bigint");
}

function groupDigitsAsciiSpace(digits: string): string {
  if (digits.length <= 3) {
    return digits;
  }
  const groups: string[] = [];
  for (let i = digits.length; i > 0; i -= 3) {
    groups.push(digits.slice(Math.max(0, i - 3), i));
  }
  groups.reverse();
  return groups.join(" ");
}

/**
 * Formats integer minor units (tiyin) into whole UZS `so'm` using deterministic BigInt division.
 * Enforces strict `number | bigint` minor-unit input and never displays raw tiyin as `so'm`.
 */
export function formatUZS(minor: number | bigint): string {
  const raw = toStrictMinorBigInt(minor);
  const negative = raw < ZERO;
  const absMinor = negative ? -raw : raw;
  const absWhole = absMinor / TIYIN_PER_UZS;
  const sign = negative && absWhole > ZERO ? "-" : "";
  return `${sign}${groupDigitsAsciiSpace(absWhole.toString())} so'm`;
}
