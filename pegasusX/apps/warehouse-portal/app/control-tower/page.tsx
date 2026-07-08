"use client";

import React, { useState, useEffect } from "react";
import {
  LiveEKGNetworkGraph,
  HexagonalControlTowerMap,
  GlassmorphismPanel,
  NetworkNode,
  NetworkLink,
  useControlTowerWebSocket,
} from "@pegasusx/ui-kit/control-tower";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Legend,
} from "recharts";
import { cellToBoundary, latLngToCell } from "h3-js";



// Removed Mock Data
const generateH3Data = () => {
  return [];
};

const performanceData: Record<string, unknown>[] = [];
const scenariosData: Record<string, unknown>[] = [];

export default function ControlTowerPage() {
  const [view, setView] = useState<"network" | "map">("network");
  
  const supplierId = "sup-demo-1";
  
  const { networkNodes, networkLinks, h3Data: wsH3Data } = useControlTowerWebSocket(supplierId);

  const displayNodes = networkNodes;
  const displayLinks = networkLinks;

  const [h3Data, setH3Data] = useState<{hex: string, count: number}[]>([]);

  useEffect(() => {
    setH3Data(generateH3Data());
  }, []);

  const displayH3Data = wsH3Data.length > 0 ? wsH3Data : h3Data;

  return (
    <div className="relative w-full h-[calc(100vh-64px)] bg-[#0a0a0a] text-white overflow-hidden p-6 flex flex-col gap-6">
      
      {/* Header / Tabs */}
      <div className="flex items-center justify-between z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">Global Control Tower</h1>
          <p className="text-sm text-gray-400">Real-time network telematics and predictive intelligence.</p>
        </div>
        <div className="flex bg-white/5 border border-white/10 rounded-lg p-1">
          <button
            onClick={() => setView("network")}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-colors \${view === "network" ? "bg-emerald-500/20 text-emerald-400" : "text-gray-400 hover:text-white"}`}
          >
            Live Network Graph
          </button>
          <button
            onClick={() => setView("map")}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-colors \${view === "map" ? "bg-emerald-500/20 text-emerald-400" : "text-gray-400 hover:text-white"}`}
          >
            Spatial Map (H3)
          </button>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 relative rounded-xl overflow-hidden border border-white/10 shadow-2xl">
        {view === "network" ? (
          <LiveEKGNetworkGraph nodes={displayNodes} links={displayLinks} width={1200} height={800} />
        ) : (
          <HexagonalControlTowerMap data={displayH3Data} />
        )}

        {/* Floating Glassmorphism Panel - Left */}
        <GlassmorphismPanel 
          className="absolute top-6 left-6 w-80 h-64 flex flex-col"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.2 }}
        >
          <h3 className="text-xs font-bold text-gray-400 tracking-wider mb-4 uppercase">Actual vs Plan</h3>
          <div className="flex-1 min-h-0">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={performanceData} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" vertical={false} />
                <XAxis dataKey="name" stroke="#666" tick={{ fill: "#666", fontSize: 12 }} />
                <YAxis stroke="#666" tick={{ fill: "#666", fontSize: 12 }} />
                <Tooltip 
                  contentStyle={{ backgroundColor: "#111", border: "1px solid #333", borderRadius: "8px" }}
                  itemStyle={{ color: "#fff" }}
                />
                <Line type="monotone" dataKey="actual" stroke="#10b981" strokeWidth={2} dot={{ r: 4, fill: "#10b981" }} />
                <Line type="monotone" dataKey="plan" stroke="#f59e0b" strokeWidth={2} strokeDasharray="5 5" dot={{ r: 4, fill: "#f59e0b" }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </GlassmorphismPanel>

        {/* Floating Glassmorphism Panel - Right */}
        <GlassmorphismPanel 
          className="absolute bottom-6 right-6 w-96 h-64 flex flex-col"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
        >
          <h3 className="text-xs font-bold text-gray-400 tracking-wider mb-4 uppercase">Baseline vs Upside Scenarios</h3>
          <div className="flex-1 min-h-0">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={scenariosData} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" vertical={false} />
                <XAxis dataKey="name" stroke="#666" tick={{ fill: "#666", fontSize: 12 }} />
                <YAxis stroke="#666" tick={{ fill: "#666", fontSize: 12 }} />
                <Tooltip 
                  contentStyle={{ backgroundColor: "#111", border: "1px solid #333", borderRadius: "8px" }}
                  cursor={{ fill: "rgba(255,255,255,0.05)" }}
                />
                <Legend iconType="circle" wrapperStyle={{ fontSize: "12px", color: "#666" }} />
                <Bar dataKey="baseline" fill="#10b981" radius={[4, 4, 0, 0]} />
                <Bar dataKey="upside" fill="#f59e0b" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </GlassmorphismPanel>

      </div>
    </div>
  );
}
