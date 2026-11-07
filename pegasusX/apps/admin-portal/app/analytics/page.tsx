"use client";

import React, { useState } from "react";
import { CopyableId } from "@/components/ui/CopyableId";

type RoleCategory = "ALL" | "SUPPLIER" | "WAREHOUSE" | "FLEET" | "RETAILER";

interface ScorecardMetric {
  name: string;
  category: "SUPPLIER" | "WAREHOUSE" | "FLEET" | "RETAILER";
  currentValue: string;
  targetValue: string;
  trend: "UP" | "DOWN" | "NEUTRAL";
  trendValue: string;
  status: "OPTIMAL" | "ATTENTION" | "CRITICAL";
  benchmarkDesc: string;
}

const SCORECARD_METRICS: ScorecardMetric[] = [
  // Supplier Metrics
  {
    name: "On-Time In-Full (OTIF) Delivery",
    category: "SUPPLIER",
    currentValue: "97.4%",
    targetValue: ">= 95.0%",
    trend: "UP",
    trendValue: "+1.8%",
    status: "OPTIMAL",
    benchmarkDesc: "SAP S/4HANA TM Gold Standard: Orders fulfilled without item omissions or stock-out splits.",
  },
  {
    name: "Factory Batch Zero-Defect Rate",
    category: "SUPPLIER",
    currentValue: "99.82%",
    targetValue: ">= 99.5%",
    trend: "UP",
    trendValue: "+0.15%",
    status: "OPTIMAL",
    benchmarkDesc: "Microbiological & packaging seal validation prior to release from factory gates.",
  },
  {
    name: "Soliq E-Factura Compliance",
    category: "SUPPLIER",
    currentValue: "100.0%",
    targetValue: "100.0%",
    trend: "NEUTRAL",
    trendValue: "0.0%",
    status: "OPTIMAL",
    benchmarkDesc: "State Tax Committee electronic fiscal receipt registration (Zero unfiscalized invoices).",
  },
  {
    name: "Days Sales Outstanding (DSO)",
    category: "SUPPLIER",
    currentValue: "4.2 Days",
    targetValue: "<= 7.0 Days",
    trend: "DOWN",
    trendValue: "-0.8 Days",
    status: "OPTIMAL",
    benchmarkDesc: "Average settlement turn from doorstep EPoD to supplier bank remittance.",
  },

  // Warehouse Metrics
  {
    name: "Pick Wave Staged Cycle Time",
    category: "WAREHOUSE",
    currentValue: "18.4 Mins",
    targetValue: "<= 25.0 Mins",
    trend: "DOWN",
    trendValue: "-3.2 Mins",
    status: "OPTIMAL",
    benchmarkDesc: "Average time from wave generation to last tote arrival at marshaling dock.",
  },
  {
    name: "Abandoned Tote Recovery Rate",
    category: "WAREHOUSE",
    currentValue: "100.0%",
    targetValue: "100.0%",
    trend: "NEUTRAL",
    trendValue: "0.0%",
    status: "OPTIMAL",
    benchmarkDesc: "Guard #10: Seamless relief picker handover preserving FEFO lot allocations after 20m stall.",
  },
  {
    name: "3D Pallet Storage Density",
    category: "WAREHOUSE",
    currentValue: "88.6%",
    targetValue: ">= 85.0%",
    trend: "UP",
    trendValue: "+2.4%",
    status: "OPTIMAL",
    benchmarkDesc: "Volume cube utilization without crushability weight-stacking violations.",
  },
  {
    name: "Depot Safe Cash Turn-In Variance",
    category: "WAREHOUSE",
    currentValue: "0.02%",
    targetValue: "<= 0.05%",
    trend: "DOWN",
    trendValue: "-0.01%",
    status: "OPTIMAL",
    benchmarkDesc: "Guard #11: Blind recount ratchet discrepancy between driver collections and vault deposits.",
  },

  // Fleet & Logistics Metrics
  {
    name: "DVIR Pre-Trip Compliance",
    category: "FLEET",
    currentValue: "99.2%",
    targetValue: ">= 99.0%",
    trend: "UP",
    trendValue: "+0.4%",
    status: "OPTIMAL",
    benchmarkDesc: "Mandatory 8-point walkaround submission prior to route dispatch authorization.",
  },
  {
    name: "Reefer Thermal Excursion Rate",
    category: "FLEET",
    currentValue: "0.08%",
    targetValue: "<= 0.10%",
    trend: "DOWN",
    trendValue: "-0.04%",
    status: "OPTIMAL",
    benchmarkDesc: "Guard #4: Cold-chain temperature breaches exceeding 30 minutes in transit.",
  },
  {
    name: "Operating Cost Per Ton/Km",
    category: "FLEET",
    currentValue: "3,420 UZS",
    targetValue: "<= 3,800 UZS",
    trend: "DOWN",
    trendValue: "-120 UZS",
    status: "OPTIMAL",
    benchmarkDesc: "Blended fuel, vehicle amortization, and driver compensation per payload ton-kilometer.",
  },
  {
    name: "Mean Time to Mid-Route Rescue",
    category: "FLEET",
    currentValue: "34 Mins",
    targetValue: "<= 45 Mins",
    trend: "DOWN",
    trendValue: "-6 Mins",
    status: "OPTIMAL",
    benchmarkDesc: "Time from vehicle breakdown alert to rescue trailer bolt-seal transshipment.",
  },

  // Retailer Metrics
  {
    name: "Doorstep First-Pass Acceptance",
    category: "RETAILER",
    currentValue: "96.5%",
    targetValue: ">= 95.0%",
    trend: "UP",
    trendValue: "+1.2%",
    status: "OPTIMAL",
    benchmarkDesc: "Orders delivered without partial offload refusal (damaged in transit or short-shipped).",
  },
  {
    name: "Card vs. Cash Settlement Ratio",
    category: "RETAILER",
    currentValue: "62.4% Card",
    targetValue: ">= 60.0%",
    trend: "UP",
    trendValue: "+4.1%",
    status: "OPTIMAL",
    benchmarkDesc: "Electronic card payments (Uzcard/Humo) mitigating Central Bank AML cash risk.",
  },
  {
    name: "Returnable RTI Crate Turnaround",
    category: "RETAILER",
    currentValue: "2.4 Days",
    targetValue: "<= 3.0 Days",
    trend: "DOWN",
    trendValue: "-0.3 Days",
    status: "OPTIMAL",
    benchmarkDesc: "Average dwell time of empty delivery crates at storefront before reverse logistics pickup.",
  },
  {
    name: "Weekly Order Velocity",
    category: "RETAILER",
    currentValue: "3.8 Orders",
    targetValue: ">= 3.5 Orders",
    trend: "UP",
    trendValue: "+0.3 Orders",
    status: "OPTIMAL",
    benchmarkDesc: "Average weekly replenishment frequency per active retailer account.",
  },
];

