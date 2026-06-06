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
