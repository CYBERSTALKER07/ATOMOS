'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { FLEEK_NAV_LINKS, isFleekNavLinkActive } from '@/app/data/fleekNav';

type FleekNavProps = {
  activeHref?: string;
};

export default function FleekNav({ activeHref }: FleekNavProps) {
  const pathname = usePathname() ?? '';

  return (
    <header className="fleek-nav axion-nav">
      <Link href="/" className="fleek-nav__brand axion-nav__brand" aria-label="Pegasus home">
        /PEGASUS
      </Link>

      <nav className="axion-nav__center" aria-label="Primary">
        {FLEEK_NAV_LINKS.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className={`axion-nav__link ${
              isFleekNavLinkActive(link.href, pathname, activeHref) ? 'is-active' : ''
            }`}
          >
            {link.label}
          </Link>
        ))}
      </nav>

      <div className="fleek-nav__actions axion-nav__actions">
        <Link href="/contact" className="axion-btn axion-btn--primary axion-btn--sm">
          Contact Us
        </Link>
        <Link href="/demo" className="axion-btn axion-btn--outline axion-btn--sm">
          Sign in
        </Link>
      </div>
    </header>
  );
}
