'use client';

import { useEffect, useRef } from 'react';
import PageSection from './layout/PageSection';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

const NODES = [
  { id: 'n1', x: 80, y: 150, type: 'start', icon: 'email', title: 'Email Trigger', subtitle: '(IMAP)' },
  { id: 'n2', x: 280, y: 150, type: 'rect', icon: 'if', title: 'If', subtitle: '' },
  { id: 'n3', x: 480, y: 150, type: 'rect', icon: 'edit', title: 'Edit Fields', subtitle: 'Manual' },
  { id: 'n4', x: 680, y: 150, type: 'rect', icon: 'agent', title: 'AI Agent', subtitle: 'Tools Agent' },
  { id: 'n5', x: 480, y: 260, type: 'rect', icon: 'code', title: 'Code', subtitle: '' },
  { id: 'n6', x: 680, y: 260, type: 'rect', icon: 'edit', title: 'Edit Fields1', subtitle: 'Manual' },
  { id: 'n7', x: 880, y: 150, type: 'rect', icon: 'send_email', title: 'Send Email', subtitle: 'Send' },
  { id: 'n8', x: 880, y: 260, type: 'rect', icon: 'telegram', title: 'Telegram', subtitle: 'sendAndWait message' },
];

const CONNECTIONS = [
  { from: 'n1', to: 'n2' },
  { from: 'n2', to: 'n3' },
  { from: 'n2', to: 'n5', branch: true },
  { from: 'n3', to: 'n4' },
  { from: 'n5', to: 'n6' },
  { from: 'n4', to: 'n7' },
  { from: 'n6', to: 'n8' },
];

