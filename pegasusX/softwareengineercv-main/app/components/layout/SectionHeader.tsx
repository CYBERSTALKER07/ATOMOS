import { cn } from '@/lib/utils';

type SectionHeaderProps = {
  eyebrow?: string;
  title: string;
  description?: string;
  align?: 'left' | 'center';
  className?: string;
  titleId?: string;
};

export default function SectionHeader({
  eyebrow,
  title,
  description,
  align = 'left',
  className,
  titleId,
}: SectionHeaderProps) {
  return (
    <header
      className={cn(
        'section-header',
        align === 'center' && 'section-header--center mx-auto text-center',
        className
      )}
    >
      {eyebrow ? <p className="editorial-eyebrow mb-3">{eyebrow}</p> : null}
      <h2
        id={titleId}
        className="text-4xl font-light tracking-tight leading-[1.05] md:text-5xl lg:text-6xl"
      >
        {title}
      </h2>
      {description ? (
        <p
          className={cn(
            'mt-4 text-base text-white/60 md:text-lg leading-relaxed',
            align === 'center' ? 'mx-auto max-w-2xl' : 'max-w-xl'
          )}
        >
          {description}
        </p>
      ) : null}
    </header>
  );
}
