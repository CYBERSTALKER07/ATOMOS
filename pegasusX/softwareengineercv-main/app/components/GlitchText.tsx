'use client';

import { FC } from 'react';

interface GlitchTextProps {
  children: string;
  speed?: number;
  enableShadows?: boolean;
  enableOnHover?: boolean;
  className?: string;
}

const GlitchText: FC<GlitchTextProps> = ({
  children,
  speed = 0.5,
  enableShadows = true,
  enableOnHover = false,
  className = ''
}) => {
  const afterDuration = speed * 3;
  const beforeDuration = speed * 2;
  
  return (
    <>
      <style jsx>{`
        @keyframes glitch-anim {
          0% { clip-path: inset(20% 0 50% 0); }
          5% { clip-path: inset(10% 0 60% 0); }
          10% { clip-path: inset(15% 0 55% 0); }
          15% { clip-path: inset(25% 0 35% 0); }
          20% { clip-path: inset(30% 0 40% 0); }
          25% { clip-path: inset(40% 0 20% 0); }
          30% { clip-path: inset(10% 0 60% 0); }
          35% { clip-path: inset(15% 0 55% 0); }
          40% { clip-path: inset(25% 0 35% 0); }
          45% { clip-path: inset(30% 0 40% 0); }
          50% { clip-path: inset(20% 0 50% 0); }
          55% { clip-path: inset(10% 0 60% 0); }
          60% { clip-path: inset(15% 0 55% 0); }
          65% { clip-path: inset(25% 0 35% 0); }
          70% { clip-path: inset(30% 0 40% 0); }
          75% { clip-path: inset(40% 0 20% 0); }
          80% { clip-path: inset(20% 0 50% 0); }
          85% { clip-path: inset(10% 0 60% 0); }
          90% { clip-path: inset(15% 0 55% 0); }
          95% { clip-path: inset(25% 0 35% 0); }
          100% { clip-path: inset(30% 0 40% 0); }
        }

        .glitch-text {
          position: relative;
          display: inline-block;
          font-weight: 900;
          color: #FFFFFF;
          font-size: clamp(2rem, 10vw, 8rem);
          user-select: none;
          cursor: pointer;
          margin: 0 auto;
        }

        .glitch-text::before,
        .glitch-text::after {
          content: attr(data-text);
          position: absolute;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          background: #000000;
        }

        .glitch-text::before {
          left: -2px;
          text-shadow: ${enableShadows ? '2px 0 #C0C0C0' : 'none'};
          animation: glitch-anim ${beforeDuration}s infinite linear alternate-reverse;
          clip-path: inset(0 0 0 0);
        }

        .glitch-text::after {
          left: 2px;
          text-shadow: ${enableShadows ? '-2px 0 #FFFFFF' : 'none'};
          animation: glitch-anim ${afterDuration}s infinite linear alternate-reverse;
          clip-path: inset(0 0 0 0);
        }

        ${enableOnHover ? `
          .glitch-text::before,
          .glitch-text::after {
            opacity: 0;
          }

          .glitch-text:hover::before,
          .glitch-text:hover::after {
            opacity: 1;
          }
        ` : ''}
      `}</style>
      
      <div 
        className={`glitch-text ${className}`}
        data-text={children}
      >
        {children}
      </div>
    </>
  );
};

export default GlitchText;
