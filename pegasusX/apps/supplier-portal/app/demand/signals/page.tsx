"use client";

import React, { useState, useEffect } from "react";


type SignalType = "PROMO" | "EVENT" | "PAYDAY" | "EVENT_DENSITY" | "COMPETITOR_PRESSURE";

interface DemandSignal {
  signalId: string;
  type: SignalType;
  scope: string;
  sku?: string | null;
  startAt: string;
  endAt: string;
  multiplier: number;
  meta?: {
    title?: string;
    description?: string;
    campaignId?: string;
  };
}

export default function SignalsPage() {
  const [signals, setSignals] = useState<DemandSignal[]>([]);
  const [loading, setLoading] = useState(true);
  
  // Filters
  const [filterType, setFilterType] = useState<string>("");
  const [filterActive, setFilterActive] = useState<boolean>(true);
  
  // Modal state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingSignal, setEditingSignal] = useState<DemandSignal | null>(null);

  // Form state
  const [form, setForm] = useState({
    type: "PROMO",
    scope: "",
    sku: "",
    startAt: "",
    endAt: "",
    multiplier: 1.0,
    title: "",
    description: ""
  });

  useEffect(() => {
    fetchSignals();
  }, [filterType, filterActive]);

  const fetchSignals = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (filterType) params.append("type", filterType);
      if (filterActive) params.append("active", "true");
      
      const res = await fetch(`/api/demand/signals?${params.toString()}`);
      if (!res.ok) throw new Error("Failed to load signals");
      const data = await res.json();
      setSignals(data || []);
    } catch (err: any) {
      console.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleOpenCreate = () => {
    setEditingSignal(null);
    setForm({
      type: "PROMO",
      scope: "",
      sku: "",
      startAt: "",
      endAt: "",
      multiplier: 1.0,
      title: "",
      description: ""
    });
    setIsModalOpen(true);
  };

  const handleOpenEdit = (sig: DemandSignal) => {
    setEditingSignal(sig);
    setForm({
      type: sig.type,
      scope: sig.scope,
      sku: sig.sku || "",
      startAt: sig.startAt.slice(0, 16), // datetime-local format
      endAt: sig.endAt.slice(0, 16),
      multiplier: sig.multiplier,
      title: sig.meta?.title || "",
      description: sig.meta?.description || ""
    });
    setIsModalOpen(true);
  };

  const handleSave = async () => {
    if (form.multiplier < 0.5 || form.multiplier > 2.5) {
      console.error("Multiplier must be between 0.5 and 2.5");
      return;
    }
    if (new Date(form.startAt) >= new Date(form.endAt)) {
      console.error("Start time must be before End time");
      return;
    }
    if (!form.scope) {
      console.error("Scope is required");
      return;
    }

    const payload = {
      type: form.type,
      scope: form.scope,
      sku: form.sku || null,
      startAt: new Date(form.startAt).toISOString(),
      endAt: new Date(form.endAt).toISOString(),
      multiplier: form.multiplier,
      meta: {
        title: form.title,
        description: form.description
      }
    };

    try {
      let res;
      if (editingSignal) {
        res = await fetch(`/api/demand/signals/${editingSignal.signalId}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        });
      } else {
        res = await fetch("/api/demand/signals", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload)
        });
      }

      if (!res.ok) {
        const d = await res.json();
        throw new Error(d.error || "Failed to save signal");
      }
      
      console.log(editingSignal ? "Signal updated" : "Signal created");
      setIsModalOpen(false);
      fetchSignals();
    } catch (err: any) {
      console.error(err.message);
    }
  };

  const handleDeactivate = async (id: string) => {
    if (!confirm("Are you sure you want to deactivate this signal?")) return;
    try {
      const res = await fetch(`/api/demand/signals/${id}/deactivate`, {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to deactivate");
      console.log("Signal created successfully");
      fetchSignals();
    } catch (err: any) {
      console.error(err.message);
    }
  };

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Demand Signals</h1>
          <p className="text-gray-500 mt-2">Manage promo and event multipliers for causal demand sensing.</p>
        </div>
        <button 
          onClick={handleOpenCreate}
          className="bg-black hover:bg-gray-800 text-white px-4 py-2 rounded-lg font-medium transition-colors"
        >
          Create Signal
        </button>
      </div>

      <div className="bg-white p-4 rounded-xl border border-gray-200 flex gap-4 items-center">
        <select 
          className="border rounded-md px-3 py-2 text-sm"
          value={filterType} 
          onChange={(e) => setFilterType(e.target.value)}
        >
          <option value="">All Types</option>
          <option value="PROMO">Promo</option>
          <option value="EVENT">Event</option>
          <option value="PAYDAY">Payday</option>
          <option value="EVENT_DENSITY">Event Density</option>
          <option value="COMPETITOR_PRESSURE">Competitor Pressure</option>
        </select>

        <label className="flex items-center gap-2 text-sm">
          <input 
            type="checkbox" 
            checked={filterActive} 
            onChange={(e) => setFilterActive(e.target.checked)}
            className="rounded text-black focus:ring-black"
          />
          Active Only
        </label>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-gray-50 text-gray-600 font-medium border-b border-gray-200">
              <tr>
                <th className="px-6 py-4">Title / Type</th>
                <th className="px-6 py-4">Scope</th>
                <th className="px-6 py-4">SKU</th>
                <th className="px-6 py-4">Multiplier</th>
                <th className="px-6 py-4">Validity</th>
                <th className="px-6 py-4">Status</th>
                <th className="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {loading ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-gray-500">Loading signals...</td></tr>
              ) : signals.length === 0 ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-gray-500">No signals found.</td></tr>
              ) : signals.map(sig => {
                const isActive = new Date() >= new Date(sig.startAt) && new Date() <= new Date(sig.endAt);
                const isExpired = new Date() > new Date(sig.endAt);
                
                return (
                  <tr key={sig.signalId} className="hover:bg-gray-50/50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="font-medium text-gray-900">{sig.meta?.title || 'Untitled'}</div>
                      <div className="text-xs text-gray-500 font-mono mt-1">{sig.type}</div>
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">{sig.scope}</td>
                    <td className="px-6 py-4">{sig.sku || <span className="text-gray-400 italic">All</span>}</td>
                    <td className="px-6 py-4">
                      <span className={`inline-flex items-center px-2 py-1 rounded text-xs font-semibold ${sig.multiplier > 1 ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                        {sig.multiplier.toFixed(2)}x
                      </span>
                    </td>
                    <td className="px-6 py-4 text-xs text-gray-600">
                      <div>{new Date(sig.startAt).toLocaleString()}</div>
                      <div className="text-gray-400">to</div>
                      <div>{new Date(sig.endAt).toLocaleString()}</div>
                    </td>
                    <td className="px-6 py-4">
                      {isActive ? (
                        <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-blue-50 text-blue-700 text-xs font-medium">
                          <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></span> Active
                        </span>
                      ) : isExpired ? (
                        <span className="inline-flex items-center px-2 py-1 rounded-full bg-gray-100 text-gray-600 text-xs font-medium">Expired</span>
                      ) : (
                        <span className="inline-flex items-center px-2 py-1 rounded-full bg-yellow-50 text-yellow-700 text-xs font-medium">Scheduled</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-right space-x-3">
                      <button onClick={() => handleOpenEdit(sig)} className="text-blue-600 hover:text-blue-800 font-medium text-xs">Edit</button>
                      <button onClick={() => handleDeactivate(sig.signalId)} className="text-red-600 hover:text-red-800 font-medium text-xs">Deactivate</button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-lg overflow-hidden flex flex-col max-h-[90vh]">
            <div className="px-6 py-4 border-b border-gray-100 flex justify-between items-center">
              <h2 className="text-lg font-semibold">{editingSignal ? "Edit Signal" : "Create Signal"}</h2>
              <button onClick={() => setIsModalOpen(false)} className="text-gray-400 hover:text-gray-600">&times;</button>
            </div>
            
            <div className="p-6 overflow-y-auto space-y-4">
              <div className="bg-blue-50 text-blue-800 text-sm p-3 rounded-lg flex items-start gap-2">
                <span className="text-xl">ℹ️</span>
                <p>This multiplies expected demand by your chosen factor for the selected scope.</p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-xs font-medium text-gray-700">Type</label>
                  <select 
                    value={form.type} 
                    onChange={e => setForm({...form, type: e.target.value})}
                    className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none"
                  >
                    <option value="PROMO">Promo</option>
                    <option value="EVENT">Event</option>
                    <option value="PAYDAY">Payday</option>
                    <option value="EVENT_DENSITY">Event Density</option>
                    <option value="COMPETITOR_PRESSURE">Competitor Pressure</option>
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-gray-700">Scope</label>
                  <input 
                    type="text" 
                    value={form.scope}
                    onChange={e => setForm({...form, scope: e.target.value})}
                    placeholder="e.g. retailer:123 or city:Tashkent"
                    className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none"
                  />
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-xs font-medium text-gray-700">SKU (Optional)</label>
                <input 
                  type="text" 
                  value={form.sku}
                  onChange={e => setForm({...form, sku: e.target.value})}
                  placeholder="Leave blank for all SKUs"
                  className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-xs font-medium text-gray-700">Start Time</label>
                  <input 
                    type="datetime-local" 
                    value={form.startAt}
                    onChange={e => setForm({...form, startAt: e.target.value})}
                    className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-gray-700">End Time</label>
                  <input 
                    type="datetime-local" 
                    value={form.endAt}
                    onChange={e => setForm({...form, endAt: e.target.value})}
                    className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none"
                  />
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-xs font-medium text-gray-700">Multiplier ({form.multiplier.toFixed(2)}x)</label>
                <input 
                  type="range" 
                  min="0.5" 
                  max="2.5" 
                  step="0.05"
                  value={form.multiplier}
                  onChange={e => setForm({...form, multiplier: parseFloat(e.target.value)})}
                  className="w-full"
                />
                <div className="flex justify-between text-xs text-gray-500">
                  <span>0.5x (Half demand)</span>
                  <span>1.0x (Normal)</span>
                  <span>2.5x (Peak)</span>
                </div>
              </div>

              <div className="space-y-1 pt-2 border-t">
                <label className="text-xs font-medium text-gray-700">Title</label>
                <input 
                  type="text" 
                  value={form.title}
                  onChange={e => setForm({...form, title: e.target.value})}
                  placeholder="e.g. Summer soft-drinks push"
                  className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium text-gray-700">Description (Optional)</label>
                <textarea 
                  value={form.description}
                  onChange={e => setForm({...form, description: e.target.value})}
                  rows={2}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:ring-1 focus:ring-black outline-none resize-none"
                />
              </div>

            </div>
            
            <div className="px-6 py-4 border-t border-gray-100 bg-gray-50 flex justify-end gap-3">
              <button 
                onClick={() => setIsModalOpen(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 hover:text-gray-900"
              >
                Cancel
              </button>
              <button 
                onClick={handleSave}
                className="px-4 py-2 text-sm font-medium text-white bg-black hover:bg-gray-800 rounded-lg shadow-sm"
              >
                {editingSignal ? "Save Changes" : "Create Signal"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
