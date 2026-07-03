'use client';

import { useEffect, useRef, useState } from 'react';
import PageSection from './layout/PageSection';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

type WorkflowRole = 'supplier' | 'warehouse' | 'retailer' | 'fleet';

const WORKFLOW_DATA = {
  supplier: {
    nodes: [
      { id: 's1', x: 80, y: 150, type: 'start', icon: 'email', title: 'Order Received', subtitle: 'EDI/API' },
      { id: 's2', x: 280, y: 150, type: 'rect', icon: 'agent', title: 'AI Inventory Check', subtitle: 'Auto-allocation' },
      { id: 's3', x: 480, y: 80, type: 'rect', icon: 'if', title: 'Stock Available?', subtitle: 'Condition' },
      { id: 's4', x: 680, y: 80, type: 'rect', icon: 'code', title: 'Release to Floor', subtitle: 'WMS Sync' },
      { id: 's5', x: 480, y: 220, type: 'rect', icon: 'edit', title: 'Backorder', subtitle: 'Manual Review' },
      { id: 's6', x: 880, y: 80, type: 'rect', icon: 'agent', title: 'Quality Assurance', subtitle: 'Vision AI' },
      { id: 's7', x: 1080, y: 80, type: 'rect', icon: 'send_email', title: 'Dispatch Alert', subtitle: 'To Carrier' },
    ],
    connections: [
      { from: 's1', to: 's2' },
      { from: 's2', to: 's3' },
      { from: 's3', to: 's4' },
      { from: 's2', to: 's5', branch: true },
      { from: 's4', to: 's6', items: '2 pallets' },
      { from: 's6', to: 's7' },
    ]
  },
  warehouse: {
    nodes: [
      { id: 'w1', x: 80, y: 150, type: 'start', icon: 'telegram', title: 'Inbound Alert', subtitle: 'Carrier ETA' },
      { id: 'w2', x: 280, y: 150, type: 'rect', icon: 'agent', title: 'Dock Scheduling', subtitle: 'AI Optimizer' },
      { id: 'w3', x: 480, y: 150, type: 'rect', icon: 'code', title: 'Scan & Sort', subtitle: 'IoT Sensors' },
      { id: 'w4', x: 680, y: 80, type: 'rect', icon: 'if', title: 'QC Pass?', subtitle: 'Condition' },
      { id: 'w5', x: 880, y: 80, type: 'rect', icon: 'edit', title: 'Putaway', subtitle: 'Forklift Task' },
      { id: 'w6', x: 680, y: 220, type: 'rect', icon: 'send_email', title: 'Quarantine', subtitle: 'Alert Supplier' },
      { id: 'w7', x: 1080, y: 80, type: 'rect', icon: 'agent', title: 'Inventory Sync', subtitle: 'ERP Update' },
    ],
    connections: [
      { from: 'w1', to: 'w2' },
      { from: 'w2', to: 'w3' },
      { from: 'w3', to: 'w4' },
      { from: 'w3', to: 'w6', branch: true },
      { from: 'w4', to: 'w5' },
      { from: 'w5', to: 'w7' },
    ]
  },
  retailer: {
    nodes: [
      { id: 'r1', x: 80, y: 150, type: 'start', icon: 'if', title: 'Low Stock Alert', subtitle: 'POS System' },
      { id: 'r2', x: 280, y: 150, type: 'rect', icon: 'agent', title: 'Demand Forecast', subtitle: 'AI Predictor' },
      { id: 'r3', x: 480, y: 150, type: 'rect', icon: 'code', title: 'Auto-Reorder', subtitle: 'Create PO' },
      { id: 'r4', x: 680, y: 150, type: 'rect', icon: 'email', title: 'Supplier Conf', subtitle: 'EDI 855' },
      { id: 'r5', x: 880, y: 150, type: 'rect', icon: 'telegram', title: 'ASN Received', subtitle: 'Inbound Prep' },
      { id: 'r6', x: 1080, y: 150, type: 'rect', icon: 'edit', title: 'Shelf Restock', subtitle: 'Store Task' },
    ],
    connections: [
      { from: 'r1', to: 'r2' },
      { from: 'r2', to: 'r3' },
      { from: 'r3', to: 'r4' },
      { from: 'r4', to: 'r5' },
      { from: 'r5', to: 'r6' }
    ]
  },
  fleet: {
    nodes: [
      { id: 'f1', x: 80, y: 150, type: 'start', icon: 'agent', title: 'Route Optimizer', subtitle: 'AI Dispatch' },
      { id: 'f2', x: 280, y: 150, type: 'rect', icon: 'code', title: 'Vehicle Assign', subtitle: 'TMS' },
      { id: 'f3', x: 480, y: 80, type: 'rect', icon: 'if', title: 'Weather Clear?', subtitle: 'API Check' },
      { id: 'f4', x: 680, y: 80, type: 'rect', icon: 'telegram', title: 'Driver Brief', subtitle: 'Mobile App' },
      { id: 'f5', x: 480, y: 220, type: 'rect', icon: 'agent', title: 'Reroute', subtitle: 'Dynamic Path' },
      { id: 'f6', x: 880, y: 80, type: 'rect', icon: 'edit', title: 'Delivery Conf', subtitle: 'ePOD' },
      { id: 'f7', x: 1080, y: 80, type: 'rect', icon: 'send_email', title: 'Invoice Client', subtitle: 'Billing' },
    ],
    connections: [
      { from: 'f1', to: 'f2' },
      { from: 'f2', to: 'f3' },
      { from: 'f2', to: 'f5', branch: true },
      { from: 'f3', to: 'f4' },
      { from: 'f5', to: 'f4', branch: true },
      { from: 'f4', to: 'f6', items: 'Live Track' },
      { from: 'f6', to: 'f7' },
    ]
  }
};

