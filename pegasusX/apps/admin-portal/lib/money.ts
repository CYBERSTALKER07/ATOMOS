/**
 * Deterministic minor-unit (tiyin / cents) and basis-point money utilities.
 * All parsing, discount, fee, ratio, and display operations execute in pure
 * BigInt / string integer arithmetic without IEEE-754 floating-point math.
 */

const ZERO = BigInt(0);
const TWO = BigInt(2);
const FIVE = BigInt(5);
const TEN = BigInt(10);
const FIFTY = BigInt(50);
const MINOR_SCALE = BigInt(100);
const TWO_HUNDRED = BigInt(200);
const TWO_THOUSAND = BigInt(2000);
const HALF_BPS = BigInt(5000);
const BPS_SCALE = BigInt(10000);
const MAX_SAFE_BIGINT = BigInt(Number.MAX_SAFE_INTEGER);
const MIN_SAFE_BIGINT = BigInt(Number.MIN_SAFE_INTEGER);

export interface ParseMoneyOptions {
  allowNegative?: boolean;
}

export interface FormatMinorOptions {
  currency?: string;
  currencyPosition?: "suffix" | "prefix";
  minimumFractionDigits?: 0 | 1 | 2;
  maximumFractionDigits?: 0 | 1 | 2;
  groupSeparator?: string;
  decimalSeparator?: string;
  signDisplay?: "auto" | "always";
}

function toSafeNumberOrNull(value: bigint): number | null {
  if (value < MIN_SAFE_BIGINT || value > MAX_SAFE_BIGINT) {
    return null;
  }
  const num = Number(value);
  return num === 0 ? 0 : num;
}

function toMinorBigInt(value: number | bigint | null | undefined): bigint {
  if (value === null || value === undefined) {
    return ZERO;
  }
  if (typeof value === "bigint") {
    return value;
  }
  if (!Number.isFinite(value)) {
    return ZERO;
  }
  return BigInt(Math.trunc(value));
}

function groupAsciiDigits(digits: string, separator: string): string {
  if (!separator || digits.length <= 3) {
    return digits;
  }
  const chunks: string[] = [];
  for (let i = digits.length; i > 0; i -= 3) {
    chunks.push(digits.slice(Math.max(0, i - 3), i));
  }
  chunks.reverse();
  return chunks.join(separator);
}

function stripSpaceGrouping(raw: string): string | null {
  if (!/[\s\u00A0\u202F]/.test(raw)) {
    return raw;
  }
  if (!/^\d{1,3}(?:[ \u00A0\u202F]\d{3})+$/.test(raw)) {
    return null;
  }
  return raw.replace(/[ \u00A0\u202F]/g, "");
}

export function parseMajorToMinorBigInt(
  input: string,
  options?: ParseMoneyOptions,
): bigint | null {
  if (typeof input !== "string") {
    return null;
  }
  let trimmed = input.trim();
  if (!trimmed) {
    return null;
  }

  let negative = false;
  if (trimmed.startsWith("-") || trimmed.startsWith("+")) {
    negative = trimmed[0] === "-";
    trimmed = trimmed.slice(1).trim();
  }
  if (!trimmed) {
    return null;
  }
  if (negative && !options?.allowNegative) {
    return null;
  }

  const hasComma = trimmed.includes(",");
  const hasDot = trimmed.includes(".");
  let wholeRaw = trimmed;
  let fracRaw = "";

  if (hasComma && hasDot) {
    const lastComma = trimmed.lastIndexOf(",");
    const lastDot = trimmed.lastIndexOf(".");
    if (lastDot > lastComma) {
      const dotParts = trimmed.split(".");
      if (dotParts.length !== 2) {
        return null;
      }
      if (!/^\d{1,3}(?:,\d{3})+$/.test(dotParts[0])) {
        return null;
      }
      wholeRaw = dotParts[0].replace(/,/g, "");
      fracRaw = dotParts[1];
    } else {
      const commaParts = trimmed.split(",");
      if (commaParts.length !== 2) {
        return null;
      }
      if (!/^\d{1,3}(?:\.\d{3})+$/.test(commaParts[0])) {
        return null;
      }
      wholeRaw = commaParts[0].replace(/\./g, "");
      fracRaw = commaParts[1];
    }
  } else if (hasComma) {
    const commaParts = trimmed.split(",");
    if (commaParts.length > 2) {
      if (!/^\d{1,3}(?:,\d{3})+$/.test(trimmed)) {
        return null;
      }
      wholeRaw = trimmed.replace(/,/g, "");
      fracRaw = "";
    } else {
      wholeRaw = commaParts[0];
      fracRaw = commaParts[1] ?? "";
    }
  } else if (hasDot) {
    const dotParts = trimmed.split(".");
    if (dotParts.length !== 2) {
      return null;
    }
    wholeRaw = dotParts[0];
    fracRaw = dotParts[1];
  }

  if (wholeRaw === "" && fracRaw === "") {
    return null;
  }

  const ungroupedWhole = wholeRaw === "" ? "0" : stripSpaceGrouping(wholeRaw);
  if (ungroupedWhole === null || !/^\d+$/.test(ungroupedWhole)) {
    return null;
  }
  if (fracRaw !== "" && !/^\d+$/.test(fracRaw)) {
    return null;
  }

  const fracExtra = fracRaw.slice(2);
  if (fracExtra.length > 0 && !/^0+$/.test(fracExtra)) {
    return null;
  }

  const fracTwo = fracRaw.slice(0, 2).padEnd(2, "0");
  const minor = BigInt(ungroupedWhole) * MINOR_SCALE + BigInt(fracTwo);
  if (minor === ZERO) {
    return ZERO;
  }
  return negative ? -minor : minor;
}

