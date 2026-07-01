import { notFound } from 'next/navigation';
import { SOLUTIONS_ACCORDION_DATA } from '@/app/data/solutionsAccordionData';
import Link from 'next/link';

export default async function SolutionDetailPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  
  let useCaseData = null;
  let parentSolution = null;

  for (const sol of SOLUTIONS_ACCORDION_DATA) {
    const found = sol.useCases.find(uc => uc.slug === slug);
    if (found) {
      useCaseData = found;
      parentSolution = sol;
      break;
    }
  }

  if (!useCaseData || !parentSolution) {
    notFound();
  }

  const imageUrl = useCaseData.image || '/Gemini_Generated_Image_un3te4un3te4un3t.png';

  return (
    <div className="min-h-screen bg-black text-white pt-32 pb-24 selection:bg-white/30">
      <div className="max-w-7xl mx-auto px-6 md:px-12">
        <div className="flex items-center gap-2 text-xs font-mono tracking-widest text-white/50 mb-12 uppercase">
          <Link href="/" className="hover:text-white transition-colors">⌂</Link>
          <span>/</span>
          <Link href="/solutions" className="hover:text-white transition-colors">Solutions</Link>
          <span>/</span>
          <span>{useCaseData.title}</span>
        </div>

        <div className="max-w-4xl mb-16">
          <p className="editorial-eyebrow mb-4 text-white/50">{parentSolution.title}</p>
          <h1 className="text-4xl md:text-6xl font-normal tracking-tight mb-8">
            {useCaseData.title}
          </h1>
          <p className="text-lg md:text-xl leading-relaxed text-white/70 max-w-3xl">
            {parentSolution.overview}
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 lg:gap-16">
          <div className="flex flex-col justify-center">
             <h3 className="text-2xl md:text-3xl font-light mb-6">Key Capabilities</h3>
             <ul className="space-y-4 text-white/70">
                <li className="flex items-start gap-3">
                  <span className="text-green-500 mt-1">✓</span>
                  <span>Real-time visibility into supply chain operations</span>
                </li>
                <li className="flex items-start gap-3">
                  <span className="text-green-500 mt-1">✓</span>
                  <span>Predictive analytics powered by AI/ML</span>
                </li>
                <li className="flex items-start gap-3">
                  <span className="text-green-500 mt-1">✓</span>
                  <span>Seamless integration with existing ERP systems</span>
                </li>
             </ul>
             <div className="mt-12">
               <Link 
                  href="/contact"
                  className="inline-flex items-center justify-center bg-white text-black px-6 py-3 text-sm font-bold tracking-wider uppercase hover:bg-gray-200 transition-colors"
                >
                  Request Demo
                </Link>
             </div>
          </div>

          <div className="border border-white/10 p-2 md:p-4 bg-[#111] hover:bg-[#1a1a1a] transition-colors rounded-none flex items-center justify-center">
            <img 
              src={imageUrl} 
              alt={useCaseData.title} 
              className="w-full h-auto object-cover border border-white/10"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