function WorkflowIcon({ type, className = "" }: { type: string, className?: string }) {
  const commonProps = {
    className,
    fill: "currentColor",
    style: { imageRendering: 'pixelated' as const }
  };
  
  switch (type) {
    case 'email':
      return <svg {...commonProps} viewBox="0 0 17.333 16"><path d="M 17.037 5.445 L 9.037 0.112 C 8.813 -0.037 8.521 -0.037 8.297 0.112 L 0.297 5.445 C 0.111 5.569 0 5.777 0 6 L 0 14.667 C 0 15.403 0.597 16 1.333 16 L 16 16 C 16.736 16 17.333 15.403 17.333 14.667 L 17.333 6 C 17.333 5.777 17.222 5.569 17.037 5.445 Z M 6.06 10.667 L 1.333 14 L 1.333 7.295 Z M 7.424 11.334 L 9.909 11.334 L 14.628 14.667 L 2.705 14.667 Z M 11.273 10.667 L 16 7.295 L 16 14 Z" /></svg>;
    case 'if':
      return <svg {...commonProps} viewBox="0 0 20.466 19"><path d="M 20.279 8.527 L 17.203 11.939 C 16.926 12.247 16.531 12.423 16.117 12.423 L 10.231 12.423 L 10.231 18.269 C 10.231 18.673 9.904 19 9.5 19 C 9.096 19 8.769 18.673 8.769 18.269 L 8.769 12.423 L 1.462 12.423 C 0.654 12.423 0 11.769 0 10.962 L 0 5.115 C 0 4.308 0.654 3.654 1.462 3.654 L 8.769 3.654 L 8.769 0.731 C 8.769 0.327 9.096 0 9.5 0 C 9.904 0 10.231 0.327 10.231 0.731 L 10.231 3.654 L 16.117 3.654 C 16.531 3.654 16.926 3.83 17.203 4.138 L 20.279 7.55 C 20.529 7.828 20.529 8.249 20.279 8.527 Z" /></svg>;
    case 'edit':
      return <svg {...commonProps} viewBox="0 0 16.001 16"><path d="M 15.626 3.949 L 12.05 0.375 C 11.81 0.135 11.485 0 11.145 0 C 10.806 0 10.48 0.135 10.24 0.375 L 0.375 10.24 C 0.134 10.479 -0.001 10.805 0 11.145 L 0 14.72 C 0 15.427 0.573 16 1.28 16 L 14.72 16 C 15.073 16 15.36 15.713 15.36 15.36 C 15.36 15.006 15.073 14.72 14.72 14.72 L 6.666 14.72 L 15.626 5.76 C 15.866 5.52 16.001 5.194 16.001 4.855 C 16.001 4.515 15.866 4.189 15.626 3.949 Z M 12.8 6.775 L 9.226 3.2 L 11.146 1.28 L 14.72 4.855 Z" /></svg>;
    case 'agent':
      return <svg {...commonProps} viewBox="0 0 19 24"><path d="M 15.74 1.317 C 16.043 0.238 14.996 -0.4 14.04 0.281 L 0.693 9.79 C -0.344 10.528 -0.181 12 0.938 12 L 4.453 12 L 4.453 11.973 L 11.303 11.973 L 5.721 13.943 L 3.261 22.683 C 2.957 23.762 4.004 24.4 4.96 23.719 L 18.307 14.21 C 19.344 13.472 19.181 12 18.062 12 L 12.732 12 Z" /></svg>;
    case 'code':
      return <svg {...commonProps} viewBox="0 0 18.909 16"><path d="M 17.455 0 L 1.455 0 C 0.651 0 0 0.651 0 1.455 L 0 14.545 C 0 15.349 0.651 16 1.455 16 L 17.455 16 C 18.258 16 18.909 15.349 18.909 14.545 L 18.909 1.455 C 18.909 0.651 18.258 0 17.455 0 Z M 6.255 9.6 C 6.576 9.841 6.641 10.297 6.4 10.618 C 6.159 10.94 5.703 11.005 5.382 10.764 L 2.473 8.582 C 2.29 8.444 2.182 8.229 2.182 8 C 2.182 7.771 2.29 7.556 2.473 7.418 L 5.382 5.236 C 5.703 4.995 6.159 5.06 6.4 5.382 C 6.641 5.703 6.576 6.159 6.255 6.4 L 4.121 8 Z M 11.608 3.109 L 8.699 13.291 C 8.632 13.545 8.432 13.743 8.177 13.809 C 7.922 13.874 7.652 13.797 7.47 13.607 C 7.288 13.416 7.224 13.142 7.301 12.891 L 10.21 2.709 C 10.326 2.33 10.724 2.114 11.105 2.223 C 11.486 2.332 11.71 2.726 11.608 3.109 Z M 16.436 8.582 L 13.527 10.764 C 13.206 11.005 12.75 10.94 12.509 10.618 C 12.268 10.297 12.333 9.841 12.655 9.6 L 14.788 8 L 12.655 6.4 C 12.333 6.159 12.268 5.703 12.509 5.382 C 12.75 5.06 13.206 4.995 13.527 5.236 L 16.436 7.418 C 16.619 7.556 16.727 7.771 16.727 8 C 16.727 8.229 16.619 8.444 16.436 8.582 Z" /></svg>;
    case 'send_email':
      return <svg {...commonProps} viewBox="0 0 14.282 15.997"><path d="M 14.282 7.989 C 14.283 8.404 14.06 8.786 13.698 8.989 L 1.704 15.847 C 1.532 15.944 1.337 15.996 1.139 15.997 C 0.77 15.995 0.424 15.814 0.211 15.511 C -0.002 15.209 -0.055 14.822 0.068 14.473 L 1.997 8.763 C 2.035 8.648 2.142 8.57 2.262 8.568 L 7.425 8.568 C 7.584 8.569 7.735 8.503 7.843 8.388 C 7.952 8.272 8.007 8.117 7.997 7.959 C 7.97 7.654 7.713 7.422 7.407 7.425 L 2.264 7.425 C 2.141 7.425 2.032 7.347 1.993 7.231 L 0.064 1.521 C -0.093 1.073 0.044 0.574 0.407 0.269 C 0.77 -0.037 1.285 -0.087 1.699 0.145 L 13.699 6.994 C 14.059 7.196 14.282 7.576 14.282 7.989 Z" /></svg>;
    case 'telegram':
      return <svg {...commonProps} viewBox="0 0 17.92 16"><path d="M 17.671 0.175 C 17.469 0.001 17.187 -0.048 16.938 0.049 L 0.725 6.394 C 0.253 6.578 -0.041 7.051 0.005 7.556 C 0.05 8.06 0.423 8.474 0.92 8.571 L 5.121 9.396 L 5.121 14.08 C 5.119 14.602 5.436 15.072 5.921 15.266 C 6.404 15.464 6.96 15.346 7.321 14.968 L 9.347 12.867 L 12.561 15.68 C 12.792 15.886 13.091 15.999 13.401 16 C 13.537 16 13.672 15.978 13.801 15.937 C 14.231 15.8 14.556 15.446 14.655 15.006 L 17.902 0.88 C 17.961 0.62 17.872 0.349 17.671 0.175 Z M 13.403 14.72 L 6.789 8.92 L 16.309 2.097 Z" /></svg>;
    default:
      return <svg {...commonProps} viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"></circle></svg>;
  }
}

