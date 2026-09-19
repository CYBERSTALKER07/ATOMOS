'use client';

import { useLanguage } from '@/app/context/LanguageContext';

import type { FlowConfig } from '@/app/data/topicTypes';
import { FlowShell } from './FlowShell';

type Props = { config?: FlowConfig };

export default function ExceptionPlaybookFlow({ config }: Props) {
  const { language } = useLanguage();

  const steps = config?.roles ?? ['Trigger', 'Guard', 'Recovery', 'Notify'];
  const branchAt = config?.highlightStep ?? 2;

  return (
    <FlowShell title="Exception playbook">
      <div className="relative pl-6">
        <div className="absolute bottom-0 left-2 top-0 w-px bg-white/20" />
        {steps.map((step, i) => (
          <div key={step} className="relative mb-6 last:mb-0">
            <div
              className={`absolute -left-[17px] top-1 h-3 w-3 border ${
                i <= branchAt ? 'border-white bg-white' : 'border-white/40 bg-black'
              }`}
            />
            <p className={`font-mono text-xs uppercase ${i <= branchAt ? 'text-white' : 'text-white/40'}`}>
              {step}
            </p>
            {i === branchAt ? (
              <p className="mt-1 text-sm text-white/60">{language === 'ru' ? 'Путь восстановления активен — состояние согласовано' : 'Recovery path engaged — state stays consistent'}</p>
            ) : null}
          </div>
        ))}
      </div>
    </FlowShell>
  );
}
