import { z } from "zod";

// ─── Payment Gateway ────────────────────────────────────────────────────────
export const PaymentGatewaySchema = z.enum(["CLICK", "PAYME", "GLOBAL_PAY", "UZCARD", "CASH"]);

// ─── Payment Session Status ─────────────────────────────────────────────────
export const PaymentSessionStatusSchema = z.enum([
    "CREATED",
    "PENDING",
    "SETTLED",
    "FAILED",
    "EXPIRED",
    "CANCELLED",
]);

// ─── Payment Attempt Status ─────────────────────────────────────────────────
export const PaymentAttemptStatusSchema = z.enum([
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
    gateway: z.enum(["CLICK", "PAYME", "GLOBAL_PAY"], {
        errorMap: () => ({ message: "Gateway must be CLICK, PAYME, or GLOBAL_PAY" }),
    }),
});

// ─── Cash Checkout Request ──────────────────────────────────────────────────
// POST /v1/order/cash-checkout
export const CashCheckoutRequestSchema = z.object({
    order_id: z.string().uuid("Invalid order ID"),
});

// ─── Payment Session Response ───────────────────────────────────────────────
export const PaymentSessionResponseSchema = z.object({
    session_id: z.string().uuid(),
    order_id: z.string().uuid(),
    retailer_id: z.string(),
    supplier_id: z.string(),
    gateway: PaymentGatewaySchema,
    locked_amount: z.number().int().nonnegative(),
    currency: z.string().length(3),
    status: PaymentSessionStatusSchema,
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

// ─── Payment Session Query ──────────────────────────────────────────────────
// GET /v1/payment/sessions?order_id=...&status=...
export const PaymentSessionQuerySchema = z.object({
    order_id: z.string().uuid().optional(),
    status: PaymentSessionStatusSchema.optional(),
    supplier_id: z.string().optional(),
});

// ─── Inferred Types ─────────────────────────────────────────────────────────
export type CardCheckoutRequest = z.infer<typeof CardCheckoutRequestSchema>;
export type CashCheckoutRequest = z.infer<typeof CashCheckoutRequestSchema>;
export type PaymentSessionResponse = z.infer<typeof PaymentSessionResponseSchema>;
export type PaymentSessionQuery = z.infer<typeof PaymentSessionQuerySchema>;
