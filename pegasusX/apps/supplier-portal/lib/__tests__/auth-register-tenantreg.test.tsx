import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import RegisterPage from "@/app/auth/register/page";

const {
  mockSupplierFetch,
  mockPersistSession,
  mockLookupPlatformINN,
  mockSendPhoneOtp,
  mockVerifyPhoneOtp,
  mockResetPhoneOtpFlow,
} = vi.hoisted(() => ({
  mockSupplierFetch: vi.fn(),
  mockPersistSession: vi.fn(),
  mockLookupPlatformINN: vi.fn(),
  mockSendPhoneOtp: vi.fn(),
  mockVerifyPhoneOtp: vi.fn(),
  mockResetPhoneOtpFlow: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  supplierFetch: mockSupplierFetch,
  persistSession: mockPersistSession,
  getSupplierToken: () => "",
  readTokenFromCookie: () => "",
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    lookupPlatformINN: mockLookupPlatformINN,
  }),
}));

vi.mock("@/lib/firebase", () => ({
  sendPhoneOtp: mockSendPhoneOtp,
  verifyPhoneOtp: mockVerifyPhoneOtp,
  resetPhoneOtpFlow: mockResetPhoneOtpFlow,
}));

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => (key: string) => key,
}));

async function advanceToProfileStep() {
  fireEvent.change(screen.getByLabelText(/Phone/i), { target: { value: "909100001" } });
  fireEvent.click(screen.getByRole("button", { name: /^Continue$/i }));

  await screen.findByLabelText(/Verification code/i);
  fireEvent.change(screen.getByLabelText(/Verification code/i), {
    target: { value: "123456" },
  });
  fireEvent.click(screen.getByRole("button", { name: /^Continue$/i }));

  await screen.findByLabelText(/supplier_portal\.onboarding_inn\.tax_id_label/i);
}

describe("Supplier Portal GS-T1 RegisterPage (/v1/platform/tenants/register)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSendPhoneOtp.mockResolvedValue(undefined);
    mockVerifyPhoneOtp.mockResolvedValue("fb-id-tok-1");
    Object.defineProperty(window, "location", {
      writable: true,
      value: { href: "http://localhost:3000/auth/register" },
    });
  });

  it("verifies INN and mints new supplier tenant via POST /v1/platform/tenants/register", async () => {
    mockLookupPlatformINN.mockResolvedValue({
      status: "FOUND",
      inn: "301234567",
      normalized_inn: "301234567",
      legal_name: "Samarkand Beverage Logistics LLC",
    });
    mockSupplierFetch.mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({
        supplier_id: "sup-new-1",
        token: "jwt-sup-new-1",
        next_step: "/setup/business",
      }),
    });

    render(<RegisterPage />);
    await advanceToProfileStep();

    fireEvent.change(screen.getByLabelText(/supplier_portal\.onboarding_inn\.tax_id_label/i), {
      target: { value: "301234567" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: /supplier_portal\.onboarding_inn\.verify_button/i }),
    );

    await waitFor(() => {
      expect(mockLookupPlatformINN).toHaveBeenCalledWith({
        inn: "301234567",
        tenant_type: "SUPPLIER",
      });
    });

    expect((screen.getByLabelText(/Legal company name/i) as HTMLInputElement).value).toBe(
      "Samarkand Beverage Logistics LLC",
    );

    fireEvent.change(screen.getByLabelText(/Primary contact name/i), {
      target: { value: "Sardor Owner" },
    });
    fireEvent.change(screen.getByLabelText(/Work email/i), {
      target: { value: "sardor@sbl.uz" },
    });
    fireEvent.change(screen.getByLabelText(/Password \/ Access PIN/i), {
      target: { value: "654321" },
    });

    fireEvent.click(screen.getByRole("button", { name: /^Create supplier$/i }));

    await waitFor(() => {
      expect(mockSupplierFetch).toHaveBeenCalledWith(
        "/v1/platform/tenants/register",
        expect.objectContaining({
          method: "POST",
        }),
      );
    });

    const sentBody = JSON.parse(mockSupplierFetch.mock.calls[0][1].body as string);
    expect(sentBody).toEqual({
      legal_name: "Samarkand Beverage Logistics LLC",
      contact_name: "Sardor Owner",
      email: "sardor@sbl.uz",
      phone: "+998909100001",
      password: "654321",
      market_code: "UZ",
      tax_id: "301234567",
    });
    expect(mockPersistSession).toHaveBeenCalledWith("jwt-sup-new-1");
    expect(window.location.href).toBe("/setup/business");
  });

  it("renders join request confirmation cleanly on 409 join_request_pending", async () => {
    mockLookupPlatformINN.mockResolvedValue({
      status: "FOUND",
      inn: "301234567",
      normalized_inn: "301234567",
      legal_name: "Samarkand Beverage Logistics LLC",
      existing_join_eligible: true,
    });
    mockSupplierFetch.mockResolvedValue({
      ok: false,
      status: 409,
      json: async () => ({
        error: "join_request_pending",
        request_id: "jr_sup_dup_1",
        legal_name: "Samarkand Beverage Logistics LLC",
        normalized_inn: "301234567",
      }),
    });

    render(<RegisterPage />);
    await advanceToProfileStep();

    fireEvent.change(screen.getByLabelText(/supplier_portal\.onboarding_inn\.tax_id_label/i), {
      target: { value: "301234567" },
    });
    fireEvent.change(screen.getByLabelText(/Legal company name/i), {
      target: { value: "Samarkand Beverage Logistics LLC" },
    });
    fireEvent.change(screen.getByLabelText(/Primary contact name/i), {
      target: { value: "Jasur Ops" },
    });
    fireEvent.change(screen.getByLabelText(/Work email/i), {
      target: { value: "jasur@sbl.uz" },
    });
    fireEvent.change(screen.getByLabelText(/Password \/ Access PIN/i), {
      target: { value: "654321" },
    });

    fireEvent.click(screen.getByRole("button", { name: /^Create supplier$/i }));

    await screen.findByText(/#jr_sup_dup_1/);
    expect(mockPersistSession).not.toHaveBeenCalled();
  });
});
