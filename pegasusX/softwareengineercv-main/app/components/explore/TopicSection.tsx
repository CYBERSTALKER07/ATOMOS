import { ContentCardEyebrow } from '@/app/components/ContentCard';

type TopicSectionProps = {
  eyebrow: string;
  title: string;
  children: React.ReactNode;
  className?: string;
};

export default function TopicSection({
  eyebrow,
  title,
  children,
  className = '',
}: TopicSectionProps) {
  return (
    <section className={`py-12 md:py-16 border-t border-white/10 ${className}`}>
      <ContentCardEyebrow>{eyebrow}</ContentCardEyebrow>
      <h2 className="mt-3 text-2xl md:text-3xl font-semibold tracking-tight">{title}</h2>
      <div className="mt-6">{children}</div>
    </section>
  );
}
