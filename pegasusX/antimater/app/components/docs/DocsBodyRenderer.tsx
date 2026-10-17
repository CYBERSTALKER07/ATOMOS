'use client';

import React, { useState } from 'react';
import {
 Check,
 Copy,
 AlertTriangle,
 Info,
 Lightbulb,
 CheckCircle2,
 Terminal,
 Activity,
 ArrowRight,
 ShieldAlert,
} from 'lucide-react';
import { DocArticle, DocCalloutType } from '@/app/data/docsData';

type DocsBodyRendererProps = {
 article: DocArticle;
};

function CodeBlock({
 code,
 lang,
 filename,
}: {
 code: string;
 lang: string;
 filename?: string;
}) {
 const [copied, setCopied] = useState(false);

 const handleCopy = async () => {
 try {
 await navigator.clipboard.writeText(code);
 setCopied(true);
 setTimeout(() => setCopied(false), 2000);
 } catch {
 // ignore
 }
 };

 return (
 <div className="my-4 rounded-none overflow-hidden border border-white/10 bg-black">
 {/* Code Header Bar */}
 <div className="flex items-center justify-between px-4 py-2 bg-black border-b border-white/10 text-xs font-mono text-zinc-400">
 <div className="flex items-center space-x-2">
 <Terminal className="w-3.5 h-3.5 text-white" />
 <span className="text-white font-medium">{filename || `${lang}-snippet`}</span>
 </div>
 <div className="flex items-center space-x-3">
 <span className="text-[10px] uppercase tracking-wider text-zinc-500">{lang}</span>
 <button
 onClick={handleCopy}
 className="flex items-center space-x-1 p-1 rounded-none hover:bg-black text-zinc-400 hover:text-white transition-colors"
 title="Copy snippet"
 >
 {copied ? (
 <>
 <Check className="w-3.5 h-3.5 text-white" />
 <span className="text-[10px] text-white">Copied</span>
 </>
 ) : (
 <>
 <Copy className="w-3.5 h-3.5" />
 <span className="text-[10px]">Copy</span>
 </>
 )}
 </button>
 </div>
 </div>

 {/* Code Content */}
 <pre className="p-4 overflow-x-auto text-xs font-mono text-zinc-300 leading-relaxed select-all">
 <code>{code}</code>
 </pre>
 </div>
 );
}

function CalloutBox({
 type,
 title,
 content,
}: {
 type: DocCalloutType;
 title: string;
 content: string;
}) {
 const styles = {
 important: {
 border: 'border-white/20',
 bg: 'bg-black',
 icon: <Info className="w-5 h-5 text-white shrink-0 mt-0.5" />,
 titleColor: 'text-white',
 },
 warning: {
 border: 'border-white/20',
 bg: 'bg-black',
 icon: <AlertTriangle className="w-5 h-5 text-zinc-300 shrink-0 mt-0.5" />,
 titleColor: 'text-white',
 },
 tip: {
 border: 'border-white/20',
 bg: 'bg-black',
 icon: <Lightbulb className="w-5 h-5 text-white shrink-0 mt-0.5" />,
 titleColor: 'text-white',
 },
 note: {
 border: 'border-white/20',
 bg: 'bg-black',
 icon: <ShieldAlert className="w-5 h-5 text-zinc-300 shrink-0 mt-0.5" />,
 titleColor: 'text-white',
 },
 }[type];

 return (
 <div className={`my-5 p-4 rounded-none border ${styles.border} ${styles.bg} flex items-start space-x-3.5`}>
 {styles.icon}
 <div className="space-y-1 text-xs sm:text-sm">
 <h5 className={`font-bold ${styles.titleColor}`}>{title}</h5>
 <p className="text-zinc-300 leading-relaxed">{content}</p>
 </div>
 </div>
 );
}

