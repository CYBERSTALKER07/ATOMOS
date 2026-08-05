import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { SOLUTIONS_ACCORDION_DATA } from '@/app/data/solutionsAccordionData';
import { SOLUTIONS_DEFAULT_IMAGE } from '@/app/lib/siteAssets';
import { breadcrumbJsonLd, jsonLdScript, pageMetadata } from '@/app/lib/seo';
import Link from 'next/link';
import SiteNav from '@/app/components/explore/SiteNav';

function findSolution(slug: string) {
  for (const sol of SOLUTIONS_ACCORDION_DATA) {
    const found = sol.useCases.find((uc) => uc.slug === slug);
    if (found) return { useCase: found, parent: sol };
  }
  return null;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const match = findSolution(slug);
  if (!match) return { title: 'Solution' };

  const { useCase, parent } = match;
  return pageMetadata({
    title: useCase.title,
    description: `${useCase.title} for ${parent.title} teams — ${parent.overview}`,
    path: `/solutions/${slug}`,
    image: useCase.image ?? SOLUTIONS_DEFAULT_IMAGE,
    imageAlt: `${useCase.title} — Pegasus ${parent.title} solution`,
  });
}

export default async function SolutionDetailPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const match = findSolution(slug);

  if (!match) {
    notFound();
  }

  const { useCase: useCaseData, parent: parentSolution } = match;
  const imageUrl = useCaseData.image || SOLUTIONS_DEFAULT_IMAGE;
  const breadcrumb = breadcrumbJsonLd([
    { name: 'Home', path: '/' },
    { name: 'Solutions', path: '/solutions' },
    { name: useCaseData.title, path: `/solutions/${slug}` },
  ]);

  return (
    <>
      <script type="application/ld+json" dangerouslySetInnerHTML={jsonLdScript(breadcrumb)} />
      <div className="min-h-screen bg-black text-white pb-24 selection:bg-white/30">
        <SiteNav activeHref="/solutions" />
        <div className="max-w-7xl mx-auto px-6 md:px-12 pt-32">
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-2 text-xs font-mono tracking-widest text-white/50 mb-12 uppercase"
          >
            <Link href="/" className="hover:text-white transition-colors">
              Home
            </Link>
            <span aria-hidden>/</span>
            <Link href="/solutions" className="hover:text-white transition-colors">
              Solutions
            </Link>
            <span aria-hidden>/</span>
            <span aria-current="page">{useCaseData.title}</span>
          </nav>

          <div className="max-w-4xl mb-16">
            <p className="editorial-eyebrow mb-4 text-white/50">{parentSolution.title}</p>
            <h1 className="text-4xl md:text-6xl font-normal tracking-tight mb-8">
              {useCaseData.title}
            </h1>
            <p className="text-lg md:text-xl leading-relaxed text-white/70 max-w-3xl">
              {parentSolution.overview}
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-12">
            <div className="border border-white/10 p-8 md:p-12 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none flex flex-col justify-center">
              <h2 className="text-2xl md:text-3xl font-light mb-8">Key Capabilities</h2>
              <ul className="space-y-6 text-white/70">
                <li className="flex items-start gap-4">
                  <span className="text-green-500 mt-1" aria-hidden>
                    ✓
                  </span>
                  <span className="text-lg">Real-time visibility into supply chain operations</span>
                </li>
                <li className="flex items-start gap-4">
                  <span className="text-green-500 mt-1" aria-hidden>
                    ✓
                  </span>
                  <span className="text-lg">Predictive analytics powered by AI/ML</span>
                </li>
                <li className="flex items-start gap-4">
                  <span className="text-green-500 mt-1" aria-hidden>
                    ✓
                  </span>
                  <span className="text-lg">Seamless integration with existing ERP systems</span>
                </li>
              </ul>
              <div className="mt-12">
                <Link
                  href="/contact"
                  className="inline-flex items-center justify-center border border-white/20 bg-transparent text-white px-8 py-4 text-sm font-bold tracking-wider uppercase hover:bg-white hover:text-black transition-colors w-full md:w-auto"
                >
                  Request Demo
                </Link>
              </div>
            </div>

            <div className="border border-white/10 p-2 md:p-4 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none flex items-center justify-center min-h-[400px]">
              <img
                src={imageUrl}
                alt={`${useCaseData.title} — Pegasus ${parentSolution.title} logistics illustration`}
                className="w-full h-full object-cover border border-white/10"
              />
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
