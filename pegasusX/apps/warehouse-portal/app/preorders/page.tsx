import { redirect } from 'next/navigation';

export default function PreordersRedirectPage() {
  redirect('/orders?tab=preorders');
}
