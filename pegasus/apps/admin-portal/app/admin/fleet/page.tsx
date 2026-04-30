import { redirect } from 'next/navigation';

// Legacy duplicate — canonical fleet telemetry lives at /fleet
export default function LegacyFleetRedirect() {
  redirect('/fleet');
}