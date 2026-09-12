'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const HIDDEN_PREFIXES = ['/admin', '/resume', '/platform', '/roles', '/technology', '/solutions', '/assistant'];

export default function SiteAssistant() {
  const pathname = usePathname();

  if (HIDDEN_PREFIXES.some((prefix) => pathname?.startsWith(prefix))) {
    return null;
  }

  return (
    <aside aria-label="Pegasus AI Assistant" className="fixed bottom-6 right-6 z-[10004]">
      <Link
        href="/assistant"
        className="glowing-squircle-launcher group focus:outline-none"
        title="Open Pegasus AI Assistant"
        aria-label="Open Pegasus AI Assistant"
      >
        <img
          src="/pegasus.jpg"
          alt="Pegasus AI Assistant"
          width={34}
          height={34}
          className="w-8 h-8 sm:w-9 sm:h-9 object-contain select-none transition-transform duration-200 group-hover:scale-110"
        />
        <span className="sr-only">Open Pegasus AI Assistant</span>
      </Link>
    </aside>
  );
}
