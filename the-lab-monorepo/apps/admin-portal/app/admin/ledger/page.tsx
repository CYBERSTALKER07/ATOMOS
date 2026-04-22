import { redirect } from 'next/navigation';

// Legacy duplicate — canonical ledger lives at /ledger
export default function LegacyLedgerRedirect() {
  redirect('/ledger');
}