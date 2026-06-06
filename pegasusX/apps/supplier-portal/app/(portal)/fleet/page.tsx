import Link from "next/link";
import { PortalSurface } from "../_components/PortalSurface";

export default function FleetPage() {
  return (
    <PortalSurface
      title="Fleet & org"
      description="Drivers, vehicles, and org members for your supplier tenant."
    >
      <div className="md-card p-6 space-y-4">
        <p className="md-typescale-body-medium">
          Fleet onboarding runs on the dedicated org-fleet surface with topology validation and idempotent create support.
        </p>
        <Link href="/org-fleet" className="md-btn md-btn-filled inline-flex">
          Open org & fleet onboarding
        </Link>
      </div>
    </PortalSurface>
  );
}
