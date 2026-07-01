import { Metadata } from 'next';
import SolutionsAccordion from './SolutionsAccordion';

export const metadata: Metadata = {
  title: 'Solutions | Pegasus',
  description: "Pegasus's Digital Brain platform is helping global companies across industries transform supply chain, revenue, and logistics planning.",
};

export default function SolutionsPage() {
  return <SolutionsAccordion />;
}
