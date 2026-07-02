import { Metadata } from 'next';
import SolutionsAccordion from './SolutionsAccordion';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Solutions',
  description:
    'Pegasus solutions for supplier control, warehouse dispatch, retailer tracking, treasury reconciliation, driver execution, factory loading, and gate returns — one platform across six roles.',
  path: '/solutions',
});

export default function SolutionsPage() {
  return <SolutionsAccordion />;
}
