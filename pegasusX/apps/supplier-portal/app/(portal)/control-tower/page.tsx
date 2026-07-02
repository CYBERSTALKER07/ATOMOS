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

// Mock Data for Network Graph
const mockNodes: NetworkNode[] = [
  { id: "WH-1", type: "warehouse", label: "Central Hub", status: "active" },
  { id: "WH-2", type: "warehouse", label: "East DC", status: "active" },
  { id: "RT-1", type: "retailer", label: "Store Alpha", status: "busy" },
  { id: "RT-2", type: "retailer", label: "Store Beta", status: "idle" },
  { id: "RT-3", type: "retailer", label: "Store Gamma", status: "active" },
  { id: "DR-1", type: "driver", label: "Driver 104", status: "active" },
  { id: "DR-2", type: "driver", label: "Driver 211", status: "busy" },
  { id: "DR-3", type: "driver", label: "Driver 305", status: "active" },
];

const mockLinks: NetworkLink[] = [
  { source: "WH-1", target: "RT-1", value: 10 },
  { source: "WH-1", target: "RT-2", value: 5 },
  { source: "WH-2", target: "RT-3", value: 8 },
  { source: "WH-1", target: "DR-1", value: 2 },
  { source: "DR-1", target: "RT-1", value: 4 },
  { source: "WH-2", target: "DR-2", value: 3 },
  { source: "DR-2", target: "RT-3", value: 6 },
  { source: "WH-1", target: "DR-3", value: 1 },
  { source: "DR-3", target: "RT-2", value: 5 },
];

// Mock Data for Map
const generateH3Data = () => {
  const data = [];
  const centerLat = 37.74;
  const centerLng = -122.4;
  for (let i = 0; i < 50; i++) {
    const lat = centerLat + (Math.random() - 0.5) * 0.5;
    const lng = centerLng + (Math.random() - 0.5) * 0.5;
    const hex = latLngToCell(lat, lng, 8);
    data.push({ hex, count: Math.floor(Math.random() * 100) });
  }
  return data;
};

// Mock Data for Charts
const performanceData = [
  { name: "Q1", actual: 4000, plan: 2400 },
  { name: "Q2", actual: 3000, plan: 1398 },
  { name: "Q3", actual: 2000, plan: 9800 },
  { name: "Q4", actual: 2780, plan: 3908 },
];

const scenariosData = [
  { name: "Revenue", baseline: 4000, upside: 5000 },
  { name: "Cash Flow", baseline: 3000, upside: 3500 },
  { name: "Gross Margin", baseline: 2000, upside: 3000 },
];

export default function ControlTowerPage() {
  const [view, setView] = useState<"network" | "map">("network");
  
  const supplierId = "sup-demo-1";
  
  const { networkNodes, networkLinks, h3Data: wsH3Data } = useControlTowerWebSocket(supplierId);

  const displayNodes = networkNodes.length > 0 ? networkNodes : mockNodes;
  const displayLinks = networkLinks.length > 0 ? networkLinks : mockLinks;

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
