import Link from 'next/link';
import type { Route } from 'next';

type ChamferButtonProps = {
  href?: string;
  onClick?: () => void;
  children: React.ReactNode;
  variant?: 'fill' | 'ghost';
  size?: 'md' | 'sm' | 'icon';
  className?: string;
  type?: 'button' | 'submit';
  'aria-label'?: string;
};

export default function ChamferButton({
  href,
  onClick,
  children,
  variant = 'fill',
  size = 'md',
  className = '',
  type = 'button',
  'aria-label': ariaLabel,
}: ChamferButtonProps) {
  const classes = [
    'chamfer-btn',
    variant === 'ghost' ? 'chamfer-btn--ghost' : 'chamfer-btn--fill',
    size === 'sm' ? 'chamfer-btn--sm' : '',
    size === 'icon' ? 'chamfer-btn--icon' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ');

  if (href) {
    const isExternal = href.startsWith('http');
    if (isExternal) {
      return (
        <a href={href} className={classes} aria-label={ariaLabel} target="_blank" rel="noopener noreferrer">
          {children}
        </a>
      );
    }
    return (
      <Link href={href as Route} className={classes} aria-label={ariaLabel}>
        {children}
      </Link>
    );
  }

  return (
    <button type={type} onClick={onClick} className={classes} aria-label={ariaLabel}>
      {children}
    </button>
  );
}

export function ChamferArrowIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden>
      <path
        d="M2 7h8M7 3l4 4-4 4"
        stroke="currentColor"
        strokeWidth="1.25"
        strokeLinecap="square"
      />
    </svg>
  );
}
