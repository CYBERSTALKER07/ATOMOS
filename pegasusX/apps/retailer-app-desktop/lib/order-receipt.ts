import { apiFetch } from "@/lib/auth";

/** Open retailer party-copy receipt (HTML) or download PDF via authenticated fetch. */
export async function openRetailerOrderReceipt(
  orderId: string,
  format: "html" | "pdf" = "html",
): Promise<void> {
  const res = await apiFetch(
    `/v1/retailer/orders/${encodeURIComponent(orderId)}/receipt?format=${format}`,
  );
  if (!res.ok) {
    throw new Error(`receipt_unavailable_${res.status}`);
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  if (format === "pdf") {
    const a = document.createElement("a");
    a.href = url;
    a.download = `pegasus-receipt-${orderId.slice(-8)}.pdf`;
    a.click();
  } else {
    window.open(url, "_blank", "noopener,noreferrer");
  }
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

/** Prefer public fiscal QR HTML when present; otherwise authenticated retailer receipt. */
export async function openTrackingReceipt(order: {
  order_id: string;
  fiscal_qr?: string;
  latest_fiscal_receipt_id?: string;
}): Promise<void> {
  const qr = (order.fiscal_qr || "").trim();
  if (qr) {
    window.open(qr, "_blank", "noopener,noreferrer");
    return;
  }
  await openRetailerOrderReceipt(order.order_id, "html");
}
