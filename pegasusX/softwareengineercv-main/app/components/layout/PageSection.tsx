import { forwardRef, type ComponentPropsWithoutRef, type ReactNode } from 'react';
import { cn } from '@/lib/utils';

type PageSectionProps = {
  innerClassName?: string;
  bleed?: boolean;
  children: ReactNode;
} & ComponentPropsWithoutRef<'section'>;

const PageSection = forwardRef<HTMLElement, PageSectionProps>(function PageSection(
  { className, innerClassName, bleed = false, children, ...rest },
  ref
) {
  return (
    <section
      ref={ref}
      className={cn('page-section bg-black text-white', className)}
      {...rest}
    >
      <div className={cn(!bleed && 'page-shell', innerClassName)}>{children}</div>
    </section>
  );
});

export default PageSection;
