import Link from "next/link";
import { PortalSurface } from "../_components/PortalSurface";

export default function TreasuryPage() {
  return (
    <PortalSurface title="Treasury" description="Payments, settlement authority, and earnings.">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Link href="/payments" className="md-card p-6 block hover:bg-[var(--color-md-surface-container-high)]">
          <h2 className="md-typescale-title-large">Payments & ledger</h2>
          <p className="mt-2 text-[var(--color-md-outline)]">Live finance stream, chargebacks, and reconciliation.</p>
        </Link>
        <Link href="/earnings" className="md-card p-6 block hover:bg-[var(--color-md-surface-container-high)]">
          <h2 className="md-typescale-title-large">Earnings & disputes</h2>
          <p className="mt-2 text-[var(--color-md-outline)]">Treasury splits and dispute operations.</p>
        </Link>
      </div>
    </PortalSurface>
  );
}
