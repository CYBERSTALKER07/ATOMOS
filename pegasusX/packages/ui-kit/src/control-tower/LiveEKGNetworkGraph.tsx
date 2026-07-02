"use client";

import React, { useEffect, useRef } from "react";
import * as d3 from "d3";

export interface NetworkNode extends d3.SimulationNodeDatum {
  id: string;
  type: "warehouse" | "retailer" | "driver";
  label: string;
  status: "active" | "idle" | "busy";
}

export interface NetworkLink extends d3.SimulationLinkDatum<NetworkNode> {
  source: string | NetworkNode;
  target: string | NetworkNode;
  value: number;
}

export interface LiveEKGNetworkGraphProps {
  nodes: NetworkNode[];
  links: NetworkLink[];
  width?: number;
  height?: number;
}

export function LiveEKGNetworkGraph({
  nodes,
  links,
  width = 800,
  height = 600,
}: LiveEKGNetworkGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    if (!svgRef.current) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove(); // Clear previous

    const color = d3.scaleOrdinal<string>()
      .domain(["warehouse", "retailer", "driver"])
      .range(["#10b981", "#3b82f6", "#f59e0b"]); // Emerald, Blue, Amber

    const simulation = d3.forceSimulation<NetworkNode>(nodes)
      .force("link", d3.forceLink<NetworkNode, NetworkLink>(links).id((d) => d.id).distance(150))
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("collide", d3.forceCollide().radius(40));

    // Glow filter
    const defs = svg.append("defs");
    const filter = defs.append("filter").attr("id", "glow");
    filter.append("feGaussianBlur").attr("stdDeviation", "3.5").attr("result", "coloredBlur");
    const feMerge = filter.append("feMerge");
    feMerge.append("feMergeNode").attr("in", "coloredBlur");
    feMerge.append("feMergeNode").attr("in", "SourceGraphic");

    const linkGroup = svg.append("g").attr("stroke-opacity", 0.6);
    
    // Draw edges
    const link = linkGroup
      .selectAll("path")
      .data(links)
      .join("path")
      .attr("fill", "none")
      .attr("stroke", "#4b5563")
      .attr("stroke-width", 2)
      .attr("class", "ekg-link");

    const nodeGroup = svg.append("g").attr("stroke", "#fff").attr("stroke-width", 1.5);

    // Draw nodes
    const node = nodeGroup
      .selectAll("g")
      .data(nodes)
      .join("g")
      .call(drag(simulation) as any);

    node.each(function(d) {
      const g = d3.select(this);
      
      if (d.type === "warehouse") {
        // Triangle
        g.append("path")
          .attr("d", d3.symbol().type(d3.symbolTriangle).size(600)())
          .attr("fill", color(d.type))
          .style("filter", "url(#glow)");
      } else if (d.type === "retailer") {
        // Square
        g.append("path")
          .attr("d", d3.symbol().type(d3.symbolSquare).size(600)())
          .attr("fill", color(d.type))
          .style("filter", "url(#glow)");
      } else {
        // Circle
        g.append("circle")
          .attr("r", 15)
          .attr("fill", color(d.type))
          .style("filter", "url(#glow)");
      }

      g.append("text")
        .text(d.label)
        .attr("x", 20)
        .attr("y", 5)
        .attr("font-family", "sans-serif")
        .attr("font-size", "12px")
        .attr("fill", "#e5e7eb")
        .attr("stroke", "none");
    });

    // Pulsing effect along links
    function pulse() {
      linkGroup.selectAll(".ekg-pulse").remove();

      linkGroup.selectAll(".ekg-link").each(function(d: any, i) {
        const path = this as SVGPathElement;
        const l = path.getTotalLength();
        
        // Randomly decide if a pulse happens on this tick to simulate traffic
        if (Math.random() > 0.5) {
          const p = linkGroup.append("circle")
            .attr("r", 4)
            .attr("fill", "#10b981")
            .attr("class", "ekg-pulse")
            .style("filter", "url(#glow)");
            
          p.transition()
            .duration(1500 + Math.random() * 1000)
            .ease(d3.easeLinear)
            .attrTween("transform", function() {
              return function(t) {
                const pt = path.getPointAtLength(t * l);
                return `translate(\${pt.x},\${pt.y})`;
              };
            })
            .remove();
        }
      });

      d3.timeout(pulse, 2000);
    }
    
    pulse();

    simulation.on("tick", () => {
      link.attr("d", (d: any) => {
        // Create curved paths
        const dx = d.target.x - d.source.x;
        const dy = d.target.y - d.source.y;
        const dr = Math.sqrt(dx * dx + dy * dy);
        return `M\${d.source.x},\${d.source.y}A\${dr},\${dr} 0 0,1 \${d.target.x},\${d.target.y}`;
      });

      node.attr("transform", (d) => `translate(\${d.x},\${d.y})`);
    });

    return () => {
      simulation.stop();
    };
  }, [nodes, links, width, height]);

  // Drag functionality
  const drag = (simulation: any) => {
    function dragstarted(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }
    function dragged(event: any, d: any) {
      d.fx = event.x;
      d.fy = event.y;
    }
    function dragended(event: any, d: any) {
      if (!event.active) simulation.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }
    return d3.drag()
      .on("start", dragstarted)
      .on("drag", dragged)
      .on("end", dragended);
  };

  return (
    <div className="w-full h-full min-h-[500px] bg-black rounded-xl overflow-hidden border border-white/10 shadow-2xl relative">
      <svg ref={svgRef} width="100%" height="100%" viewBox={`0 0 \${width} \${height}`} preserveAspectRatio="xMidYMid meet" />
      <div className="absolute top-4 left-4 flex gap-4 text-xs font-mono text-gray-400">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-emerald-500 rounded-sm"></div> Warehouse
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-blue-500 rounded-sm"></div> Retailer
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-amber-500 rounded-full"></div> Driver
        </div>
      </div>
    </div>
  );
}
