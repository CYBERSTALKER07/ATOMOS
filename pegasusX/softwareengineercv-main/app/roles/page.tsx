import { ROLES_DATA } from '@/app/data/rolesData';
import Link from 'next/link';

export default function RolesPage() {
  return (
    <div className="min-h-screen bg-black text-white pt-32 pb-24">
      <div className="container mx-auto max-w-7xl px-4">
        <header className="mb-20">
          <p className="editorial-eyebrow mb-4">Solutions by Role</p>
          <h1 className="text-5xl md:text-7xl font-medium tracking-tight mb-6">
            O9-Class Planning & <br /> Execution Model
          </h1>
          <p className="text-xl text-white/60 max-w-2xl">
            Explore Pegasus features mapped to business roles. Discover how we handle complex edge cases and power live operations from forecasting to dispatch.
          </p>
        </header>

        <div className="space-y-32">
          {ROLES_DATA.map((role) => (
            <section key={role.id} className="scroll-mt-32" id={role.id}>
              <div className="mb-12 border-b border-white/10 pb-8">
                <h2 className="text-3xl md:text-5xl font-medium tracking-tight mb-4">
                  {role.name}
                </h2>
                <p className="text-lg text-white/60 max-w-3xl">
                  {role.description}
                </p>
              </div>

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                {role.subtopics.map((subtopic) => (
                  <div 
                    key={subtopic.id} 
                    className="flex flex-col group border border-white/15 bg-[#111] transition-colors hover:border-white/30"
                  >
                    {/* Image Half */}
                    <div className="aspect-[4/3] bg-[#222] relative overflow-hidden flex items-center justify-center border-b border-white/15">
                      <p className="text-white/30 font-mono text-sm uppercase tracking-widest text-center px-4">
                        [ Visualization Area ]<br/>
                        <span className="text-[10px] text-white/20 mt-2 block">
                          Waiting for user image for: {subtopic.title}
                        </span>
                      </p>
                    </div>

                    {/* Text Half */}
                    <div className="p-6 md:p-8 flex flex-col flex-1">
                      <h3 className="text-2xl font-medium mb-3">{subtopic.title}</h3>
                      <p className="text-sm text-white/50 mb-6 flex-1">
                        {subtopic.description}
                      </p>
                      
                      <div className="space-y-4 mb-8">
                        <div>
                          <p className="text-[10px] uppercase font-mono text-white/40 tracking-wider mb-1">Business Logic</p>
                          <p className="text-sm text-white/80 leading-relaxed">{subtopic.businessLogic}</p>
                        </div>
                        <div>
                          <p className="text-[10px] uppercase font-mono text-white/40 tracking-wider mb-1">Edge Cases</p>
                          <p className="text-sm text-white/80 leading-relaxed">{subtopic.edgeCases}</p>
                        </div>
                      </div>

                      <Link 
                        href={`/roles/${role.id}#${subtopic.id}`}
                        className="inline-flex items-center gap-2 text-sm font-medium tracking-wide uppercase group-hover:text-white transition-colors text-white/60 mt-auto"
                      >
                        Explore More
                        <span className="group-hover:translate-x-1 transition-transform">→</span>
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      </div>
    </div>
  );
}
