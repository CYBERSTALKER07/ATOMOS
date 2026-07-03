import Link from 'next/link';

export default function DemoPortal() {
  const personas = [
    {
      name: 'Supplier',
      href: '/demo/supplier',
      desc: 'Manage outbound orders, track inventory, and view fulfillment SLAs.',
      icon: (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          <polyline points="3.27 6.96 12 12.01 20.73 6.96" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          <line x1="12" y1="22.08" x2="12" y2="12" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      )
    },
    {
      name: 'Warehouse',
      href: '/demo/warehouse',
      desc: 'Monitor live dock utilization, inbound/outbound queues, and cross-dock times.',
      icon: (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          <line x1="3" y1="9" x2="21" y2="9" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          <line x1="9" y1="21" x2="9" y2="9" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      )
    },
    {
      name: 'Retailer',
      href: '/demo/retailer',
      desc: 'Track in-transit deliveries, view real-time fleet ETA, and manage receiving.',
      icon: (
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          <polyline points="9 22 9 12 15 12 15 22" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      )
    }
  ];

  return (
    <div className="flex items-center justify-center min-h-[80vh]">
      <div className="w-full max-w-4xl px-4">
        <div className="text-center mb-16">
          <h1 className="text-4xl md:text-5xl font-medium tracking-tight mb-4">Pegasus Dashboard</h1>
          <p className="text-white/50 text-lg">Select a persona below to experience the specialized workflows.</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {personas.map((p) => (
            <Link 
              key={p.name} 
              href={p.href}
              className="group bg-[#0a0a0a] border border-white/10 p-8 rounded block transition-all hover:border-white/30 hover:bg-[#111]"
            >
              <div className="w-12 h-12 rounded bg-white/5 border border-white/10 flex items-center justify-center text-white/70 mb-6 group-hover:text-white group-hover:scale-110 transition-all">
                {p.icon}
              </div>
              <h2 className="text-xl font-medium text-white/90 mb-3">{p.name} Portal</h2>
              <p className="text-sm text-white/50 leading-relaxed">
                {p.desc}
              </p>
              <div className="mt-8 text-xs font-mono tracking-widest text-white/30 group-hover:text-white/80 transition-colors flex items-center gap-2">
                ENTER PLATFORM
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M5 12h14M12 5l7 7-7 7" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
              </div>
            </Link>
          ))}
        </div>
        
        <div className="mt-16 text-center">
          <Link href="/" className="text-sm text-white/40 hover:text-white transition-colors underline underline-offset-4">
            Return to Landing Page
          </Link>
        </div>
      </div>
    </div>
  );
}