export default function AnalyticsScorecardPage() {
  const [selectedCategory, setSelectedCategory] = useState<RoleCategory>("ALL");

  const filteredMetrics = SCORECARD_METRICS.filter(
    (m) => selectedCategory === "ALL" || m.category === selectedCategory
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Multi-Tier Ecosystem Scorecards &amp; Performance Radar
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Enterprise benchmarks across Suppliers, Warehouses, Logistics Carriers, and Retailers inspired by o9 Solutions Digital Brain.
          </p>
        </div>

        {/* Category Filter */}
        <div className="flex rounded-lg border bg-gray-100 p-0.5 text-xs font-semibold">
          {(["ALL", "SUPPLIER", "WAREHOUSE", "FLEET", "RETAILER"] as RoleCategory[]).map((cat) => (
            <button
              key={cat}
              type="button"
              onClick={() => setSelectedCategory(cat)}
              className={`rounded-md px-3 py-1 transition-colors ${
                selectedCategory === cat
                  ? "bg-white text-gray-900 shadow-sm"
                  : "text-gray-600 hover:text-gray-900"
              }`}
            >
              {cat === "ALL" ? "All Tiers" : cat.charAt(0) + cat.slice(1).toLowerCase()}
            </button>
          ))}
        </div>
      </div>

      {/* Grid of Scorecard Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {filteredMetrics.map((metric) => (
          <div
            key={metric.name}
            className="rounded-xl border bg-white p-5 shadow-sm space-y-3 flex flex-col justify-between"
          >
            <div>
              <div className="flex items-center justify-between">
                <span className="rounded bg-indigo-50 px-2 py-0.5 text-[10px] font-bold text-indigo-700">
                  {metric.category}
                </span>
                <span className="text-[10px] font-bold text-emerald-700 bg-emerald-50 px-1.5 py-0.5 rounded">
                  {metric.status}
                </span>
              </div>

              <h3 className="text-xs font-bold text-gray-800 mt-2">
                {metric.name}
              </h3>

              <div className="mt-2 flex items-baseline justify-between">
                <div className="text-2xl font-extrabold text-gray-900">
                  {metric.currentValue}
                </div>
                <div className="text-xs font-bold text-emerald-600 flex items-center gap-0.5">
                  {metric.trend === "UP" ? "↑" : metric.trend === "DOWN" ? "↓" : "•"} {metric.trendValue}
                </div>
              </div>

              <div className="text-[11px] text-gray-400 mt-0.5">
                Target: <span className="font-semibold text-gray-600">{metric.targetValue}</span>
              </div>
            </div>

            <div className="pt-2 border-t border-gray-100">
              <p className="text-[11px] text-gray-500 line-clamp-2" title={metric.benchmarkDesc}>
                {metric.benchmarkDesc}
              </p>
            </div>
          </div>
        ))}
      </div>

      {/* Enterprise Grade Comparison & SLA Matrix */}
      <div className="rounded-xl border bg-white p-6 shadow-sm space-y-4">
        <div className="border-b pb-3">
          <h2 className="text-sm font-bold text-gray-900">
            Enterprise Tier Comparison &amp; SLA Adherence Matrix
          </h2>
          <p className="text-xs text-gray-500">
            Real-time rating of ecosystem participants against contract SLAs and regulatory standards.
          </p>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left text-xs">
            <thead className="border-b bg-gray-50 uppercase tracking-wider text-gray-500 font-bold">
              <tr>
                <th className="px-4 py-2.5">Participant Entity</th>
                <th className="px-4 py-2.5">Role Domain</th>
                <th className="px-4 py-2.5">Fulfillment Score</th>
                <th className="px-4 py-2.5">Compliance Grade</th>
                <th className="px-4 py-2.5">SLA Rating</th>
                <th className="px-4 py-2.5 text-right">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              <tr className="hover:bg-gray-50">
                <td className="px-4 py-3 font-semibold text-gray-900">
                  Tashkent FMCG Distribution LLC (<CopyableId id="sup_tashkent_fmcg" label="Supplier" />)
                </td>
                <td className="px-4 py-3 text-gray-600 font-mono">SUPPLIER</td>
                <td className="px-4 py-3 font-bold text-emerald-700">98.4% OTIF</td>
                <td className="px-4 py-3 font-semibold text-gray-800">100% Soliq Fiscal</td>
                <td className="px-4 py-3 font-extrabold text-indigo-700">TIER-1 (A+)</td>
                <td className="px-4 py-3 text-right">
                  <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-bold text-emerald-800">
                    PREFERRED
                  </span>
                </td>
              </tr>

              <tr className="hover:bg-gray-50">
                <td className="px-4 py-3 font-semibold text-gray-900">
                  Tashkent Central Super-Depot (<CopyableId id="depot_tashkent_central" label="Depot" />)
                </td>
                <td className="px-4 py-3 text-gray-600 font-mono">WAREHOUSE</td>
                <td className="px-4 py-3 font-bold text-emerald-700">18.4m Wave Cycle</td>
                <td className="px-4 py-3 font-semibold text-gray-800">0.02% Shortage Variance</td>
                <td className="px-4 py-3 font-extrabold text-indigo-700">TIER-1 (A+)</td>
                <td className="px-4 py-3 text-right">
                  <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-bold text-emerald-800">
                    OPTIMAL
                  </span>
                </td>
              </tr>

              <tr className="hover:bg-gray-50">
                <td className="px-4 py-3 font-semibold text-gray-900">
                  Isuzu Reefer Fleet #01 (<CopyableId id="veh_isuzu_01" label="Fleet" />)
                </td>
                <td className="px-4 py-3 text-gray-600 font-mono">FLEET</td>
                <td className="px-4 py-3 font-bold text-amber-700">RED-TAG IN MAINTENANCE</td>
                <td className="px-4 py-3 font-semibold text-rose-700">Failed Brakes Walkaround</td>
                <td className="px-4 py-3 font-extrabold text-rose-700">GROUNDED (C)</td>
                <td className="px-4 py-3 text-right">
                  <span className="rounded-full bg-rose-100 px-2 py-0.5 text-[10px] font-bold text-rose-800">
                    BAY SERVICE
                  </span>
                </td>
              </tr>

              <tr className="hover:bg-gray-50">
                <td className="px-4 py-3 font-semibold text-gray-900">
                  Korzinka Supermarket #01 (<CopyableId id="ret_korzinka_01" label="Retailer" />)
                </td>
                <td className="px-4 py-3 text-gray-600 font-mono">RETAILER</td>
                <td className="px-4 py-3 font-bold text-emerald-700">97.8% Doorstep Acceptance</td>
                <td className="px-4 py-3 font-semibold text-gray-800">Uncapped Multi-Tender (14.5M Daily)</td>
                <td className="px-4 py-3 font-extrabold text-indigo-700">TIER-1 (A+)</td>
                <td className="px-4 py-3 text-right">
                  <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-bold text-emerald-800">
                    EXCELLENT
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
