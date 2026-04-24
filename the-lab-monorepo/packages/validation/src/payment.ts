import { z } from "zod";

// ─── GlobalPaynt Gateway ────────────────────────────────────────────────────────
export const GlobalPayntGatewaySchema = z.enum(["CASH", "GLOBAL_PAY", "GLOBAL_PAY", "UZCARD", "CASH"]);

// ─── GlobalPaynt Session Status ─────────────────────────────────────────────────
export const GlobalPayntSessionStatusSchema = z.enum([
    "CREATED",
    "PENDING",
    "SETTLED",
    "FAILED",
    "EXPIRED",
    "CANCELLED",
]);

// ─── GlobalPaynt Attempt Status ─────────────────────────────────────────────────
export const GlobalPayntAttemptStatusSchema = z.enum([
    "INITIATED",
    "REDIRECTED",
    "PROCESSING",
    "SUCCESS",
    "FAILED",
    "CANCELLED",
    "TIMED_OUT",
]);

// ─── Card Checkout Request ──────────────────────────────────────────────────
// POST /v1/order/card-checkout
export const CardCheckoutRequestSchema = z.object({
    order_id: z.string().uuid("Invalid order ID"),
    gateway: z.enum(["CASH", "GLOBAL_PAY", "GLOBAL_PAY"], {
        errorMap: () => ({ message: "Gateway must be CASH, GLOBAL_PAY, or GLOBAL_PAY" }),
    }),
});

// ─── Cash Checkout Request ──────────────────────────────────────────────────
// POST /v1/order/cash-checkout
export const CashCheckoutRequestSchema = z.object({
    order_id: z.string().uuid("Invalid order ID"),
});

// ─── GlobalPaynt Session Response ───────────────────────────────────────────────
export const GlobalPayntSessionResponseSchema = z.object({
    session_id: z.string().uuid(),
    order_id: z.string().uuid(),
    retailer_id: z.string(),
    supplier_id: z.string(),
    gateway: GlobalPayntGatewaySchema,
    locked_amount: z.number().int().nonnegative(),
    currency: z.string().length(3),
    status: GlobalPayntSessionStatusSchema,
    current_attempt_no: z.number().int().nonnegative(),
    invoice_id: z.string().nullable(),
    redirect_url: z.string().url().nullable(),
    expires_at: z.string().datetime().nullable(),
    last_error_code: z.string().nullable(),
    last_error_message: z.string().nullable(),
    created_at: z.string().datetime(),
    updated_at: z.string().datetime().nullable(),
    settled_at: z.string().datetime().nullable(),
});

// ─── GlobalPaynt Session Query ──────────────────────────────────────────────────
// GET /v1/global_paynt/sessions?order_id=...&status=...
export const GlobalPayntSessionQuerySchema = z.object({
    order_id: z.string().uuid().optional(),
    status: GlobalPayntSessionStatusSchema.optional(),
    supplier_id: z.string().optional(),
});

// ─── Inferred Types ─────────────────────────────────────────────────────────
export type CardCheckoutRequest = z.infer<typeof CardCheckoutRequestSchema>;
export type CashCheckoutRequest = z.infer<typeof CashCheckoutRequestSchema>;
export type GlobalPayntSessionResponse = z.infer<typeof GlobalPayntSessionResponseSchema>;
export type GlobalPayntSessionQuery = z.infer<typeof GlobalPayntSessionQuerySchema>;
