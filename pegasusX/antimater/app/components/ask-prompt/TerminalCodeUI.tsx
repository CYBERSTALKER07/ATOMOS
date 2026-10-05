'use client';

import React, { useEffect, useRef } from 'react';
import { gsap } from 'gsap';

export default function TerminalCodeUI() {
  const codeLines = [
    { text: 'pegasus query --natural "Show me network latency across US regions"', isCmd: true },
    { text: 'Analyzing network topology...', delay: 0.5 },
    { text: 'Compiling query to GraphQL...', delay: 1.0 },
    { text: 'Fetching realtime metrics from edge nodes...', delay: 1.5 },
    { text: '> US-East: 12ms', delay: 2.0, color: 'text-emerald-400' },
    { text: '> US-West: 24ms', delay: 2.1, color: 'text-emerald-400' },
    { text: '> US-Central: 18ms', delay: 2.2, color: 'text-emerald-400' },
    { text: 'Query complete. Dashboard generated.', delay: 2.8, color: 'text-white' },
  ];

  const terminalRef = useRef<HTMLDivElement>(null);
  const cursorRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!terminalRef.current || !cursorRef.current) return;

    // Blinking cursor
    gsap.to(cursorRef.current, {
      opacity: 0,
      repeat: -1,
      yoyo: true,
      duration: 0.5,
      ease: 'steps(1)',
    });

    // Reveal lines one by one
    const lines = terminalRef.current.querySelectorAll('.terminal-line');
    
    const tl = gsap.timeline({ scrollTrigger: { trigger: terminalRef.current, start: 'top 80%' } });
    
    lines.forEach((line, i) => {
      const delay = codeLines[i].delay || 0;
      if (codeLines[i].isCmd) {
        tl.to(line, { display: 'block', duration: 0.01 }, delay)
          .fromTo(line, { opacity: 0, width: 0 }, { opacity: 1, width: '100%', duration: 1.5, ease: 'steps(40)' }, delay);
      } else {
        tl.fromTo(line, { opacity: 0, display: 'none' }, { opacity: 1, display: 'block', duration: 0.1 }, delay);
      }
    });

    return () => {
      tl.kill();
    };
  }, []);

  return (
    <div className="w-full max-w-2xl mx-auto rounded-xl overflow-hidden border border-zinc-800 shadow-2xl bg-[#0a0a0a] font-mono text-sm sm:text-base">
      {/* Terminal Header */}
      <div className="flex items-center px-4 py-3 bg-[#111] border-b border-zinc-800">
        <div className="flex gap-2">
          <div className="w-3 h-3 rounded-full bg-red-500/80" />
          <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
          <div className="w-3 h-3 rounded-full bg-green-500/80" />
        </div>
        <div className="mx-auto text-zinc-500 text-xs flex-1 text-center pr-8">
          pegasus-cli ~ bash
        </div>
      </div>
      
      {/* Terminal Body */}
      <div ref={terminalRef} className="p-5 sm:p-6 text-zinc-300 min-h-[300px]">
        {codeLines.map((line, i) => (
          <div
            key={i}
            className={`terminal-line hidden mb-2 ${line.color || 'text-zinc-400'}`}
            style={{ whiteSpace: 'nowrap', overflow: 'hidden' }}
          >
            {line.isCmd && <span className="text-emerald-500 mr-2">$</span>}
            {line.text}
          </div>
        ))}
        <div className="mt-2 text-emerald-500">
          $ <span ref={cursorRef} className="inline-block w-2.5 h-4 bg-emerald-500 align-middle -mt-1" />
        </div>
      </div>
    </div>
  );
}
