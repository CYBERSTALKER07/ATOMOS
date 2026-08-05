export const FLEEK_NAV_LINKS = [
  { label: 'About us', href: '/platform' },
  { label: 'Services', href: '/capabilities' },
  { label: 'Our Approach', href: '/operations' },
  { label: 'Technology', href: '/technology' },
] as const;

/** Match active state for Fleek nav links (hub prefixes + app-family routes). */
export function isFleekNavLinkActive(href: string, pathname: string, activeHref?: string): boolean {
  const ref = activeHref ?? pathname;

  if (href === '/contact' && ref === '/contact') {
    return true;
  }

  if (href === '/platform' && (ref === '/projects' || ref.startsWith('/projects/'))) {
    return true;
  }

  if (href === '/capabilities' && (ref === '/roles' || ref.startsWith('/roles/'))) {
    return true;
  }

  if (href === '/apps-deploy') {
    return (
      ref === href ||
      ref.startsWith('/apps-deploy/') ||
      ref === '/web-apps' ||
      ref === '/mobile-apps' ||
      ref === '/desktop-apps'
    );
  }

  if (href === '/demo') {
    return ref === href || ref.startsWith('/demo/');
  }

  return ref === href || ref.startsWith(`${href}/`);
}
