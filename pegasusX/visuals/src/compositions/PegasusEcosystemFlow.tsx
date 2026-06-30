import { Series } from 'remotion';
import { LineCanvas } from '../components/LineCanvas';
import { StrokeDraw } from '../components/StrokeDraw';
import { secondsToFrames } from '../style/tokens';
import { useVideoConfig } from 'remotion';

const CHAPTERS = [
  { title: 'ONE CONTROL PLANE', subtitle: 'Hub + six roles' },
  { title: 'HOW PEGASUS WORKS', subtitle: 'Order to payment' },
  { title: 'SIX ROLES, ONE ORDER', subtitle: 'Swimlanes + parity' },
  { title: 'MORNING DISPATCH', subtitle: 'Board + fleet' },
  { title: 'WHEN THINGS BREAK', subtitle: 'Exception playbooks' },
  { title: 'PAYMENT & TREASURY', subtitle: 'Pay at delivery' },
  { title: 'RELIABLE UPDATES', subtitle: 'Outbox + sync' },
  { title: 'UNDER THE HOOD', subtitle: 'Stack + AI + deploy' },
] as const;

const ChapterCard = ({
  title,
  subtitle,
}: {
  title: string;
  subtitle: string;
}) => {
  const { fps } = useVideoConfig();

  return (
    <LineCanvas>
      <StrokeDraw
        d="M 460 340 H 1460 V 740 H 460 Z"
        drawStartFrame={0}
        drawDurationFrames={secondsToFrames(2, fps)}
      />
      <StrokeDraw
        d="M 560 420 H 1360 M 560 500 H 1200 M 560 580 H 1000"
        drawStartFrame={secondsToFrames(1, fps)}
        drawDurationFrames={secondsToFrames(2, fps)}
        opacity={0.5}
      />
      <text
        x={960}
        y={480}
        textAnchor="middle"
        fill="#FFFFFF"
        fontFamily="ui-monospace, monospace"
        fontSize={28}
        letterSpacing="8"
      >
        {title}
      </text>
      <text
        x={960}
        y={540}
        textAnchor="middle"
        fill="rgba(255,255,255,0.6)"
        fontFamily="ui-monospace, monospace"
        fontSize={14}
        letterSpacing="4"
      >
        {subtitle}
      </text>
      <text
        x={960}
        y={820}
        textAnchor="middle"
        fill="rgba(255,255,255,0.6)"
        fontFamily="ui-monospace, monospace"
        fontSize={12}
        letterSpacing="3"
      >
        PEGASUS ECOSYSTEM FLOW — EXPAND THIS CHAPTER
      </text>
    </LineCanvas>
  );
};

/** 10-minute film scaffold — replace each chapter with full scene sequences */
export const PegasusEcosystemFlow = () => {
  const { fps } = useVideoConfig();
  const chapterSeconds = 75;

  return (
    <Series>
      {CHAPTERS.map((chapter) => (
        <Series.Sequence
          key={chapter.title}
          durationInFrames={secondsToFrames(chapterSeconds, fps)}
        >
          <ChapterCard title={chapter.title} subtitle={chapter.subtitle} />
        </Series.Sequence>
      ))}
    </Series>
  );
};