export default function LogisticsWorkflow() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let ctx = gsap.context(() => {
      const tl = gsap.timeline({
        scrollTrigger: {
          trigger: containerRef.current,
          start: "top 70%",
        }
      });

      tl.fromTo(".wf-element", 
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.6, stagger: 0.05, ease: "power3.out" }
      );

      // Flow animation
      gsap.to(".flow-dash", {
        strokeDashoffset: -20,
        duration: 0.8,
        ease: "none",
        repeat: -1,
      });

    }, containerRef);
    
    return () => ctx.revert();
  }, []);

  const getNodePorts = (id: string) => {
    const node = NODES.find(n => n.id === id);
    if (!node) return { outX: 0, outY: 0, inX: 0, inY: 0 };
    
    const w = 150;
    const h = 48;
    return {
      outX: node.x + w, // Right edge
      outY: node.y + h / 2,
      inX: node.x, // Left edge
      inY: node.y + h / 2,
      centerX: node.x + w / 2,
      bottomY: node.y + h
    };
  };

  const renderConnection = (conn: any, index: number) => {
    const from = getNodePorts(conn.from);
    const to = getNodePorts(conn.to);
    
    let path = "";
    if (conn.branch) {
      // Draw path from bottom of 'from' node to left of 'to' node
      const midY = (from.bottomY + to.inY) / 2;
      path = `M ${from.centerX} ${from.bottomY} L ${from.centerX} ${to.inY} L ${to.inX} ${to.inY}`;
    } else {
      // Orthogonal path from out to in
      const midX = (from.outX + to.inX) / 2;
      path = `M ${from.outX} ${from.outY} L ${midX} ${from.outY} L ${midX} ${to.inY} L ${to.inX} ${to.inY}`;
    }

    return (
      <g key={index} className="wf-element">
        {/* Base line */}
        <path d={path} stroke="rgba(255,255,255,0.15)" strokeWidth="1.5" strokeDasharray="4 4" fill="none" />
        
        {/* Flow line animated overlay */}
        <path className="flow-dash" d={path} stroke="#ffffff" strokeWidth="1.5" strokeDasharray="10 10" fill="none" opacity="0.4" />
        
        {/* Items badge */}
        {conn.items && !conn.branch && (
          <g transform={`translate(${(from.outX + to.inX)/2}, ${from.outY})`}>
            <rect x="-24" y="-10" width="48" height="20" rx="10" fill="#111" stroke="rgba(255,255,255,0.2)" />
            <text x="0" y="3" fill="#888" fontSize="9" textAnchor="middle" alignmentBaseline="middle" fontFamily="monospace">{conn.items}</text>
          </g>
        )}
      </g>
    );
  };

  return (
    <div ref={containerRef} className="bg-[#050505] w-full relative overflow-hidden font-sans border-t border-white/5 py-32">
      <div className="w-[90%] max-w-[1600px] mx-auto z-10 relative">
        
        {/* Header aligned left to match "armory" style */}
        <div className="mb-12 wf-element">
          <div className="flex items-center gap-3 text-white/40 mb-6">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M4 15l8-8 8 8" />
            </svg>
            <span className="text-[10px] tracking-[0.2em] uppercase font-mono">Our Product</span>
          </div>
          <h2 className="text-5xl md:text-6xl font-medium tracking-tight mb-6 text-white leading-tight max-w-xl">
            Build logic at scale
          </h2>
          <p className="text-white/50 max-w-xl text-base md:text-lg leading-relaxed">
            Design, deploy, and manage sophisticated logistics workflows through an intuitive visual interface. No complex coding—just pure supply chain logic.
          </p>
        </div>

        {/* BENTO GRID LAYOUT */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        
          {/* Workflow Editor Frame */}
          <div className="lg:col-span-3 wf-element border border-white/10 bg-[#0a0a0a] rounded-xl overflow-hidden shadow-[0_30px_100px_rgba(0,0,0,0.8)] flex flex-col md:flex-row h-[700px] relative">
          
          {/* Left Sidebar */}
          <div className="w-full md:w-[280px] border-b md:border-b-0 md:border-r border-white/10 bg-[#0c0c0c] flex flex-col z-20 shrink-0">
            
            {/* Top Tabs */}
            <div className="p-6 pb-4">
              <div className="flex gap-2">
                <button className="flex-1 bg-white text-black text-xs font-bold py-3 rounded-md tracking-wide hover:bg-gray-200 transition-colors">
                  AI AGENT
                </button>
                <button className="flex-1 border border-white/10 bg-white/5 text-white/60 text-xs font-bold py-3 rounded-md tracking-wide hover:text-white hover:bg-white/10 transition-colors">
                  AI CHAT
                </button>
              </div>
            </div>

            <div className="px-6 flex-1">
              <div className="text-[10px] tracking-widest uppercase text-white/40 font-mono mb-4 mt-4">
                Stack
              </div>
              
              <div className="grid grid-cols-3 gap-3">
                {['order', 'filter', 'agent', 'code', 'webhook', 'condition', 'telegram', 'sms', '+'].map((icon, i) => (
                  <div key={i} className="aspect-square border border-white/10 rounded-lg bg-[#111] flex items-center justify-center text-white/50 hover:text-white hover:bg-white/10 hover:border-white/30 transition-all cursor-pointer group shadow-sm">
                    {icon === '+' ? (
                      <span className="text-xl font-light">+</span>
                    ) : (
                      <WorkflowIcon type={icon} className="w-5 h-5 group-hover:scale-110 transition-transform" />
                    )}
                  </div>
                ))}
              </div>
            </div>
            
            <div className="p-6 border-t border-white/10 text-[10px] text-white/30 font-mono uppercase tracking-widest">
              Auto <span className="opacity-0">saving...</span>
            </div>
          </div>

          {/* Canvas Area */}
          <div className="flex-1 relative bg-[#0a0a0a] overflow-hidden" style={{ backgroundImage: 'radial-gradient(circle at center, rgba(255,255,255,0.05) 1px, transparent 1px)', backgroundSize: '24px 24px' }}>
            
            {/* Top Canvas Bar */}
            <div className="absolute top-0 left-0 right-0 p-6 flex gap-4 z-20">
              <div className="bg-[#111] border border-white/10 rounded-md px-4 py-2 text-xs font-mono text-white/60 flex items-center gap-2 shadow-xl hover:bg-[#1a1a1a] transition-colors cursor-pointer">
                <span>AGENT MODE</span>
                <div className="w-2 h-2 bg-white rounded-sm rotate-45 ml-2"></div>
              </div>
              <div className="bg-[#111] border border-white/10 rounded-md px-4 py-2 text-xs font-mono text-white/60 flex items-center gap-2 shadow-xl hover:bg-[#1a1a1a] transition-colors cursor-pointer">
                <span>UNTITLED</span>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="ml-2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg>
              </div>
            </div>

            {/* SVG Connection Lines & Overlays */}
            <svg className="absolute inset-0 w-full h-full pointer-events-none z-0">
              <defs>
                <marker id="port-dot" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4">
                  <circle cx="5" cy="5" r="4" fill="#fff" />
                </marker>
                <marker id="port-ring" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6">
                  <circle cx="5" cy="5" r="4" fill="#0a0a0a" stroke="#fff" strokeWidth="1.5" />
                </marker>
              </defs>
              
              {CONNECTIONS.map((conn, i) => renderConnection(conn, i))}
            </svg>

            {/* Nodes */}
            <div className="absolute inset-0 overflow-auto">
              {NODES.map((node) => {
                const width = 'w-[150px]';
                return (
                  <div 
                    key={node.id}
                    className="absolute z-10 flex flex-col items-center wf-element group cursor-pointer"
                    style={{ left: node.x, top: node.y }}
                  >
                    {/* The Node Shape */}
                    <div className={`
                      ${width} h-12 ${node.type === 'start' ? 'rounded-l-[20px] rounded-r-[8px]' : 'rounded-[8px]'} bg-[#0d0d0d] border border-white shadow-xl 
                      flex items-center relative transition-all duration-300
                      group-hover:border-white/70 group-hover:bg-[#151515] group-hover:-translate-y-1 group-hover:shadow-[0_10px_30px_rgba(255,255,255,0.1)]
                    `}>
                      {/* Left Port */}
                      {node.type !== 'start' && <div className="absolute -left-1 w-2 h-2 rounded-full bg-white"></div>}
                      
                      {/* Right Port */}
                      <div className="absolute -right-1 w-2 h-2 rounded-full bg-white"></div>

                      <div className="flex items-center gap-3 w-full px-4">
                        <div className="shrink-0 text-white flex items-center justify-center">
                          <WorkflowIcon type={node.icon} className="w-4 h-4" />
                        </div>
                        <div className="flex flex-col text-left truncate leading-tight">
                          <span className="text-[11px] font-mono text-white/90">{node.title}</span>
                          {node.subtitle && <span className="text-[9px] font-sans text-white/50 tracking-wide">{node.subtitle}</span>}
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>

            {/* Bottom Right Floating Badge (like "Made in Framer") */}
            <div className="absolute bottom-6 right-6 z-30 wf-element">
              <div className="bg-white text-black px-4 py-2 rounded-full text-xs font-bold flex items-center gap-2 shadow-xl hover:scale-105 transition-transform cursor-pointer">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M4 15l8-8 8 8" /></svg>
                Pegasus Platform
              </div>
            </div>

          </div>
        </div>

        {/* Close BENTO GRID LAYOUT */}
        </div>

        {/* Bento Cards Bottom Row */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-6">
        <div className="lg:col-span-1 wf-element border border-white/10 bg-[#0a0a0a] p-8 rounded-2xl flex flex-col justify-between hover:border-white/20 transition-all shadow-lg overflow-hidden relative group">
          <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/10 blur-[50px] group-hover:bg-blue-500/20 transition-all" />
          <div>
            <div className="flex items-center gap-2 text-white/40 text-xs font-mono mb-6 uppercase tracking-widest">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
              Processing Speed
            </div>
            <div className="text-4xl font-light tracking-tight text-white mb-2">12.4<span className="text-2xl text-white/50 ml-1">ms</span></div>
            <div className="text-sm text-white/50">Average event latency</div>
          </div>
          <div className="h-16 mt-6 border-b border-white/10 relative">
            <svg className="absolute inset-0 w-full h-full" preserveAspectRatio="none" viewBox="0 0 100 40">
              <path d="M0 30 L10 25 L20 35 L30 15 L40 25 L50 10 L60 20 L70 5 L80 15 L90 5 L100 0" fill="none" stroke="rgba(59,130,246,0.5)" strokeWidth="2" vectorEffect="non-scaling-stroke" />
              <path d="M0 40 L10 35 L20 40 L30 25 L40 35 L50 20 L60 30 L70 15 L80 25 L90 15 L100 10" fill="none" stroke="rgba(255,255,255,0.1)" strokeWidth="2" vectorEffect="non-scaling-stroke" />
            </svg>
          </div>
        </div>

        <div className="lg:col-span-1 wf-element border border-white/10 bg-[#0a0a0a] p-8 rounded-2xl flex flex-col justify-between hover:border-white/20 transition-all shadow-lg overflow-hidden relative group">
          <div className="absolute bottom-0 left-0 w-32 h-32 bg-green-500/10 blur-[50px] group-hover:bg-green-500/20 transition-all" />
          <div className="flex items-center gap-2 text-white/40 text-xs font-mono mb-8 uppercase tracking-widest">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path></svg>
            System Status
          </div>
          <div className="flex-1 flex flex-col justify-center gap-5">
            {[
              { name: 'Core API', status: 'Operational', color: 'bg-green-500' },
              { name: 'Edge Network', status: 'Operational', color: 'bg-green-500' },
              { name: 'AI Agents', status: 'Processing', color: 'bg-blue-500' },
            ].map((item, i) => (
              <div key={i} className="flex items-center justify-between">
                <span className="text-sm text-white/80">{item.name}</span>
                <div className="flex items-center gap-2">
                  <span className="text-[10px] text-white/40 uppercase font-mono tracking-wide">{item.status}</span>
                  <div className={`w-1.5 h-1.5 rounded-full ${item.color} shadow-[0_0_8px_${item.color}]`}></div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="lg:col-span-1 wf-element border border-white/10 bg-[#0a0a0a] p-8 rounded-2xl flex flex-col justify-between hover:border-white/20 transition-all shadow-lg overflow-hidden relative group">
          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-48 h-48 bg-purple-500/10 blur-[60px] group-hover:bg-purple-500/20 transition-all" />
          <div className="flex items-center gap-2 text-white/40 text-xs font-mono mb-6 uppercase tracking-widest relative z-10">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>
            Global Availability
          </div>
          <div className="relative z-10 flex-1 flex flex-col justify-end">
            <div className="flex items-baseline gap-2 mb-2">
              <span className="text-5xl font-light tracking-tight text-white">99.99</span>
              <span className="text-xl text-white/50">%</span>
            </div>
            <div className="text-sm text-white/50">Uptime over last 90 days</div>
            
            <div className="mt-6 flex gap-1 h-8">
              {[...Array(30)].map((_, i) => (
                <div 
                  key={i} 
                  className={`flex-1 rounded-sm ${i === 14 ? 'bg-yellow-500/50' : 'bg-green-500/40'}`} 
                  title={i === 14 ? 'Minor degraded performance' : 'No downtime'}
                />
              ))}
            </div>
          </div>
        </div>
        </div>
        
        {/* Row 2 of Bento Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mt-6">
          {/* Card 4: Integration Hub */}
          <div className="lg:col-span-1 wf-element border border-white/10 bg-[#0a0a0a] p-8 rounded-2xl flex flex-col justify-between hover:border-white/20 transition-all shadow-lg overflow-hidden relative group">
            <div className="absolute top-0 left-0 w-32 h-32 bg-orange-500/10 blur-[50px] group-hover:bg-orange-500/20 transition-all" />
            <div className="flex items-center gap-2 text-white/40 text-xs font-mono mb-8 uppercase tracking-widest relative z-10">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
              Integration Hub
            </div>
            <div className="flex-1 flex items-center justify-center gap-4 relative z-10">
              <div className="w-12 h-12 rounded-full bg-[#111] border border-white/10 flex items-center justify-center shadow-lg group-hover:-translate-y-1 transition-transform">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.5"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
              </div>
              <div className="w-16 h-16 rounded-full bg-white/5 border border-white/20 flex items-center justify-center shadow-xl group-hover:scale-110 transition-transform relative z-10">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.5"><path d="M22 12h-4l-3 9L9 3l-3 9H2"></path></svg>
              </div>
              <div className="w-12 h-12 rounded-full bg-[#111] border border-white/10 flex items-center justify-center shadow-lg group-hover:-translate-y-1 transition-transform">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.5"><rect x="2" y="2" width="20" height="20" rx="5" ry="5"></rect><path d="M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37z"></path><line x1="17.5" y1="6.5" x2="17.51" y2="6.5"></line></svg>
              </div>
            </div>
            <div className="mt-6 text-sm text-white/50 text-center">Seamlessly connect your entire stack</div>
          </div>

          {/* Card 5: Data Processed */}
          <div className="lg:col-span-1 wf-element border border-white/10 bg-[#0a0a0a] p-8 rounded-2xl flex flex-col justify-between hover:border-white/20 transition-all shadow-lg overflow-hidden relative group">
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-40 h-40 bg-pink-500/10 blur-[50px] group-hover:bg-pink-500/20 transition-all" />
            <div className="flex items-center gap-2 text-white/40 text-xs font-mono mb-6 uppercase tracking-widest relative z-10">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
              Volume Handled
            </div>
            <div className="relative z-10">
              <div className="text-5xl font-light tracking-tight text-white mb-2">2.4<span className="text-2xl text-white/50 ml-1">B+</span></div>
              <div className="text-sm text-white/50 mb-6">Events processed monthly</div>
              
              <div className="flex items-center gap-2">
                <div className="text-xs font-mono text-green-400 bg-green-400/10 px-2 py-1 rounded">↑ 18%</div>
                <span className="text-xs text-white/30 font-mono">vs last month</span>
              </div>
            </div>
          </div>

          {/* Card 6: Security */}
          <div className="lg:col-span-1 wf-element border border-white/10 bg-[#0a0a0a] p-8 rounded-2xl flex flex-col justify-between hover:border-white/20 transition-all shadow-lg overflow-hidden relative group">
            <div className="absolute bottom-0 right-0 w-32 h-32 bg-indigo-500/10 blur-[50px] group-hover:bg-indigo-500/20 transition-all" />
            <div className="flex items-center gap-2 text-white/40 text-xs font-mono mb-8 uppercase tracking-widest relative z-10">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
              Security
            </div>
            <div className="flex-1 flex flex-col justify-center relative z-10">
              <div className="text-2xl font-light text-white mb-4">Enterprise Grade</div>
              <ul className="space-y-3">
                {[
                  'SOC 2 Type II Certified',
                  'End-to-end Encryption',
                  'Role-based Access Control'
                ].map((feature, i) => (
                  <li key={i} className="flex items-center gap-3 text-sm text-white/70">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#10b981" strokeWidth="2"><polyline points="20 6 9 17 4 12"></polyline></svg>
                    {feature}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