const ROLE_LABELS = {
  supplier: 'SUPPLIER',
  warehouse: 'WAREHOUSE',
  retailer: 'RETAILER',
  fleet: 'FLEET'
};

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
  const canvasRef = useRef<HTMLDivElement>(null);
  const [activeRole, setActiveRole] = useState<WorkflowRole>('supplier');

  const { nodes, connections } = WORKFLOW_DATA[activeRole];

  useEffect(() => {
    let ctx = gsap.context(() => {
      // Re-trigger entrance animation on tab switch
      gsap.fromTo(".wf-node",
        { opacity: 0, y: 10 },
        { opacity: 1, y: 0, duration: 0.4, stagger: 0.03, ease: "power2.out" }
      );

      // Flow animation
      gsap.to(".flow-dash", {
        strokeDashoffset: -20,
        duration: 0.8,
        ease: "none",
        repeat: -1,
      });
    }, canvasRef); // Scope to canvas so we only animate canvas items on tab switch

    return () => ctx.revert();
  }, [activeRole]); // Re-run animation when role changes

  useEffect(() => {
    // Initial scroll trigger for the whole component
    let ctx = gsap.context(() => {
      const tl = gsap.timeline({
        scrollTrigger: {
          trigger: containerRef.current,
          start: "top 70%",
        }
      });

      tl.fromTo(".wf-header",
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.6, stagger: 0.1, ease: "power3.out" }
      );
    }, containerRef);

    return () => ctx.revert();
  }, []);

  const getNodePorts = (id: string) => {
    const node = nodes.find(n => n.id === id);
    if (!node) return { outX: 0, outY: 0, inX: 0, inY: 0, centerX: 0, bottomY: 0 };

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
      <g key={`${activeRole}-conn-${index}`} className="wf-node">
        {/* Base line */}
        <path d={path} stroke="rgba(255,255,255,0.15)" strokeWidth="1.5" strokeDasharray="4 4" fill="none" />

        {/* Flow line animated overlay */}
        <path className="flow-dash" d={path} stroke="#ffffff" strokeWidth="1.5" strokeDasharray="10 10" fill="none" opacity="0.4" />

        {/* Items badge */}
        {conn.items && !conn.branch && (
          <g transform={`translate(${(from.outX + to.inX) / 2}, ${from.outY})`}>
            <rect x="-30" y="-10" width="60" height="20" rx="10" fill="#111" stroke="rgba(255,255,255,0.2)" />
            <text x="0" y="3" fill="#888" fontSize="9" textAnchor="middle" alignmentBaseline="middle" fontFamily="monospace">{conn.items}</text>
          </g>
        )}
      </g>
    );
  };

  return (
    <div ref={containerRef} className="bg-black w-full relative overflow-hidden font-sans border-t border-white/5 py-32">
      <div className="w-[90%] max-w-[1600px] mx-auto z-10 relative">

        {/* Header aligned left to match "armory" style */}
        <div className="mb-12 wf-header">
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
            Design, deploy, and manage sophisticated logistics workflows across every facet of your ecosystem. Switch roles below to view distinct operational logic.
          </p>
        </div>

        {/* BENTO GRID LAYOUT */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

          {/* Workflow Editor Frame */}
          <div className="lg:col-span-3 wf-header border border-white/10 bg-[#0a0a0a] rounded-xl overflow-hidden shadow-[0_30px_100px_rgba(0,0,0,0.8)] flex flex-col md:flex-row h-[800px] relative">

            {/* Left Sidebar */}
            <div className="w-full md:w-[280px] border-b md:border-b-0 md:border-r border-white/10 bg-black flex flex-col z-20 shrink-0">

              {/* Top Tabs for Roles */}
              <div className="p-6 flex-1 flex flex-col gap-3">
                <div className="text-[10px] tracking-widest uppercase text-white/40 font-mono mb-2">
                  Workflow Role
                </div>
                <div className="flex flex-col gap-2">
                  {(Object.keys(ROLE_LABELS) as WorkflowRole[]).map((role) => (
                    <button
                      key={role}
                      onClick={() => setActiveRole(role)}
                      className={`
                        text-xs  py-4 rounded-none tracking-wide transition-colors text-left px-4 border
                        ${activeRole === role
                          ? 'bg-white text-black border-white'
                          : 'border-white/10 bg-black/5 text-white/60 hover:text-white hover:bg-white/10'}
                      `}
                    >
                      {ROLE_LABELS[role]}
                    </button>
                  ))}
                </div>
              </div>

              <div className="p-6 border-t border-white/10 text-[10px] text-white/30 font-mono uppercase tracking-widest">
                Auto <span className="opacity-0">saving...</span>
              </div>
            </div>

            {/* Canvas Area with Scroll */}
            <div className="flex-1 relative bg-[#0a0a0a] overflow-auto no-scrollbar" style={{ backgroundColor: 'black', backgroundSize: '24px 24px' }}>

              {/* Inner Canvas Container that holds absolute elements */}
              <div ref={canvasRef} className="relative min-w-[1300px] min-h-[800px] w-full h-full">

                {/* Top Canvas Bar (Sticky so it stays visible while scrolling) */}


                {/* SVG Connection Lines & Overlays */}
                <svg className="absolute inset-0 w-full h-full pointer-events-none z-0">
                  <defs>
                    <marker id="port-dot" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4">
                      <circle cx="5" cy="5" r="4" fill="#fff" />
                    </marker>
                  </defs>

                  {connections.map((conn, i) => renderConnection(conn, i))}
                </svg>

                {/* Nodes */}
                <div className="absolute inset-0 pt-20">
                  {nodes.map((node) => {
                    const width = 'w-[150px]';
                    return (
                      <div
                        key={`${activeRole}-${node.id}`}
                        className="absolute z-10 flex flex-col items-center wf-node group cursor-pointer"
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

                {/* Bottom Right Floating Badge */}


              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