export function parseMajorToMinor(
  input: string,
  options?: ParseMoneyOptions,
): number | null {
  const parsed = parseMajorToMinorBigInt(input, options);
  if (parsed === null) {
    return null;
  }
  return toSafeNumberOrNull(parsed);
}

export function parseMinorUnitsInt(
  input: string | number | bigint,
  options?: ParseMoneyOptions,
): number | null {
  if (typeof input === "bigint") {
    if (input < ZERO && !options?.allowNegative) {
      return null;
    }
    return toSafeNumberOrNull(input);
  }
  if (typeof input === "number") {
    if (!Number.isSafeInteger(input)) {
      return null;
    }
    if (input < 0 && !options?.allowNegative) {
      return null;
    }
    return input === 0 ? 0 : input;
  }
  if (typeof input !== "string") {
    return null;
  }
  let trimmed = input.trim();
  if (!trimmed) {
    return null;
  }
  let negative = false;
  if (trimmed.startsWith("-") || trimmed.startsWith("+")) {
    negative = trimmed[0] === "-";
    trimmed = trimmed.slice(1).trim();
  }
  if (!trimmed || (negative && !options?.allowNegative)) {
    return null;
  }
  const normalized = trimmed.includes(",")
    ? /^\d{1,3}(?:,\d{3})+$/.test(trimmed)
      ? trimmed.replace(/,/g, "")
      : null
    : stripSpaceGrouping(trimmed);
  if (normalized === null || !/^\d+$/.test(normalized)) {
    return null;
  }
  const raw = BigInt(normalized);
  const signed = negative && raw > ZERO ? -raw : raw;
  return toSafeNumberOrNull(signed);
}

export function majorIntToMinorBigInt(major: number | bigint | string): bigint {
  if (typeof major === "bigint") {
    return major * MINOR_SCALE;
  }
  if (typeof major === "string") {
    const parsed = parseMajorToMinorBigInt(major, { allowNegative: true });
    return parsed ?? ZERO;
  }
  if (!Number.isFinite(major)) {
    return ZERO;
  }
  return BigInt(Math.trunc(major)) * MINOR_SCALE;
}

export function majorIntToMinor(major: number | bigint | string): number {
  const value = majorIntToMinorBigInt(major);
  const safe = toSafeNumberOrNull(value);
  if (safe === null) {
    throw new RangeError("Minor-unit amount exceeds Number.MAX_SAFE_INTEGER; use majorIntToMinorBigInt.");
  }
  return safe;
}

export function minorToWholeTruncBigInt(minor: number | bigint | null | undefined): bigint {
  return toMinorBigInt(minor) / MINOR_SCALE;
}

export function minorToWholeTrunc(minor: number | bigint | null | undefined): number {
  const whole = minorToWholeTruncBigInt(minor);
  const num = Number(whole);
  return num === 0 ? 0 : num;
}

export function minorToWholeRoundBigInt(minor: number | bigint | null | undefined): bigint {
  const raw = toMinorBigInt(minor);
  const negative = raw < ZERO;
  const abs = negative ? -raw : raw;
  const rounded = (abs + FIFTY) / MINOR_SCALE;
  return negative && rounded > ZERO ? -rounded : rounded;
}

export function minorToWholeRound(minor: number | bigint | null | undefined): number {
  const whole = minorToWholeRoundBigInt(minor);
  const num = Number(whole);
  return num === 0 ? 0 : num;
}

export function applyDiscountBpsBigInt(
  amountMinor: number | bigint,
  discountBps: number | bigint,
): bigint {
  const rawAmount = toMinorBigInt(amountMinor);
  const rawBps = toMinorBigInt(discountBps);
  const clampedBps = rawBps < ZERO ? ZERO : rawBps > BPS_SCALE ? BPS_SCALE : rawBps;
  const negative = rawAmount < ZERO;
  const abs = negative ? -rawAmount : rawAmount;
  const discounted = (abs * (BPS_SCALE - clampedBps) + HALF_BPS) / BPS_SCALE;
  return negative && discounted > ZERO ? -discounted : discounted;
}

export function applyDiscountBpsMinor(amountMinor: number, discountBps: number): number {
  const discounted = applyDiscountBpsBigInt(amountMinor, discountBps);
  const safe = toSafeNumberOrNull(discounted);
  if (safe === null) {
    throw new RangeError("Discounted minor-unit amount exceeds Number.MAX_SAFE_INTEGER.");
  }
  return safe;
}

