import Link from 'next/link';
import type { Route } from 'next';
import { cn } from '@/lib/utils';

type ChamferButtonProps = {
  href?: string;
  onClick?: () => void;
  children: React.ReactNode;
  variant?: 'fill' | 'ghost';
  size?: 'md' | 'sm' | 'icon';
  className?: string;
  type?: 'button' | 'submit';
  disabled?: boolean;
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
  disabled = false,
  'aria-label': ariaLabel,
}: ChamferButtonProps) {
  const classes = cn(
    'chamfer-btn',
    variant === 'ghost' ? 'chamfer-btn--ghost' : 'chamfer-btn--fill',
    size === 'sm' && 'chamfer-btn--sm',
    size === 'icon' && 'chamfer-btn--icon',
    disabled && 'pointer-events-none opacity-50',
    className
  );

  if (href) {
    const isExternal = href.startsWith('http');
    const isHashOrSpecial =
      href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:');
    if (isExternal) {
      return (
        <a href={href} className={classes} aria-label={ariaLabel} target="_blank" rel="noopener noreferrer">
          {children}
        </a>
      );
    }
    if (isHashOrSpecial) {
      return (
        <a href={href} className={classes} aria-label={ariaLabel}>
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
    <button type={type} onClick={onClick} className={classes} aria-label={ariaLabel} disabled={disabled}>
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
