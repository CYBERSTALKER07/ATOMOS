import './index.css';
import { Composition } from 'remotion';
import {
  compositionDurationInFrames,
  PEGASUS_COMPOSITIONS,
} from './lib/compositions';
import { OrderLifecycle } from './compositions/OrderLifecycle';
import { PegasusEcosystemFlow } from './compositions/PegasusEcosystemFlow';
import { PEGASUS_VIDEO } from './style/tokens';

const { width, height, fps } = PEGASUS_VIDEO;

export const RemotionRoot: React.FC = () => {
  const orderLifecycle = PEGASUS_COMPOSITIONS.find((c) => c.id === 'OrderLifecycle')!;
  const ecosystem = PEGASUS_COMPOSITIONS.find((c) => c.id === 'PegasusEcosystemFlow')!;

  return (
    <>
      <Composition
        id={orderLifecycle.id}
        component={OrderLifecycle}
        durationInFrames={compositionDurationInFrames(orderLifecycle)}
        fps={fps}
        width={width}
        height={height}
      />
      <Composition
        id={ecosystem.id}
        component={PegasusEcosystemFlow}
        durationInFrames={compositionDurationInFrames(ecosystem)}
        fps={fps}
        width={width}
        height={height}
      />
    </>
  );
};