export function applyBpsFeeBigInt(
  amount: number | bigint,
  bps: number | bigint,
): bigint {
  const rawAmount = toMinorBigInt(amount);
  const rawBps = toMinorBigInt(bps);
  const effectiveBps = rawBps < ZERO ? ZERO : rawBps;
  const negative = rawAmount < ZERO;
  const abs = negative ? -rawAmount : rawAmount;
  const fee = (abs * effectiveBps + HALF_BPS) / BPS_SCALE;
  return negative && fee > ZERO ? -fee : fee;
}

export function applyBpsFee(amount: number, bps: number): number {
  const fee = applyBpsFeeBigInt(amount, bps);
  const safe = toSafeNumberOrNull(fee);
  if (safe === null) {
    throw new RangeError("Fee amount exceeds Number.MAX_SAFE_INTEGER.");
  }
  return safe;
}

export function formatMinorUnits(
  minor: number | bigint | null | undefined,
  options?: FormatMinorOptions,
): string {
  const raw = toMinorBigInt(minor);
  const maxFrac = options?.maximumFractionDigits ?? 2;
  const minFrac = Math.min(options?.minimumFractionDigits ?? 2, maxFrac) as 0 | 1 | 2;
  const groupSeparator = options?.groupSeparator ?? ",";
  const decimalSeparator = options?.decimalSeparator ?? ".";
  const signDisplay = options?.signDisplay ?? "auto";

  const negative = raw < ZERO;
  const abs = negative ? -raw : raw;

  let whole: bigint;
  let fracStr = "";

  if (maxFrac === 0) {
    whole = (abs + FIFTY) / MINOR_SCALE;
  } else if (maxFrac === 1) {
    const tenths = (abs + FIVE) / TEN;
    whole = tenths / TEN;
    const rem = tenths % TEN;
    if (minFrac === 1 || rem !== ZERO) {
      fracStr = rem.toString();
    }
  } else {
    whole = abs / MINOR_SCALE;
    const rem = abs % MINOR_SCALE;
    const twoDigits = rem.toString().padStart(2, "0");
    if (minFrac === 2) {
      fracStr = twoDigits;
    } else if (minFrac === 1) {
      fracStr = twoDigits.endsWith("0") ? twoDigits.slice(0, 1) : twoDigits;
    } else {
      fracStr = twoDigits.replace(/0+$/, "");
    }
  }

  const isZeroOutput = whole === ZERO && (fracStr === "" || /^0+$/.test(fracStr));
  const signPrefix = isZeroOutput
    ? ""
    : negative
      ? "-"
      : signDisplay === "always"
        ? "+"
        : "";

  const groupedWhole = groupAsciiDigits(whole.toString(), groupSeparator);
  const numericBody = fracStr ? `${groupedWhole}${decimalSeparator}${fracStr}` : groupedWhole;
  const signedNumber = `${signPrefix}${numericBody}`;

  const currency = options?.currency?.trim();
  if (!currency) {
    return signedNumber;
  }
  if (options?.currencyPosition === "prefix") {
    return `${signPrefix}${currency} ${numericBody}`;
  }
  return `${signedNumber} ${currency}`;
}

export function formatMinorFixed(
  minor: number | bigint | null | undefined,
  options?: { signDisplay?: "auto" | "always"; currency?: string },
): string {
  return formatMinorUnits(minor, {
    groupSeparator: "",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    signDisplay: options?.signDisplay,
    currency: options?.currency,
  });
}

export function formatBpsPercent(
  bps: number | bigint | null | undefined,
  fractionDigits: 0 | 1 | 2 = 2,
): string {
  return formatMinorUnits(bps, {
    groupSeparator: "",
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  });
}

export function formatRatioPercentTenths(
  numerator: number | bigint,
  denominator: number | bigint,
): string {
  const den = toMinorBigInt(denominator);
  if (den <= ZERO) {
    return "0.0";
  }
  const num = toMinorBigInt(numerator);
  const negative = num < ZERO;
  const absNum = negative ? -num : num;
  const tenths = (absNum * TWO_THOUSAND + den) / (TWO * den);
  const whole = tenths / TEN;
  const frac = tenths % TEN;
  const prefix = negative && tenths > ZERO ? "-" : "";
  return `${prefix}${whole.toString()}.${frac.toString()}`;
}

export function ratioPercentInt(
  numerator: number | bigint,
  denominator: number | bigint,
  capAtHundred = true,
): number {
  const den = toMinorBigInt(denominator);
  if (den <= ZERO) {
    return 0;
  }
  const num = toMinorBigInt(numerator);
  if (num <= ZERO) {
    return 0;
  }
  const pct = (num * TWO_HUNDRED + den) / (TWO * den);
  if (capAtHundred && pct > MINOR_SCALE) {
    return 100;
  }
  return Number(pct);
}

export function formatGroupedInteger(
  value: number | bigint | null | undefined,
  groupSeparator = ",",
): string {
  const raw = toMinorBigInt(value);
  const negative = raw < ZERO;
  const abs = negative ? -raw : raw;
  const grouped = groupAsciiDigits(abs.toString(), groupSeparator);
  return negative && abs > ZERO ? `-${grouped}` : grouped;
}
