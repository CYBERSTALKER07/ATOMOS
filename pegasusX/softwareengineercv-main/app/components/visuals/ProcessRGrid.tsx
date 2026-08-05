'use client';

type ProcessStep = { title: string; description: string };

type ProcessRGridProps = {
  steps: ProcessStep[];
};

const GRID_AREAS = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'] as const;

type CellConfig = {
  area: typeof GRID_AREAS[number];
  arrow?: string;
  arrowPosition: 'bottom' | 'top-right' | 'inline';
  numberPosition: 'bottom' | 'top-right';
  shape: 'default' | 'bulge-right' | 'bulge-left' | 'curve-left' | 'arrow-end';
  rowSpan?: number;
};

const CELL_CONFIGS: CellConfig[] = [
  { area: 'a', arrow: '→', arrowPosition: 'bottom', numberPosition: 'bottom', shape: 'default' },
  { area: 'b', arrow: '→', arrowPosition: 'bottom', numberPosition: 'bottom', shape: 'default' },
  { area: 'c', arrow: '↓', arrowPosition: 'bottom', numberPosition: 'bottom', shape: 'bulge-right' },
  { area: 'd', arrow: '←', arrowPosition: 'top-right', numberPosition: 'bottom', shape: 'default' },
  { area: 'e', arrow: '←', arrowPosition: 'top-right', numberPosition: 'bottom', shape: 'default' },
  { area: 'f', arrow: '↓', arrowPosition: 'inline', numberPosition: 'top-right', shape: 'bulge-left', rowSpan: 2 },
  { area: 'g', arrow: '→', arrowPosition: 'top-right', numberPosition: 'bottom', shape: 'curve-left' },
  { area: 'h', arrowPosition: 'bottom', numberPosition: 'bottom', shape: 'arrow-end' },
];

function stepTone(index: number, total: number): string {
  const tones = ['tone-1', 'tone-2', 'tone-3', 'tone-4', 'tone-5', 'tone-6', 'tone-7', 'tone-8'];
  if (total <= 1) return 'tone-8';
  const mapped = Math.round((index / (total - 1)) * (tones.length - 1));
  return tones[mapped];
}

export default function ProcessRGrid({ steps }: ProcessRGridProps) {
  const slots = steps.slice(0, 8);

  return (
    <div className="process-r-grid" role="list">
      {slots.map((step, i) => {
        const config = CELL_CONFIGS[i];
        if (!config) return null;

        const isWide = config.shape === 'arrow-end';
        const isTall = config.rowSpan === 2;

        return (
          <article
            key={`${step.title}-${i}`}
            role="listitem"
            className={[
              'process-r-grid__cell',
              `process-r-grid__cell--${config.area}`,
              `process-r-grid__cell--${config.shape}`,
              stepTone(i, slots.length),
              isWide ? 'is-wide' : '',
              isTall ? 'is-tall' : '',
              config.arrowPosition === 'top-right' ? 'has-arrow-top' : '',
              config.numberPosition === 'top-right' ? 'has-num-top' : '',
              config.arrowPosition === 'inline' ? 'has-inline-flow' : '',
            ]
              .filter(Boolean)
              .join(' ')}
            style={{ gridArea: config.area }}
          >
            {config.arrowPosition === 'top-right' && config.arrow ? (
              <span className="process-r-grid__arrow process-r-grid__arrow--corner" aria-hidden>
                {config.arrow}
              </span>
            ) : null}

            {config.numberPosition === 'top-right' ? (
              <span className="process-r-grid__num process-r-grid__num--top">
                {String(i + 1).padStart(2, '0')}
              </span>
            ) : null}

            <div className="process-r-grid__body">
              <h3 className="process-r-grid__title">{step.title}</h3>
              {config.arrowPosition === 'inline' && config.arrow ? (
                <div className="process-r-grid__title-row">
                  <span className="process-r-grid__arrow process-r-grid__arrow--inline" aria-hidden>
                    {config.arrow}
                  </span>
                </div>
              ) : null}
              <p className="process-r-grid__desc">{step.description}</p>
            </div>

            {config.numberPosition === 'bottom' ? (
              <div className="process-r-grid__meta">
                <span className="process-r-grid__num">{String(i + 1).padStart(2, '0')}</span>
                {config.arrowPosition === 'bottom' && config.arrow ? (
                  <span className="process-r-grid__arrow" aria-hidden>{config.arrow}</span>
                ) : null}
              </div>
            ) : null}
          </article>
        );
      })}
    </div>
  );
}

export type { ProcessStep };
