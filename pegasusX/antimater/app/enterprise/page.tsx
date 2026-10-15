'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import ChamferButton from '@/app/components/ChamferButton';
import { useLanguage } from '@/app/context/LanguageContext';
import type { EnterpriseRequest, RequestStatus } from '@/app/lib/supabase/types';

export default function EnterprisePage() {
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState<'request' | 'console'>('request');

  // Form State
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    company: '',
    role: 'Supplier Operations',
    fleet_size: '4–10 sites',
    inquiry_type: 'demo' as const,
    subject: '',
    message: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitStatus, setSubmitStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [submitError, setSubmitError] = useState('');

  // Console State
  const [requests, setRequests] = useState<EnterpriseRequest[]>([]);
  const [selectedReq, setSelectedReq] = useState<EnterpriseRequest | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [isLoadingConsole, setIsLoadingConsole] = useState(false);

  // Newsletter State
  const [subEmail, setSubEmail] = useState('');
  const [subStatus, setSubStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');

  useEffect(() => {
    document.title = 'Enterprise Logistics OS | Pegasus';
  }, []);

  const fetchConsoleData = async () => {
    setIsLoadingConsole(true);
    try {
      const url =
        statusFilter === 'all'
          ? '/api/admin/requests'
          : `/api/admin/requests?status=${statusFilter}`;
      const res = await fetch(url);
      if (res.ok) {
        const json = await res.json();
        if (json.data) {
          setRequests(json.data);
        }
      }
    } catch (err) {
      console.warn('[Enterprise] Failed to fetch live console data:', err);
    } finally {
      setIsLoadingConsole(false);
    }
  };

  useEffect(() => {
    if (activeTab === 'console') {
      fetchConsoleData();
      const interval = setInterval(fetchConsoleData, 8000);
      return () => clearInterval(interval);
    }
  }, [activeTab, statusFilter]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name.trim() || !formData.email.trim()) {
      setSubmitStatus('error');
      setSubmitError('Name and Corporate Email are required.');
      return;
    }

    setIsSubmitting(true);
    setSubmitStatus('idle');
    setSubmitError('');

    try {
      const res = await fetch('/api/enterprise/request', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...formData,
          source_page: '/enterprise',
        }),
      });

      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Submission failed');

      setSubmitStatus('success');
      setFormData({
        name: '',
        email: '',
        company: '',
        role: 'Supplier Operations',
        fleet_size: '4–10 sites',
        inquiry_type: 'demo',
        subject: '',
        message: '',
      });
    } catch (err) {
      setSubmitStatus('error');
      setSubmitError(err instanceof Error ? err.message : 'Error submitting request');
    } finally {
      setIsSubmitting(false);
    }
  };

  const updateRequestStatus = async (id: string, newStatus: RequestStatus) => {
    try {
      const res = await fetch('/api/admin/requests', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, status: newStatus }),
      });
      if (res.ok) {
        setRequests((prev) =>
          prev.map((r) => (r.id === id ? { ...r, status: newStatus } : r))
        );
        if (selectedReq && selectedReq.id === id) {
          setSelectedReq((prev) => (prev ? { ...prev, status: newStatus } : null));
        }
      }
    } catch (err) {
      console.error('[Enterprise] Failed to update status:', err);
    }
  };

  const handleSubscribe = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!subEmail.trim()) return;
    setSubStatus('loading');
    try {
      const res = await fetch('/api/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: subEmail.trim(), source: 'enterprise_portal' }),
      });
      if (res.ok) {
        setSubStatus('success');
        setSubEmail('');
        setTimeout(() => setSubStatus('idle'), 4000);
      } else {
        setSubStatus('error');
      }
    } catch {
      setSubStatus('error');
    }
  };

  const filteredRequests = requests.filter((r) => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return (
      r.name.toLowerCase().includes(q) ||
      r.email.toLowerCase().includes(q) ||
      (r.company && r.company.toLowerCase().includes(q)) ||
      (r.role && r.role.toLowerCase().includes(q))
    );
  });

  return (
    <div className="min-h-screen bg-[#000000] text-white selection:bg-[#A9EBF9] selection:text-black font-sans">
      {/* TOP CAD NAVIGATION BAR */}
      <header className="border-b border-white/10 bg-[#080808]/90 backdrop-blur sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-3 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Link
              href="/"
              className="text-xs font-mono tracking-widest text-[#A9EBF9] hover:text-white uppercase flex items-center gap-2.5"
            >
              <div className="w-7 h-7 border border-white/20 bg-[#0d0d0d] flex items-center justify-center p-1">
                <Image
                  src="/pegasus-nav.png"
                  alt="Pegasus Emblem"
                  width={20}
                  height={20}
                  className="w-full h-full object-contain"
                />
              </div>
              <span>PEGASUS // ARCHITECTURE</span>
            </Link>
            <span className="text-white/20">|</span>
            <span className="font-mono text-[11px] uppercase tracking-wider text-white/50">
              ENTERPRISE PLATFORM HUB
            </span>
          </div>

          <div className="flex items-center gap-3">
            <div className="flex border border-white/20 bg-black p-0.5">
              <button
                onClick={() => setActiveTab('request')}
                className={`px-3 py-1 text-xs font-mono uppercase tracking-wider transition-colors ${
                  activeTab === 'request'
                    ? 'bg-white text-black font-semibold'
                    : 'text-white/60 hover:text-white'
                }`}
              >
                REQUEST ACCESS
              </button>
              <button
                onClick={() => setActiveTab('console')}
                className={`px-3 py-1 text-xs font-mono uppercase tracking-wider transition-colors ${
                  activeTab === 'console'
                    ? 'bg-white text-black font-semibold'
                    : 'text-white/60 hover:text-white'
                }`}
              >
                LIVE CONSOLE
              </button>
            </div>
            <Link
              href="/admin"
              className="border border-white/20 px-3 py-1 text-xs font-mono uppercase tracking-wider text-white/60 hover:border-white hover:text-white transition-colors"
            >
              ADMIN PORTAL →
            </Link>
          </div>
        </div>
      </header>

      {/* HERO BANNER */}
      <section className="border-b border-white/10 bg-[radial-gradient(ellipse_at_top,rgba(169,235,249,0.06)_0%,transparent_60%)] py-12 px-4 sm:px-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-wrap items-center justify-between gap-6">
            <div>
              <div className="flex items-center gap-2 mb-3">
                <span className="h-2 w-2 rounded-full bg-[#8DDC96] animate-pulse" />
                <span className="font-mono text-xs uppercase tracking-widest text-[#8DDC96]">
                  SUPABASE CLOUD SYNC ACTIVE
                </span>
                <span className="text-white/20">·</span>
                <span className="font-mono text-xs uppercase tracking-widest text-white/40">
                  POSTGRES WITH RLS & AUDIT LOGS
                </span>
              </div>
              <h1 className="text-3xl sm:text-5xl font-light tracking-tight text-white">
                Enterprise Logistics Deployment
              </h1>
              <p className="mt-2 text-sm sm:text-base text-white/60 max-w-2xl">
                Onboard supplier-led fulfillment networks, request pilot sandboxes, and inspect live
                operational requests across 6 integrated platform roles.
              </p>
            </div>

            <div className="border border-white/15 bg-black/80 p-4 font-mono text-xs space-y-1.5 min-w-[240px]">
              <div className="text-white/40 uppercase tracking-widest text-[10px]">
                PLATFORM CAPABILITIES
              </div>
              <div className="flex justify-between text-white/80">
                <span>Multi-Site Dispatch:</span>
                <span className="text-[#A9EBF9]">ENABLED</span>
              </div>
              <div className="flex justify-between text-white/80">
                <span>Native Clients:</span>
                <span className="text-[#FBFF63]">SWIFT + KOTLIN</span>
              </div>
              <div className="flex justify-between text-white/80">
                <span>Response SLA:</span>
                <span className="text-[#8DDC96]">&lt; 24 HOURS</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MAIN CONTENT AREA */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-10">
        {activeTab === 'request' ? (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
            {/* Left 7 Cols: Form */}
            <div className="lg:col-span-7 border border-white/15 bg-[#0A0A0A] p-6 sm:p-8 relative">
              <div className="border-b border-white/10 pb-4 mb-6">
                <span className="font-mono text-[10px] uppercase tracking-widest text-[#A9EBF9]">
                  STEP 01 // SPECIFY REQUIREMENTS
                </span>
                <h2 className="text-2xl font-light text-white mt-1">
                  Request Enterprise Sandbox or Live Demo
                </h2>
              </div>

              <form onSubmit={handleSubmit} className="space-y-5" noValidate>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                      Contact Name *
                    </label>
                    <input
                      type="text"
                      required
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      placeholder="e.g. Elena Rostova"
                      disabled={isSubmitting}
                      className="w-full bg-black border border-white/20 px-3.5 py-2.5 text-sm text-white placeholder-white/25 focus:border-[#A9EBF9] focus:outline-none transition-colors"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                      Corporate Email *
                    </label>
                    <input
                      type="email"
                      required
                      value={formData.email}
                      onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                      placeholder="operations@enterprise.com"
                      disabled={isSubmitting}
                      className="w-full bg-black border border-white/20 px-3.5 py-2.5 text-sm text-white placeholder-white/25 focus:border-[#A9EBF9] focus:outline-none transition-colors"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                      Organization / Company
                    </label>
                    <input
                      type="text"
                      value={formData.company}
                      onChange={(e) => setFormData({ ...formData, company: e.target.value })}
                      placeholder="e.g. SilkRoad Freight LLC"
                      disabled={isSubmitting}
                      className="w-full bg-black border border-white/20 px-3.5 py-2.5 text-sm text-white placeholder-white/25 focus:border-[#A9EBF9] focus:outline-none transition-colors"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                      Operational Role
                    </label>
                    <select
                      value={formData.role}
                      onChange={(e) => setFormData({ ...formData, role: e.target.value })}
                      disabled={isSubmitting}
                      className="w-full bg-black border border-white/20 px-3.5 py-2.5 text-sm text-white focus:border-[#A9EBF9] focus:outline-none transition-colors"
                    >
                      <option value="Supplier Operations">Supplier Operations</option>
                      <option value="Warehouse Manager">Warehouse Manager</option>
                      <option value="Fleet / Dispatch Lead">Fleet / Dispatch Lead</option>
                      <option value="IT / Platform Owner">IT / Platform Owner</option>
                      <option value="Executive / Founder">Executive / Founder</option>
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                      Fleet / Network Scope
                    </label>
                    <select
                      value={formData.fleet_size}
                      onChange={(e) => setFormData({ ...formData, fleet_size: e.target.value })}
                      disabled={isSubmitting}
                      className="w-full bg-black border border-white/20 px-3.5 py-2.5 text-sm text-white focus:border-[#A9EBF9] focus:outline-none transition-colors"
                    >
                      <option value="1–3 sites">1–3 sites (Pilot)</option>
                      <option value="4–10 sites">4–10 sites (Regional)</option>
                      <option value="11–25 sites">11–25 sites (Multi-Hub)</option>
                      <option value="25+ sites">25+ sites (Enterprise)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                      Inquiry Type
                    </label>
                    <select
                      value={formData.inquiry_type}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          inquiry_type: e.target.value as any,
                        })
                      }
                      disabled={isSubmitting}
                      className="w-full bg-black border border-white/20 px-3.5 py-2.5 text-sm text-white focus:border-[#A9EBF9] focus:outline-none transition-colors"
                    >
                      <option value="demo">Interactive Demo</option>
                      <option value="pilot">Sandbox Pilot Program</option>
                      <option value="sales">Commercial Contract</option>
                      <option value="partnership">Ecosystem Partnership</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono uppercase tracking-wider text-white/50 mb-1.5">
                    Operational Requirements / Constraints
                  </label>
                  <textarea
                    rows={4}
                    value={formData.message}
                    onChange={(e) => setFormData({ ...formData, message: e.target.value })}
                    placeholder="Specify number of distribution centers, daily SKU throughput, vehicle fleet specs, or ERP integrations needed..."
                    disabled={isSubmitting}
                    className="w-full bg-black border border-white/20 p-3 text-sm text-white placeholder-white/25 focus:border-[#A9EBF9] focus:outline-none transition-colors"
                  />
                </div>

                {submitStatus === 'success' && (
                  <div className="border border-[#8DDC96] bg-[#8DDC96]/10 p-4 font-mono text-xs text-[#8DDC96]">
                    ✓ Enterprise request stored in Supabase & queued for operational review.
                  </div>
                )}
                {submitStatus === 'error' && (
                  <div className="border border-[#FE5934] bg-[#FE5934]/10 p-4 font-mono text-xs text-[#FE5934]">
                    ✗ {submitError}
                  </div>
                )}

                <ChamferButton
                  type="submit"
                  variant="fill"
                  className="w-full justify-center"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'TRANSMITTING REQUEST...' : 'SUBMIT ENTERPRISE REQUEST →'}
                </ChamferButton>
              </form>
            </div>

            {/* Right 5 Cols: Architecture & App Matrix */}
            <div className="lg:col-span-5 space-y-6">
              <div className="border border-white/15 bg-[#0D0D0D] p-6">
                <span className="font-mono text-[10px] uppercase tracking-widest text-[#FBFF63] block mb-2">
                  CROSS-PLATFORM ARCHITECTURE
                </span>
                <h3 className="text-xl font-light text-white mb-4">
                  One Unified Source of Truth
                </h3>
                <div className="space-y-3 font-mono text-xs">
                  <div className="border border-white/10 p-3 bg-black">
                    <div className="flex justify-between text-white font-semibold mb-1">
                      <span>1. Web Experience</span>
                      <span className="text-[#A9EBF9]">Next.js 16 + React 19</span>
                    </div>
                    <p className="text-white/50 font-sans text-xs">
                      Server-rendered enterprise hub, live telemetry, and zero-latency admin ops.
                    </p>
                  </div>

                  <div className="border border-white/10 p-3 bg-black">
                    <div className="flex justify-between text-white font-semibold mb-1">
                      <span>2. Native Desktop App</span>
                      <span className="text-[#FBFF63]">Swift (macOS SwiftUI)</span>
                    </div>
                    <p className="text-white/50 font-sans text-xs">
                      Native macOS client with sidebar inspection, keyboard workflows, and offline caching.
                    </p>
                  </div>

                  <div className="border border-white/10 p-3 bg-black">
                    <div className="flex justify-between text-white font-semibold mb-1">
                      <span>3. Native Android App</span>
                      <span className="text-[#8DDC96]">Kotlin (Jetpack Compose)</span>
                    </div>
                    <p className="text-white/50 font-sans text-xs">
                      Low-power driver and warehouse handheld client with instant CAD status updates.
                    </p>
                  </div>
                </div>
              </div>

              {/* Newsletter Callout */}
              <div className="border border-white/15 bg-[#0D0D0D] p-6">
                <span className="font-mono text-[10px] uppercase tracking-widest text-white/50 block mb-2">
                  DISPATCH // NEWSLETTER
                </span>
                <p className="text-sm text-white/70 mb-4">
                  Get monthly engineering memos and architectural releases on consistent logistics
                  routing and micro-hub topologies.
                </p>
                <form onSubmit={handleSubscribe} className="space-y-2">
                  <div className="flex gap-2">
                    <input
                      type="email"
                      required
                      value={subEmail}
                      onChange={(e) => setSubEmail(e.target.value)}
                      placeholder="engineer@domain.com"
                      className="flex-1 bg-black border border-white/20 px-3 py-2 text-xs text-white placeholder-white/30 focus:border-[#A9EBF9] focus:outline-none"
                    />
                    <button
                      type="submit"
                      disabled={subStatus === 'loading'}
                      className="border border-white/30 px-3 py-2 font-mono text-xs uppercase tracking-wider text-white hover:bg-white hover:text-black transition-colors"
                    >
                      {subStatus === 'loading' ? '...' : 'JOIN'}
                    </button>
                  </div>
                  {subStatus === 'success' && (
                    <p className="font-mono text-[11px] text-[#8DDC96]">✓ Subscribed to dispatch!</p>
                  )}
                  {subStatus === 'error' && (
                    <p className="font-mono text-[11px] text-[#FE5934]">✗ Subscription failed.</p>
                  )}
                </form>
              </div>
            </div>
          </div>
        ) : (
          /* CONSOLE TAB */
          <div className="border border-white/15 bg-[#0A0A0A] p-6">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 pb-4 mb-6">
              <div>
                <span className="font-mono text-[10px] uppercase tracking-widest text-[#8DDC96]">
                  OPERATIONS CONSOLE // SUPABASE PIPELINE
                </span>
                <h2 className="text-2xl font-light text-white mt-1">Live Enterprise Queue</h2>
              </div>

              <div className="flex flex-wrap gap-2 items-center">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search name, org, role..."
                  className="bg-black border border-white/20 px-3 py-1.5 text-xs text-white placeholder-white/30 focus:border-[#A9EBF9] focus:outline-none font-mono"
                />

                <div className="flex border border-white/20 bg-black">
                  {['all', 'new', 'in_review', 'qualified', 'closed'].map((f) => (
                    <button
                      key={f}
                      onClick={() => setStatusFilter(f)}
                      className={`px-2.5 py-1 text-[11px] font-mono uppercase tracking-wider transition-colors ${
                        statusFilter === f
                          ? 'bg-white text-black font-semibold'
                          : 'text-white/60 hover:text-white'
                      }`}
                    >
                      {f.replace('_', ' ')}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {isLoadingConsole ? (
              <div className="p-12 text-center font-mono text-xs uppercase tracking-widest text-white/40">
                Polling Supabase database...
              </div>
            ) : filteredRequests.length === 0 ? (
              <div className="p-12 text-center border border-white/10 font-mono text-xs text-white/40">
                No enterprise requests matching current criteria.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left font-mono text-xs border-collapse">
                  <thead>
                    <tr className="border-b border-white/15 text-white/40 uppercase text-[10px] tracking-wider">
                      <th className="py-2.5 px-3">Status</th>
                      <th className="py-2.5 px-3">Contact</th>
                      <th className="py-2.5 px-3">Organization</th>
                      <th className="py-2.5 px-3">Role</th>
                      <th className="py-2.5 px-3">Scope</th>
                      <th className="py-2.5 px-3">Date</th>
                      <th className="py-2.5 px-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-white/10">
                    {filteredRequests.map((req) => (
                      <tr key={req.id} className="hover:bg-white/[0.03] transition-colors">
                        <td className="py-3 px-3">
                          <span
                            className={`border px-2 py-0.5 text-[10px] uppercase font-bold tracking-wider ${
                              req.status === 'new'
                                ? 'border-[#FBFF63] bg-[#FBFF63]/10 text-[#FBFF63]'
                                : req.status === 'in_review'
                                ? 'border-[#A9EBF9] bg-[#A9EBF9]/10 text-[#A9EBF9]'
                                : req.status === 'qualified'
                                ? 'border-[#8DDC96] bg-[#8DDC96]/10 text-[#8DDC96]'
                                : 'border-zinc-600 text-zinc-400'
                            }`}
                          >
                            {req.status.replace('_', ' ')}
                          </span>
                        </td>
                        <td className="py-3 px-3 text-white font-medium">
                          {req.name}
                          <span className="block text-[11px] text-white/50">{req.email}</span>
                        </td>
                        <td className="py-3 px-3 text-white/80">{req.company || '—'}</td>
                        <td className="py-3 px-3 text-[#A9EBF9]">{req.role || '—'}</td>
                        <td className="py-3 px-3 text-white/60">{req.fleet_size || '—'}</td>
                        <td className="py-3 px-3 text-white/40">
                          {new Date(req.created_at).toLocaleDateString()}
                        </td>
                        <td className="py-3 px-3 text-right space-x-2">
                          <button
                            onClick={() => setSelectedReq(req)}
                            className="border border-white/20 px-2.5 py-1 text-[11px] hover:border-white hover:text-white text-white/70 transition-colors"
                          >
                            INSPECT
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </main>

      {/* DETAIL MODAL */}
      {selectedReq && (
        <div className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center p-4">
          <div className="bg-[#0D0D0D] border-2 border-white p-6 md:p-8 max-w-xl w-full">
            <div className="flex justify-between items-start mb-4">
              <div>
                <span className="font-mono text-xs uppercase tracking-widest text-[#A9EBF9]">
                  RECORD ID // {selectedReq.id.slice(0, 8)}
                </span>
                <h3 className="text-2xl font-light text-white mt-1">{selectedReq.name}</h3>
              </div>
              <button
                onClick={() => setSelectedReq(null)}
                className="text-white hover:text-[#FE5934] text-xl"
              >
                ✕
              </button>
            </div>

            <div className="space-y-4 font-mono text-xs">
              <div className="grid grid-cols-2 gap-3 border border-white/10 p-3 bg-black">
                <div>
                  <span className="text-white/40 block">EMAIL:</span>
                  <a
                    href={`mailto:${selectedReq.email}`}
                    className="text-[#A9EBF9] hover:underline break-all"
                  >
                    {selectedReq.email}
                  </a>
                </div>
                <div>
                  <span className="text-white/40 block">ROLE:</span>
                  <span className="text-white">{selectedReq.role || '—'}</span>
                </div>
                <div>
                  <span className="text-white/40 block">COMPANY:</span>
                  <span className="text-white">{selectedReq.company || '—'}</span>
                </div>
                <div>
                  <span className="text-white/40 block">FLEET / SCOPE:</span>
                  <span className="text-white">{selectedReq.fleet_size || '—'}</span>
                </div>
              </div>

              {selectedReq.message && (
                <div>
                  <span className="text-white/40 block mb-1">SPECIFICATIONS / MESSAGE:</span>
                  <p className="bg-black border border-white/10 p-3 text-white/90 font-sans text-sm whitespace-pre-wrap">
                    {selectedReq.message}
                  </p>
                </div>
              )}

              <div className="border border-white/15 p-3 bg-black">
                <span className="text-white/50 block mb-2 uppercase text-[10px]">
                  UPDATE STATUS IN SUPABASE:
                </span>
                <div className="flex flex-wrap gap-2">
                  {(['new', 'in_review', 'contacted', 'qualified', 'closed'] as RequestStatus[]).map(
                    (st) => (
                      <button
                        key={st}
                        onClick={() => updateRequestStatus(selectedReq.id, st)}
                        className={`border px-2.5 py-1 uppercase text-[10px] tracking-wider transition-colors ${
                          selectedReq.status === st
                            ? 'bg-white text-black font-bold'
                            : 'border-white/20 text-white/60 hover:border-white'
                        }`}
                      >
                        {st.replace('_', ' ')}
                      </button>
                    )
                  )}
                </div>
              </div>

              <div className="flex gap-3 pt-2">
                <a
                  href={`mailto:${selectedReq.email}?subject=Pegasus Enterprise Logistics`}
                  className="editorial-btn editorial-btn--inverted flex-1 text-center"
                >
                  Reply via Email
                </a>
                <button
                  onClick={() => setSelectedReq(null)}
                  className="editorial-btn flex-1 text-center"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
