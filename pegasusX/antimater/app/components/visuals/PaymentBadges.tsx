'use client';

export default function PaymentBadges({ dark = true }: { dark?: boolean }) {
  const bgClass = dark
    ? 'bg-[#141417] border-white/10 text-white shadow-none'
    : 'bg-white border-zinc-200 text-zinc-900 shadow-2xs';

  return (
    <div className="flex flex-wrap items-center gap-1.5 select-none" aria-label="Accepted secure payment & fiscal methods">
      {/* Visa */}
      <div className={`h-6 px-2 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <span className="text-[10px] font-black italic tracking-tighter text-[#3b82f6] font-sans">VISA</span>
      </div>

      {/* Mastercard */}
      <div className={`h-6 px-2 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <div className="flex items-center -space-x-1">
          <div className="w-3 h-3 rounded-full bg-[#EB001B] opacity-90" />
          <div className="w-3 h-3 rounded-full bg-[#F79E1B] opacity-90" />
        </div>
      </div>

      {/* Discover */}
      <div className={`h-6 px-1.5 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <div className="flex items-center">
          <span className="text-[8px] font-black tracking-tight text-white/90">DISC</span>
          <div className="w-2 h-2 rounded-full bg-[#FF6000] -mx-0.5" />
          <span className="text-[8px] font-black tracking-tight text-white/90">VER</span>
        </div>
      </div>

      {/* Amex */}
      <div className={`h-6 px-1.5 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <span className="text-[8px] font-black tracking-tighter text-[#38bdf8] bg-[#38bdf8]/10 px-1 rounded-2xs">AMEX</span>
      </div>

      {/* Diners Club */}
      <div className={`h-6 px-1.5 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <div className="w-3.5 h-3.5 rounded-full border border-[#38bdf8] flex items-center justify-center">
          <div className="w-1.5 h-2 border-l border-r border-[#38bdf8]" />
        </div>
      </div>

      {/* JCB */}
      <div className={`h-6 px-1.5 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <div className="flex items-center gap-0.5">
          <div className="w-1.5 h-3 bg-[#0E4C92] rounded-2xs" />
          <div className="w-1.5 h-3 bg-[#E60012] rounded-2xs" />
          <div className="w-1.5 h-3 bg-[#00873C] rounded-2xs" />
        </div>
      </div>

      {/* Soliq Fiscal */}
      <div className={`h-6 px-1.5 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <span className="text-[8px] font-mono font-bold tracking-tight text-emerald-400">SOLIQ</span>
      </div>

      {/* Apple Pay */}
      <div className={`h-6 px-2 border rounded-xs flex items-center justify-center gap-0.5 ${bgClass}`}>
        <svg className="w-2.5 h-2.5 fill-current text-white/90" viewBox="0 0 170 170">
          <path d="M150.37 130.25c-2.45 5.66-5.35 10.87-8.71 15.66-4.58 6.53-8.33 11.05-11.22 13.56-4.48 4.12-9.28 6.23-14.42 6.35-3.69 0-8.14-1.05-13.32-3.18-5.19-2.12-9.97-3.17-14.34-3.17-4.58 0-9.49 1.05-14.75 3.17-5.26 2.13-9.5 3.24-12.74 3.35-4.35.13-9.16-1.9-14.42-6.08-3.69-3.04-7.67-7.81-11.96-14.34-6.3-9.68-11.22-20.73-14.76-33.15-3.54-12.43-5.31-23.77-5.31-34.02 0-14.93 3.8-27.17 11.39-36.72 7.6-9.55 17.28-14.44 29.04-14.67 4.8 0 10.23 1.25 16.3 3.76 6.07 2.5 10.15 3.82 12.23 3.94 1.7 0 5.86-1.37 12.5-4.1 6.63-2.73 12.16-3.99 16.6-3.76 12.6.89 22.75 5.76 30.43 14.6-11 6.64-16.38 15.66-16.14 27.08.24 9.02 3.8 16.59 10.7 22.72 6.9 6.13 15.08 9.5 24.54 10.13-2.34 7.24-5.29 14.7-8.86 22.39zM119.22 31.95c0-7.39 2.65-14.4 7.95-21.03 5.3-6.63 11.92-10.63 19.86-12 1.01 7.84-1.4 15.02-7.23 21.54-5.83 6.52-12.69 10.36-20.58 11.49z" />
        </svg>
        <span className="text-[9px] font-semibold tracking-tight text-white/90 font-sans">Pay</span>
      </div>

      {/* Swift */}
      <div className={`h-6 px-1.5 border rounded-xs flex items-center justify-center ${bgClass}`}>
        <span className="text-[8px] font-mono font-bold tracking-tight text-white/70">SWIFT</span>
      </div>
    </div>
  );
}
