'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

export default function DemoLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  const getPersona = () => {
    if (pathname.includes('/supplier')) return 'Supplier';
    if (pathname.includes('/warehouse')) return 'Warehouse';
    if (pathname.includes('/retailer')) return 'Retailer';
    return 'Select Persona';
  };

  const persona = getPersona();
  const isHome = pathname === '/demo';

  const navLinks = [
    { name: 'Supplier View', href: '/demo/supplier', active: persona === 'Supplier' },
    { name: 'Warehouse View', href: '/demo/warehouse', active: persona === 'Warehouse' },
    { name: 'Retailer View', href: '/demo/retailer', active: persona === 'Retailer' },
  ];

  return (
    <div className="min-h-screen bg-[#050505] text-white flex flex-col md:flex-row font-sans">
      
      {/* Mobile Header / Desktop Sidebar */}
      {!isHome && (
        <aside className="w-full md:w-64 bg-[#0a0a0a] border-r border-white/5 flex flex-col flex-shrink-0">
          <div className="p-6 border-b border-white/5">
            <Link href="/" className="flex items-center gap-2">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <path d="M12 22C17.5228 22 22 17.5228 22 12C22 6.47715 17.5228 2 12 2C6.47715 2 2 6.47715 2 12C2 17.5228 6.47715 22 12 22Z" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                <path d="M12 8V16" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                <path d="M8 12H16" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
              <span className="font-bold tracking-widest uppercase">Pegasus</span>
            </Link>
            <div className="mt-4 text-xs font-mono text-white/40 uppercase tracking-widest">
              Demo Portal
            </div>
          </div>
          
          <nav className="flex-1 p-4 space-y-2">
            <div className="text-[10px] text-white/30 uppercase tracking-[0.2em] mb-4 px-2">Personas</div>
            {navLinks.map((link) => (
              <Link
                key={link.name}
                href={link.href}
                className={`block px-4 py-3 text-sm rounded-sm transition-colors ${
                  link.active 
                    ? 'bg-white/10 text-white font-medium border-l-2 border-white' 
                    : 'text-white/50 hover:bg-white/5 hover:text-white/90 border-l-2 border-transparent'
                }`}
              >
                {link.name}
              </Link>
            ))}
          </nav>
          
          <div className="p-4 border-t border-white/5">
            <Link href="/" className="text-xs text-white/40 hover:text-white transition-colors flex items-center gap-2">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M19 12H5M12 19l-7-7 7-7" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
              Back to Landing
            </Link>
          </div>
        </aside>
      )}

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col min-h-screen overflow-hidden">
        {/* Topbar */}
        {!isHome && (
          <header className="h-16 border-b border-white/5 bg-[#0a0a0a]/50 backdrop-blur-md flex items-center justify-between px-8 flex-shrink-0">
            <div className="text-sm font-medium text-white/90 flex items-center gap-3">
              <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
              {persona} Dashboard Live
            </div>
            
            <div className="flex items-center gap-4">
              <div className="w-8 h-8 rounded bg-white/5 border border-white/10 flex items-center justify-center text-white/60">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
              </div>
              <div className="h-8 w-8 rounded-full bg-gradient-to-tr from-gray-700 to-gray-500 border border-white/20" />
            </div>
          </header>
        )}
        
        {/* Page Content */}
        <div className="flex-1 overflow-y-auto p-4 md:p-8 relative">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,#444_1px,transparent_1px)] bg-[size:24px_24px] opacity-[0.15] pointer-events-none" />
          <div className="relative z-10 max-w-7xl mx-auto">
            {children}
          </div>
        </div>
      </main>
    </div>
  );
}
