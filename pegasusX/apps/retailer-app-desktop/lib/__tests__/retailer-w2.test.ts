import { describe, expect, it } from "vitest";
import {
  formatWholeSom,
  minorToSomInput,
  parseClaimErrorMessage,
  parseOwnerApprovalRequired,
  somInputToMinor,
  W2_COPY,
} from "../retailer-w2";

describe("retailer W2 money formatting", () => {
  it("formats minor units into rounded integer so'm without raw tiyin", () => {
    const whole = formatWholeSom(125_000);
    expect(whole).toContain("1");
    expect(whole).toContain("250");
    expect(whole.toLowerCase()).toContain("so'm");
    expect(whole).not.toContain("125000");

    const rounded = formatWholeSom(125_060);
    expect(rounded).toContain("251");
    expect(rounded).not.toContain("125060");
  });

  it("converts whole so'm cap inputs to minor units and back", () => {
    expect(somInputToMinor("1 500")).toBe(150_000);
    expect(somInputToMinor("0")).toBe(0);
    expect(somInputToMinor("-10")).toBeNull();
    expect(somInputToMinor("12.5")).toBeNull();
    expect(minorToSomInput(150_000)).toBe("1500");
    expect(minorToSomInput(0)).toBe("0");
  });
});

describe("retailer W2 claim errors", () => {
  it("maps reused_photo_evidence_refused across locales", () => {
    const body = {
      code: "reused_photo_evidence_refused",
      message: "Photo evidence hash already exists",
    };
    expect(parseClaimErrorMessage(body, 422, "en")).toBe(
      W2_COPY.en.claimReusedPhoto,
    );
    expect(parseClaimErrorMessage(body, 422, "ru")).toBe(
      W2_COPY.ru.claimReusedPhoto,
    );
    expect(parseClaimErrorMessage(body, 422, "uz")).toBe(
      W2_COPY.uz.claimReusedPhoto,
    );
  });

  it("maps photo_evidence_required and invalid_evidence_uri cleanly", () => {
    expect(
      parseClaimErrorMessage({ error: "photo_evidence_required" }, 400, "en"),
    ).toBe(W2_COPY.en.claimPhotoRequired);
    expect(
      parseClaimErrorMessage({ code: "invalid_evidence_uri" }, 400, "en"),
    ).toBe(W2_COPY.en.claimInvalidPhoto);
  });
});

describe("retailer W2 409 owner_approval_required parser", () => {
  it("extracts request_id, rule, minor caps, and cash/card fallbacks on 409", () => {
    const parsed = parseOwnerApprovalRequired(409, {
      code: "owner_approval_required",
      error: "owner_approval_required",
      request_id: "req-409-1",
      client_order_id: "ord-409-1",
      status: "PENDING_OWNER",
      tripped_rule: "DAILY_CREDIT_CAP_EXCEEDED",
      amount_minor: 250_000,
      threshold_minor: 200_000,
      remaining_cap_minor: 50_000,
      message: "Daily credit limit exceeded.",
      fallback_payment_methods: ["cash", "card"],
    });
    expect(parsed).not.toBeNull();
    expect(parsed?.request_id).toBe("req-409-1");
    expect(parsed?.amount_minor).toBe(250_000);
    expect(parsed?.threshold_minor).toBe(200_000);
    expect(parsed?.remaining_cap_minor).toBe(50_000);
    expect(parsed?.fallback_payment_methods).toEqual(["CASH", "CARD"]);
  });

  it("returns null on non-409 or unrelated 409 codes", () => {
    expect(
      parseOwnerApprovalRequired(409, { code: "ALL_ITEMS_OUT_OF_STOCK" }),
    ).toBeNull();
    expect(
      parseOwnerApprovalRequired(422, { code: "owner_approval_required" }),
    ).toBeNull();
  });
});
