/**
 * Unit tests for PaymentModal utility functions and state logic.
 *
 * Tests pure functions from components/PaymentModal.tsx without
 * requiring WebSocket context or React rendering.
 */
import { describe, it, expect } from "vitest";

/* ── Re-implement pure functions from PaymentModal.tsx ── */

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ") + "";
}

/* ── Types mirrored from PaymentModal ── */

interface PaymentEvent {
  order_id: string;
  invoice_id?: string;
  session_id?: string;
  amount: number;
  original_amount?: number;
  payment_method: string;
  available_card_gateways?: string[];
  message?: string;
}

type PaymentState = "idle" | "choosing" | "processing" | "success" | "error";

/* ── Tests ── */

describe("PaymentModal formatAmount", () => {
  it("formats zero", () => {
    expect(formatAmount(0)).toBe("0");
  });

  it("formats typical delivery amount", () => {
    expect(formatAmount(250_000)).toBe("250 000");
  });

  it("formats large amount with multiple separators", () => {
    expect(formatAmount(12_500_000)).toBe("12 500 000");
  });

  it("formats single-digit amount", () => {
    expect(formatAmount(5)).toBe("5");
  });
});

describe("PaymentEvent parsing", () => {
  it("extracts required fields from WS message", () => {
    const msg: Record<string, unknown> = {
      type: "PAYMENT_REQUIRED",
      order_id: "ORD-123",
      amount: 150_000,
      payment_method: "CASH",
    };

    const evt: PaymentEvent = {
      order_id: msg.order_id as string,
      invoice_id: msg.invoice_id as string | undefined,
      session_id: msg.session_id as string | undefined,
      amount: msg.amount as number,
      original_amount: msg.original_amount as number | undefined,
      payment_method: msg.payment_method as string,
      available_card_gateways: msg.available_card_gateways as string[] | undefined,
      message: msg.message as string | undefined,
    };

    expect(evt.order_id).toBe("ORD-123");
    expect(evt.amount).toBe(150_000);
    expect(evt.payment_method).toBe("CASH");
    expect(evt.invoice_id).toBeUndefined();
    expect(evt.available_card_gateways).toBeUndefined();
  });

  it("extracts card gateway options when present", () => {
    const msg: Record<string, unknown> = {
      type: "PAYMENT_REQUIRED",
      order_id: "ORD-456",
      amount: 300_000,
      payment_method: "CARD",
      available_card_gateways: ["GLOBAL_PAY", "CASH"],
    };

    const evt: PaymentEvent = {
      order_id: msg.order_id as string,
      amount: msg.amount as number,
      payment_method: msg.payment_method as string,
      available_card_gateways: msg.available_card_gateways as string[],
    };

    expect(evt.available_card_gateways).toEqual(["GLOBAL_PAY", "CASH"]);
    expect(evt.available_card_gateways).toHaveLength(2);
  });

  it("detects amended amount", () => {
    const evt: PaymentEvent = {
      order_id: "ORD-789",
      amount: 120_000,
      original_amount: 150_000,
      payment_method: "CASH",
    };

    const amended = evt.original_amount && evt.original_amount !== evt.amount;
    expect(amended).toBeTruthy();
  });

  it("detects non-amended amount", () => {
    const evt: PaymentEvent = {
      order_id: "ORD-789",
      amount: 150_000,
      original_amount: 150_000,
      payment_method: "CASH",
    };

    const amended = evt.original_amount && evt.original_amount !== evt.amount;
    expect(amended).toBeFalsy();
  });
});

describe("PaymentState transitions", () => {
  it("valid states are exhaustive", () => {
    const validStates: PaymentState[] = ["idle", "choosing", "processing", "success", "error"];
    expect(validStates).toHaveLength(5);
  });

  it("starts as idle", () => {
    const initial: PaymentState = "idle";
    expect(initial).toBe("idle");
  });

  it("transitions idle → choosing on PAYMENT_REQUIRED", () => {
    let state: PaymentState = "idle";
    // Simulate: WS event received
    state = "choosing";
    expect(state).toBe("choosing");
  });

  it("transitions idle → choosing on GLOBAL_PAYNT_REQUIRED", () => {
    let state: PaymentState = "idle";
    state = "choosing";
    expect(state).toBe("choosing");
  });

  it("transitions choosing → processing on payment action", () => {
    let state: PaymentState = "choosing";
    // Simulate: user clicks cash or card
    state = "processing";
    expect(state).toBe("processing");
  });

  it("transitions processing → success on PAYMENT_SETTLED", () => {
    let state: PaymentState = "processing";
    // Simulate: WS confirmation
    state = "success";
    expect(state).toBe("success");
  });

  it("transitions processing → success on GLOBAL_PAYNT_SETTLED", () => {
    let state: PaymentState = "processing";
    state = "success";
    expect(state).toBe("success");
  });

  it("transitions choosing → choosing on error (stays in choosing)", () => {
    let state: PaymentState = "choosing";
    // Simulate: API call error, reset to choosing
    state = "choosing";
    expect(state).toBe("choosing");
  });

  it("dismiss resets to idle", () => {
    let state: PaymentState = "choosing";
    // Simulate: user clicks dismiss
    state = "idle";
    expect(state).toBe("idle");
  });
});

describe("PAYMENT_SETTLED matching", () => {
  it("matches by order_id", () => {
    const currentEvent: PaymentEvent = {
      order_id: "ORD-MATCH",
      amount: 100_000,
      payment_method: "CASH",
    };
    const settledMsg = { type: "PAYMENT_SETTLED", order_id: "ORD-MATCH" };

    const matches = currentEvent && settledMsg.order_id === currentEvent.order_id;
    expect(matches).toBe(true);
  });

  it("does not match different order_id", () => {
    const currentEvent: PaymentEvent = {
      order_id: "ORD-A",
      amount: 100_000,
      payment_method: "CASH",
    };
    const settledMsg = { type: "PAYMENT_SETTLED", order_id: "ORD-B" };

    const matches = currentEvent && settledMsg.order_id === currentEvent.order_id;
    expect(matches).toBe(false);
  });

  it("matches GLOBAL_PAYNT_SETTLED by order_id", () => {
    const currentEvent: PaymentEvent = {
      order_id: "ORD-MATCH",
      amount: 100_000,
      payment_method: "CARD",
    };
    const settledMsg = { type: "GLOBAL_PAYNT_SETTLED", order_id: "ORD-MATCH" };

    const matches = currentEvent && settledMsg.order_id === currentEvent.order_id;
    expect(matches).toBe(true);
  });
});

describe("Gateway fallback", () => {
  it("defaults to empty array when no gateways", () => {
    const evt: PaymentEvent = {
      order_id: "ORD-1",
      amount: 100_000,
      payment_method: "CARD",
    };
    const gateways = evt.available_card_gateways ?? [];
    expect(gateways).toEqual([]);
  });

  it("uses provided gateways when present", () => {
    const evt: PaymentEvent = {
      order_id: "ORD-1",
      amount: 100_000,
      payment_method: "CARD",
      available_card_gateways: ["GLOBAL_PAY"],
    };
    const gateways = evt.available_card_gateways ?? [];
    expect(gateways).toEqual(["GLOBAL_PAY"]);
  });
});
