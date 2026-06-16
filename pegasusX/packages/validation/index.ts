import { z } from "zod";

export const supplierLoginSchema = z.object({
  phone: z.string().trim().min(8).max(32),
  password: z.string().min(6).max(128),
});

export const supplierRegisterAccountSchema = z.object({
  phone: z.string().trim().min(8).max(32),
  password: z.string().min(8).max(128),
  contact_name: z.string().trim().min(1).max(120),
  email: z.string().trim().email().max(254),
});

export type SupplierLoginInput = z.infer<typeof supplierLoginSchema>;
export type SupplierRegisterAccountInput = z.infer<typeof supplierRegisterAccountSchema>;

export type NormalizeEanResult =
  | { ok: true; code: string }
  | { ok: false; error: string };

function validGtinChecksum(code: string): boolean {
  const n = code.length;
  if (n < 8) return false;
  let sum = 0;
  for (let i = 0; i < n - 1; i++) {
    const d = Number(code[i]);
    const posFromRight = n - 1 - i;
    sum += posFromRight % 2 === 1 ? d * 3 : d;
  }
  const check = Number(code[n - 1]);
  return (10 - (sum % 10)) % 10 === check;
}

/** Strip non-digits and validate EAN-8/12/13/14 GTIN checksum (matches backend returns.NormalizeBarcode). */
export function normalizeEanBarcode(raw: string): NormalizeEanResult {
  const digits = raw.replace(/\D/g, "");
  if (!digits) return { ok: false, error: "barcode_required" };
  if (![8, 12, 13, 14].includes(digits.length)) {
    return { ok: false, error: "unsupported_barcode_length" };
  }
  if (!validGtinChecksum(digits)) {
    return { ok: false, error: "invalid_barcode_checksum" };
  }
  return { ok: true, code: digits };
}
