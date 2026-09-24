import React from 'react';

export default function Loading() {
  return (
    <div
      className="fixed inset-0 z-[9999] pointer-events-none flex items-center justify-center bg-black/10 dark:bg-black/20 backdrop-blur-[2px] transition-opacity duration-150"
      aria-hidden="true"
    >
      <div className="w-7 h-7 border-2 border-[#10B981]/25 border-t-[#10B981] rounded-full animate-spin" />
    </div>
  );
}
