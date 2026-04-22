/**
 * @file lib/money.ts
 * Shared monetary formatting utilities for the Admin Portal.
 * Currency-agnostic — formats any amount with the appropriate symbol.
 */

const CURRENCY_SYMBOLS: Record<string, string> = {
  UZS: "so'm",
  USD: '$',
  EUR: '€',
  RUB: '₽',
  KZT: '₸',
  GBP: '£',
};

/**
 * Format a monetary amount for display.
 * @param amount  Integer amount in minor units (e.g. UZS has 0 decimals, so 50000 → "50,000")
 * @param currency ISO 4217 code — defaults to "" for backward compatibility
 */
export function formatAmount(amount: number, currency = 'UZS'): string {
  const symbol = CURRENCY_SYMBOLS[currency] ?? currency;
  const formatted = new Intl.NumberFormat('en-US').format(amount);
  return `${formatted} ${symbol}`;
}

/**
 * Format a monetary amount without the currency suffix (just the number).
 */
export function formatNumber(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}
