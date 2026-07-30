"use client";

import React, { useState, useEffect } from "react";


interface DemandSignal {
  signalId: string;
  type: string;
  scope: string;
  startAt: string;
  endAt: string;
  multiplier: number;
  meta?: {
    title?: string;
    description?: string;
  };
}

export default function PaydayCalendarPage() {
  const [signals, setSignals] = useState<DemandSignal[]>([]);
  const [loading, setLoading] = useState(true);

  // Form state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [form, setForm] = useState({
    title: "",
    startAt: "",
    endAt: "",
    multiplier: 1.5, // Default for payday
  });

  useEffect(() => {
    fetchPaydays();
  }, []);

  const fetchPaydays = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/demand/signals?type=PAYDAY`);
      if (!res.ok) throw new Error("Failed to load payday signals");
      const data = await res.json();
      setSignals(data || []);
    } catch (err: any) {
      console.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (new Date(form.startAt) >= new Date(form.endAt)) {
      console.error("Start time must be before End time");
      return;
    }

    const payload = {
      type: "PAYDAY",
      scope: "country:UZ", // Paydays are generally country-wide
      startAt: new Date(form.startAt).toISOString(),
      endAt: new Date(form.endAt).toISOString(),
      multiplier: form.multiplier,
      meta: {
        title: form.title || "Payday Spike",
        description: "Monthly salary payout"
      }
    };

    try {
      const res = await fetch("/api/demand/signals", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });

      if (!res.ok) {
        const d = await res.json();
        throw new Error(d.error || "Failed to create payday signal");
      }
      
      console.log("Payday window created successfully!");
      setIsModalOpen(false);
      fetchPaydays();
    } catch (err: any) {
      console.error(err.message);
    }
  };

  const handleDeactivate = async (id: string) => {
    if (!confirm("Are you sure you want to remove this payday window?")) return;
    try {
      const res = await fetch(`/api/demand/signals/${id}/deactivate`, {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to remove");
      console.log("Payday window removed");
      fetchPaydays();
    } catch (err: any) {
      console.error(err.message);
    }
  };

  return (
    <div className="p-8 max-w-5xl mx-auto space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Payday Calendar</h1>
          <p className="text-gray-500 mt-2">Manage predictable demand spikes across the region.</p>
        </div>
        <button 
          onClick={() => setIsModalOpen(true)}
          className="bg-black hover:bg-gray-800 text-white px-4 py-2 rounded-lg font-medium transition-colors"
        >
          Add Payday Window
        </button>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden shadow-sm">
        <table className="w-full text-sm text-left">
          <thead className="bg-gray-50 text-gray-600 font-medium border-b border-gray-200">
            <tr>
              <th className="px-6 py-4">Title</th>
              <th className="px-6 py-4">Scope</th>
              <th className="px-6 py-4">Dates</th>
              <th className="px-6 py-4">Multiplier</th>
              <th className="px-6 py-4 text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {loading ? (
              <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-500">Loading calendar...</td></tr>
            ) : signals.length === 0 ? (
              <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-500">No paydays scheduled.</td></tr>
            ) : signals.map(sig => (
              <tr key={sig.signalId} className="hover:bg-gray-50/50">
                <td className="px-6 py-4 font-medium text-gray-900">{sig.meta?.title || 'Payday'}</td>
                <td className="px-6 py-4 font-mono text-xs text-gray-500">{sig.scope}</td>
                <td className="px-6 py-4 text-gray-600">
                  {new Date(sig.startAt).toLocaleDateString()} &rarr; {new Date(sig.endAt).toLocaleDateString()}
                </td>
                <td className="px-6 py-4">
                  <span className="bg-green-100 text-green-700 px-2 py-1 rounded text-xs font-bold">
                    {sig.multiplier}x
                  </span>
                </td>
                <td className="px-6 py-4 text-right">
                  <button onClick={() => handleDeactivate(sig.signalId)} className="text-red-600 hover:text-red-800 font-medium text-xs">Remove</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="bg-white rounded-2xl shadow-xl w-full max-w-md overflow-hidden flex flex-col">
            <div className="px-6 py-4 border-b border-gray-100 flex justify-between items-center">
              <h2 className="text-lg font-semibold">Schedule Payday Spike</h2>
              <button onClick={() => setIsModalOpen(false)} className="text-gray-400 hover:text-gray-600">&times;</button>
            </div>
            
            <div className="p-6 space-y-4">
              <div className="space-y-1">
                <label className="text-xs font-medium text-gray-700">Title</label>
                <input 
                  type="text" 
                  value={form.title}
                  onChange={e => setForm({...form, title: e.target.value})}
                  className="w-full border rounded-md px-3 py-2 text-sm"
                  placeholder="e.g. August State Pension Payout"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-xs font-medium text-gray-700">Start Time</label>
                  <input 
                    type="datetime-local" 
                    value={form.startAt}
                    onChange={e => setForm({...form, startAt: e.target.value})}
                    className="w-full border rounded-md px-3 py-2 text-sm"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-gray-700">End Time</label>
                  <input 
                    type="datetime-local" 
                    value={form.endAt}
                    onChange={e => setForm({...form, endAt: e.target.value})}
                    className="w-full border rounded-md px-3 py-2 text-sm"
                  />
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-xs font-medium text-gray-700">Demand Multiplier (Expected Lift)</label>
                <input 
                  type="number" 
                  step="0.1"
                  min="1.0"
                  max="3.0"
                  value={form.multiplier}
                  onChange={e => setForm({...form, multiplier: parseFloat(e.target.value)})}
                  className="w-full border rounded-md px-3 py-2 text-sm"
                />
              </div>
            </div>
            
            <div className="px-6 py-4 bg-gray-50 border-t border-gray-100 flex justify-end gap-3">
              <button 
                onClick={() => setIsModalOpen(false)}
                className="px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-800"
              >
                Cancel
              </button>
              <button 
                onClick={handleSave}
                className="px-4 py-2 text-sm font-medium bg-black text-white rounded-lg hover:bg-gray-800"
              >
                Schedule
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