export default function DocsBodyRenderer({ article }: DocsBodyRendererProps) {
 const [checkedItems, setCheckedItems] = useState<Record<number, boolean>>({});

 const toggleCheck = (idx: number) => {
 setCheckedItems((prev) => ({ ...prev, [idx]: !prev[idx] }));
 };

 return (
 <div className="mt-8 space-y-10">
 {/* 1. Key Operational Highlights Cards */}
 {article.highlights && article.highlights.length > 0 && (
 <section className="space-y-3">
 <h3 className="text-xs font-mono uppercase tracking-widest text-zinc-400 flex items-center space-x-2">
 <Activity className="w-3.5 h-3.5 text-white" />
 <span>Architecture & Performance Highlights</span>
 </h3>
 <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
 {article.highlights.map((item, idx) => (
 <div
 key={idx}
 className="p-4 rounded-none bg-black border border-white/10 hover:border-white/30 transition-all flex flex-col justify-between"
 >
 <div>
 {item.metric && (
 <span className="inline-block text-[11px] font-mono font-bold px-2 py-0.5 rounded-none bg-white/10 text-white border border-white/20 mb-2">
 {item.metric}
 </span>
 )}
 <h4 className="text-sm font-semibold text-white">{item.title}</h4>
 <p className="text-xs text-zinc-400 mt-1.5 leading-relaxed">{item.desc}</p>
 </div>
 </div>
 ))}
 </div>
 </section>
 )}

 {/* 2. Architecture Execution Flow (if applicable) */}
 {article.architectureFlow && (
 <section className="p-6 rounded-none bg-black border border-white/10 space-y-4">
 <div className="flex items-center justify-between">
 <h3 className="text-sm font-bold text-white flex items-center space-x-2">
 <span className="w-2 h-2 rounded-none bg-white" />
 <span>{article.architectureFlow.title}</span>
 </h3>
 <span className="text-[10px] font-mono text-zinc-400 uppercase tracking-wider">
 Sequence Flow
 </span>
 </div>

 <div className="space-y-2.5">
 {article.architectureFlow.steps.map((step, idx) => (
 <div
 key={idx}
 className="flex items-start space-x-3 p-3 rounded-none bg-black border border-white/10 text-xs"
 >
 <div className="w-5 h-5 rounded-none bg-white/10 text-white font-mono text-[10px] flex items-center justify-center shrink-0 mt-0.5 border border-white/20">
 {idx + 1}
 </div>
 <p className="text-zinc-300 leading-relaxed">{step}</p>
 </div>
 ))}
 </div>

 {article.architectureFlow.caption && (
 <p className="text-xs text-zinc-400 italic font-mono pt-1">
 * {article.architectureFlow.caption}
 </p>
 )}
 </section>
 )}

 {/* 3. Step-by-Step Operator Procedures */}
 {article.steps && article.steps.length > 0 && (
 <section className="space-y-6">
 <h3 className="text-base font-bold text-white flex items-center space-x-2 border-b border-white/10 pb-3">
 <span>Standard Operating Procedure</span>
 </h3>

 <div className="space-y-6">
 {article.steps.map((step) => (
 <div key={step.stepNumber} className="relative pl-8 pb-4 border-l border-white/10">
 {/* Step Circle Indicator */}
 <div className="absolute -left-3.5 top-0 w-7 h-7 rounded-none bg-black border border-white text-white font-mono text-xs font-bold flex items-center justify-center ">
 {step.stepNumber}
 </div>

 <div className="space-y-2">
 <h4 className="text-sm font-semibold text-white">{step.title}</h4>
 <p className="text-xs sm:text-sm text-zinc-400 leading-relaxed">{step.desc}</p>

 {/* Checklist if provided */}
 {step.checklist && (
 <ul className="mt-3 space-y-1.5 pl-1">
 {step.checklist.map((c, i) => (
 <li key={i} className="flex items-center space-x-2 text-xs text-zinc-300">
 <CheckCircle2 className="w-3.5 h-3.5 text-white shrink-0" />
 <span>{c}</span>
 </li>
 ))}
 </ul>
 )}

 {/* Inline Code if provided */}
 {step.code && (
 <CodeBlock
 code={step.code.code}
 lang={step.code.lang}
 filename={step.code.filename}
 />
 )}
 </div>
 </div>
 ))}
 </div>
 </section>
 )}

 {/* 4. Standalone Code Examples */}
 {article.codeExamples && article.codeExamples.length > 0 && (
 <section className="space-y-4">
 <h3 className="text-base font-bold text-white border-b border-white/10 pb-3">
 Code & Configuration References
 </h3>
 <div className="space-y-4">
 {article.codeExamples.map((ex, i) => (
 <div key={i} className="space-y-1">
 {ex.title && <h5 className="text-xs font-semibold text-zinc-300">{ex.title}</h5>}
 {ex.description && <p className="text-xs text-zinc-400">{ex.description}</p>}
 <CodeBlock code={ex.code} lang={ex.lang} filename={ex.filename} />
 </div>
 ))}
 </div>
 </section>
 )}

 {/* 5. Alerts & Callouts */}
 {article.callouts && article.callouts.length > 0 && (
 <div className="space-y-3">
 {article.callouts.map((c, i) => (
 <CalloutBox key={i} type={c.type} title={c.title} content={c.content} />
 ))}
 </div>
 )}

 {/* 6. Operator Readiness Checklist */}
 {article.operatorChecklist && article.operatorChecklist.length > 0 && (
 <section className="p-5 rounded-none bg-black border border-white/10 space-y-3">
 <div className="flex items-center justify-between border-b border-white/10 pb-2.5">
 <h4 className="text-xs font-mono uppercase tracking-widest text-zinc-400 flex items-center space-x-2">
 <CheckCircle2 className="w-4 h-4 text-white" />
 <span>Operator Execution Checklist</span>
 </h4>
 <span className="text-[11px] font-mono text-zinc-400">
 {Object.values(checkedItems).filter(Boolean).length}/{article.operatorChecklist.length} Verified
 </span>
 </div>

 <div className="space-y-2">
 {article.operatorChecklist.map((item, idx) => {
 const isChecked = !!checkedItems[idx];
 return (
 <div
 key={idx}
 onClick={() => toggleCheck(idx)}
 className={`flex items-start space-x-3 p-2.5 rounded-none cursor-pointer transition-colors border ${
 isChecked
 ? 'bg-black border-white/30'
 : 'bg-black border-white/10 hover:border-white/20'
 }`}
 >
 <div
 className={`w-4 h-4 rounded-none mt-0.5 flex items-center justify-center border transition-colors ${
 isChecked
 ? 'bg-white border-white text-black'
 : 'border-zinc-700 bg-black'
 }`}
 >
 {isChecked && <Check className="w-3 h-3 stroke-[3]" />}
 </div>
 <div className="text-xs">
 <span className={`font-semibold ${isChecked ? 'text-zinc-400 line-through' : 'text-white'}`}>
 {item.label}
 </span>
 <p className="text-zinc-400 mt-0.5">{item.detail}</p>
 </div>
 </div>
 );
 })}
 </div>
 </section>
 )}
 </div>
 );
}
