'use client';

import { useState } from 'react';
import PageSectionBlock from './PageSectionBlock';
import { cn } from '@/lib/utils';
import { useLanguage } from '@/app/context/LanguageContext';

type RoleRow = { role: string; touchpoint: string };

type RoleMatrixProps = {
  crossRole: RoleRow[];
  variant?: 'tabs' | 'table';
};

export default function RoleMatrix({ crossRole, variant = 'tabs' }: RoleMatrixProps) {
  const [active, setActive] = useState(0);
  const { t } = useLanguage();

  if (variant === 'table') {
    return (
      <PageSectionBlock eyebrow={t('sec_roles_eyebrow')} title={t('sec_roles_title')}>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[400px] text-left text-sm">
            <thead>
              <tr className="border-b border-white/20 font-mono text-xs uppercase text-white/50">
                <th className="py-3 pr-4">{t('sec_role_col')}</th>
                <th className="py-3">{t('sec_touchpoint_col')}</th>
              </tr>
            </thead>
            <tbody>
              {crossRole.map((row) => (
                <tr key={row.role} className="border-b border-white/10">
                  <td className="py-3 pr-4 font-medium">{row.role}</td>
                  <td className="py-3 text-white/60">{row.touchpoint}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </PageSectionBlock>
    );
  }

  const current = crossRole[active];

  return (
    <PageSectionBlock eyebrow={t('sec_roles_eyebrow')} title={t('sec_roles_title')}>
      <div className="flex flex-wrap gap-2">
        {crossRole.map((row, i) => (
          <button
            key={row.role}
            type="button"
            onClick={() => setActive(i)}
            className={cn(
              'border px-4 py-2 text-xs font-mono uppercase tracking-wider transition-colors',
              i === active
                ? 'border-white bg-white text-black'
                : 'border-white/20 text-white/60 hover:border-white/40 hover:text-white'
            )}
          >
            {row.role}
          </button>
        ))}
      </div>
      {current ? (
        <div className="mt-6 border border-white/15 bg-[#0a0a0a] p-6 md:p-8">
          <p className="font-mono text-[10px] uppercase tracking-widest text-white/40">
            {current.role}
          </p>
          <p className="mt-3 text-lg text-white/85">{current.touchpoint}</p>
        </div>
      ) : null}
    </PageSectionBlock>
  );
}
