import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();

export async function openSupplierOrderReceipt(
  orderId: string,
  format: "html" | "pdf" = "html",
): Promise<void> {
  // Prefer public HTML URL from metadata when available (no blob needed).
  if (format === "html") {
    try {
      const meta = await api.getOrderReceipt("supplier", orderId);
      if (meta.html_url) {
        window.open(meta.html_url, "_blank", "noopener,noreferrer");
        return;
      }
      if (meta.qr_url) {
        window.open(meta.qr_url, "_blank", "noopener,noreferrer");
        return;
      }
    } catch {
      /* fall through to authenticated blob */
    }
  }
  const blob = await api.fetchOrderReceiptBlob("supplier", orderId, format);
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
