import { Metadata } from 'next';
import SolutionsAccordion from './SolutionsAccordion';

export const metadata: Metadata = {
  title: 'Solutions | PegasusX',
  description: "PegasusX is helping global supply chains transform their logistics, warehouse management, and operational decision-making.",
};

export default function SolutionsPage() {
  return <SolutionsAccordion />;
}
