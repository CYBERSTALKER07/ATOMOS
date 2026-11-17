import { createWarehouseApi } from "@/lib/api";

const api = createWarehouseApi();

function safeExternalUrl(raw?: string | null): string | null {
  if (!raw) return null;
  try {
    const base = typeof window !== "undefined" ? window.location.origin : "https://pegasusx.app";
    const u = new URL(raw.trim(), base);
    return u.protocol === "https:" || u.protocol === "http:" ? u.toString() : null;
  } catch {
    return null;
  }
}

export async function openWarehouseOrderReceipt(
  orderId: string,
  format: "html" | "pdf" = "html",
): Promise<void> {
  if (format === "html") {
    try {
      const meta = await api.getOrderReceipt("warehouse", orderId);
      const htmlUrl = safeExternalUrl(meta.html_url);
      if (htmlUrl) {
        window.open(htmlUrl, "_blank", "noopener,noreferrer");
        return;
      }
      const qrUrl = safeExternalUrl(meta.qr_url);
      if (qrUrl) {
        window.open(qrUrl, "_blank", "noopener,noreferrer");
        return;
      }
    } catch {
      /* fall through */
    }
  }
  const blob = await api.fetchOrderReceiptBlob("warehouse", orderId, format);
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
