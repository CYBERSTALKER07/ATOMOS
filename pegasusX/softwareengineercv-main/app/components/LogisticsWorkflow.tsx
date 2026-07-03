'use client';

import { useEffect, useRef } from 'react';
import PageSection from './layout/PageSection';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

const NODES = [
  { id: 'n1', x: 200, y: 200, type: 'square', icon: 'order', title: 'Order Sync', subtitle: 'Shopify Trigger' },
  { id: 'n2', x: 400, y: 200, type: 'square', icon: 'filter', title: 'Validate', subtitle: 'Inventory' },
  { id: 'n3', x: 600, y: 200, type: 'rect', icon: 'agent', title: 'Pegasus AI', subtitle: 'Routing Agent' },
  { id: 'n4', x: 850, y: 200, type: 'square', icon: 'code', title: 'Optimize', subtitle: 'Algorithm' },
  { id: 'n5', x: 1050, y: 200, type: 'square', icon: 'webhook', title: 'Dispatch', subtitle: 'Manifest' },
  { id: 'n6', x: 600, y: 350, type: 'square', icon: 'telegram', title: 'Telegram', subtitle: 'Alert Driver' },
  { id: 'n7', x: 850, y: 350, type: 'square', icon: 'condition', title: 'If', subtitle: 'Traffic > 20m' },
  { id: 'n8', x: 1050, y: 350, type: 'square', icon: 'sms', title: 'SMS', subtitle: 'Retailer' },
];

const CONNECTIONS = [
  { from: 'n1', to: 'n2', items: '1 item' },
  { from: 'n2', to: 'n3', items: '1 item' },
  { from: 'n3', to: 'n4', items: '1 item' },
  { from: 'n4', to: 'n5', items: '1 item' },
  { from: 'n3', to: 'n6', branch: true },
  { from: 'n6', to: 'n7' },
  { from: 'n7', to: 'n8' },
];

function WorkflowIcon({ type, className = "" }: { type: string, className?: string }) {
  switch (type) {
    case 'order':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M6 2L3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4z"></path><line x1="3" y1="6" x2="21" y2="6"></line><path d="M16 10a4 4 0 0 1-8 0"></path></svg>;
    case 'filter':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"></polygon></svg>;
    case 'agent':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>;
    case 'code':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>;
    case 'webhook':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path><polyline points="15 3 21 3 21 9"></polyline><line x1="10" y1="14" x2="21" y2="3"></line></svg>;
    case 'condition':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><line x1="9" y1="9" x2="15" y2="15"></line><line x1="15" y1="9" x2="9" y2="15"></line></svg>;
    case 'telegram':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg>;
    case 'sms':
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"></path></svg>;
    default:
      return <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"></circle></svg>;
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

  // Helper to get node coordinates including their width/height offset to connect ports
  const getNodePorts = (id: string) => {
    const node = NODES.find(n => n.id === id);
    if (!node) return { outX: 0, outY: 0, inX: 0, inY: 0 };
    
    const w = node.type === 'rect' ? 160 : 64;
    const h = 64;
    return {
      outX: node.x + w + 8, // 8px for port
      outY: node.y + h / 2,
      inX: node.x - 8,
      inY: node.y + h / 2,
      centerX: node.x + w / 2,
      bottomY: node.y + h + 8
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

        {/* Workflow Editor Frame */}
        <div className="wf-element border border-white/10 bg-[#0a0a0a] rounded-xl overflow-hidden shadow-[0_30px_100px_rgba(0,0,0,0.8)] flex flex-col md:flex-row h-[700px] relative">
          
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
                const isRect = node.type === 'rect';
                const width = isRect ? 'w-[160px]' : 'w-16';
                return (
                  <div 
                    key={node.id}
                    className="absolute z-10 flex flex-col items-center wf-element group cursor-pointer"
                    style={{ left: node.x, top: node.y }}
                  >
                    {/* The Node Shape */}
                    <div className={`
                      ${width} h-16 rounded-2xl bg-[#0e0e0e] border-2 border-white/10 shadow-xl 
                      flex items-center justify-center relative transition-all duration-300
                      group-hover:border-white/40 group-hover:bg-[#161616] group-hover:-translate-y-1 group-hover:shadow-[0_10px_30px_rgba(255,255,255,0.1)]
                    `}>
                      {/* Left Port */}
                      <div className="absolute -left-1.5 w-3 h-3 rounded-full bg-[#0e0e0e] border border-white/40 group-hover:border-white"></div>
                      
                      {/* Right Port */}
                      <div className="absolute -right-1.5 w-3 h-3 rounded-full bg-[#0e0e0e] border border-white/40 group-hover:border-white"></div>
                      
                      {/* Top Port (for branching sometimes) - purely visual */}
                      <div className="absolute -top-1.5 w-3 h-3 rounded-full bg-[#0e0e0e] border border-white/40 opacity-0 group-hover:opacity-100 transition-opacity"></div>
                      
                      {/* Bottom Port (for branching) */}
                      <div className="absolute -bottom-1.5 w-3 h-3 rounded-full bg-[#0e0e0e] border border-white/40 group-hover:border-white"></div>

                      <div className="flex items-center gap-3 w-full px-4">
                        <div className="shrink-0 text-white flex items-center justify-center">
                          <WorkflowIcon type={node.icon} className="w-6 h-6" />
                        </div>
                        {isRect && (
                          <div className="flex-1 text-left truncate">
                            <div className="text-[13px] font-semibold text-white tracking-wide">{node.title}</div>
                            <div className="text-[10px] text-white/50">{node.subtitle}</div>
                          </div>
                        )}
                      </div>
                    </div>
                    
                    {/* Node Text Below */}
                    {!isRect && (
                      <div className="mt-4 text-center">
                        <div className="text-xs font-mono text-white/80">{node.title}</div>
                        <div className="text-[10px] font-mono text-white/40 mt-1">{node.subtitle}</div>
                      </div>
                    )}
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

      </div>
    </div>
  );
}
