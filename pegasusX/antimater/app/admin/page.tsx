'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { gsap } from 'gsap';
import AdminShell from './components/AdminShell';
import { useLanguage } from '../context/LanguageContext';
import type { EnterpriseRequest, RequestStatus } from '@/app/lib/supabase/types';

export default function AdminPage() {
  const [requests, setRequests] = useState<EnterpriseRequest[]>([]);
  const [selectedReq, setSelectedReq] = useState<EnterpriseRequest | null>(null);
  const [activeFilter, setActiveFilter] = useState<string>('all');
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isUpdating, setIsUpdating] = useState<boolean>(false);
  const [adminNoteInput, setAdminNoteInput] = useState<string>('');
  const listRef = useRef<HTMLDivElement>(null);
  const { t } = useLanguage();

  const loadRequests = useCallback(async () => {
    try {
      const url =
        activeFilter === 'all'
          ? '/api/admin/requests'
          : `/api/admin/requests?status=${activeFilter}`;

      const res = await fetch(url);
      if (res.ok) {
        const json = await res.json();
        if (json.data) {
          setRequests(json.data);
          return;
        }
      }
    } catch (err) {
      console.warn('[ADMIN] Could not fetch from Supabase API, checking fallback:', err);
    }

    // Fallback to localStorage if API is unreachable
    try {
      const stored = localStorage.getItem('team_applications');
      if (stored) {
        const apps = JSON.parse(stored);
        setRequests(
          apps.map((a: any) => ({
            id: a.id || crypto.randomUUID(),
            created_at: a.timestamp || a.created_at || new Date().toISOString(),
            updated_at: a.timestamp || a.created_at || new Date().toISOString(),
            name: a.name || 'Anonymous',
            email: a.email || '',
            company: a.portfolio || a.company || '',
            role: a.position || a.role || 'Supplier Operations',
            fleet_size: a.fleet_size || null,
            inquiry_type: 'demo',
            subject: 'Demo Request',
            message: a.message || '',
            status: a.read ? 'contacted' : 'new',
            priority: 'normal',
          }))
        );
      }
    } catch {}
  }, [activeFilter]);

  useEffect(() => {
    document.title = 'Admin Dashboard | Enterprise Requests';
    setIsLoading(true);
    loadRequests().finally(() => setIsLoading(false));

    // Live refresh every 6 seconds
    const interval = setInterval(() => {
      loadRequests();
    }, 6000);

    return () => clearInterval(interval);
  }, [loadRequests]);

  useEffect(() => {
    if (listRef.current) {
      gsap.fromTo(
        listRef.current,
        { opacity: 0, y: 15 },
        { opacity: 1, y: 0, duration: 0.4, ease: 'power2.out' }
      );
    }
  }, [activeFilter]);

  const updateStatus = async (id: string, newStatus: RequestStatus) => {
    setIsUpdating(true);
    try {
      const res = await fetch('/api/admin/requests', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id,
          status: newStatus,
          admin_notes: adminNoteInput.trim() || undefined,
        }),
      });

      if (res.ok) {
        setRequests((prev) =>
          prev.map((r) =>
            r.id === id
              ? {
                  ...r,
                  status: newStatus,
                  admin_notes: adminNoteInput.trim() || r.admin_notes,
                  updated_at: new Date().toISOString(),
                }
              : r
          )
        );

        if (selectedReq && selectedReq.id === id) {
          setSelectedReq((prev) =>
            prev
              ? {
                  ...prev,
                  status: newStatus,
                  admin_notes: adminNoteInput.trim() || prev.admin_notes,
                }
              : null
          );
        }
      }
    } catch (err) {
      console.error('[ADMIN] Error updating request status:', err);
    } finally {
      setIsUpdating(false);
    }
  };

  const deleteRequest = async (id: string) => {
    if (!confirm('Are you sure you want to delete this request record?')) return;

    try {
      await fetch(`/api/admin/requests?id=${id}`, { method: 'DELETE' });
      setRequests((prev) => prev.filter((r) => r.id !== id));
      setSelectedReq(null);
    } catch (err) {
      console.error('[ADMIN] Error deleting request:', err);
    }
  };

  const openModal = (req: EnterpriseRequest) => {
    setSelectedReq(req);
    setAdminNoteInput(req.admin_notes || '');

    // Auto mark 'new' to 'in_review' upon opening
    if (req.status === 'new') {
      updateStatus(req.id, 'in_review');
    }
  };

  const closeModal = () => {
    setSelectedReq(null);
  };

  const getStatusBadge = (status: RequestStatus) => {
    switch (status) {
      case 'new':
        return 'border-[#FBFF63] bg-[#FBFF63]/10 text-[#FBFF63]';
      case 'in_review':
        return 'border-[#A9EBF9] bg-[#A9EBF9]/10 text-[#A9EBF9]';
      case 'contacted':
        return 'border-blue-400 bg-blue-500/10 text-blue-400';
      case 'qualified':
        return 'border-[#8DDC96] bg-[#8DDC96]/10 text-[#8DDC96]';
      case 'closed':
        return 'border-zinc-600 bg-zinc-800 text-zinc-400';
      default:
        return 'border-zinc-500 text-zinc-400';
    }
  };

  const unreadCount = requests.filter((r) => r.status === 'new').length;

  return (
    <AdminShell
      title={t('admin_title', 'Enterprise Requests & Demos')}
      subtitle={t('admin_subtitle', 'Live Supabase pipeline for partner demos, fleet onboarding, and requests.')}
      badge={
        unreadCount > 0 ? (
          <span className="border border-[#FBFF63] bg-[#FBFF63]/10 px-4 py-2 font-mono text-sm text-[#FBFF63]">
            {unreadCount} {t('admin_new', 'NEW REQUESTS')}
          </span>
        ) : undefined
      }
      nav={[
        { label: t('admin_nav_apps', 'Demo Requests'), href: '/admin', active: true },
        { label: t('admin_nav_messages', 'General Inquiries'), href: '/admin/messages' },
      ]}
    >
      {/* FILTER BUTTONS */}
      <div className="mb-6 flex flex-wrap gap-2 font-mono text-xs">
        {['all', 'new', 'in_review', 'contacted', 'qualified', 'closed'].map((filter) => (
          <button
            key={filter}
            onClick={() => setActiveFilter(filter)}
            className={`border px-3 py-1.5 uppercase transition-colors ${
              activeFilter === filter
                ? 'border-white bg-white text-black'
                : 'border-white/20 text-white/60 hover:border-white/40 hover:text-white'
            }`}
          >
            {filter.replace('_', ' ')}
          </button>
        ))}
      </div>

      <div ref={listRef}>
        {isLoading ? (
          <div className="border border-white/10 p-12 text-center font-mono text-xs uppercase tracking-widest text-white/40">
            Connecting to Supabase...
          </div>
        ) : requests.length === 0 ? (
          <div className="border border-white/15 p-8 text-center md:p-12">
            <h3 className="text-xl font-semibold">{t('admin_no_apps', 'No requests found')}</h3>
            <p className="mt-2 text-sm text-white/50">
              {t('admin_no_apps_desc', 'Enterprise inquiries and demo submissions will appear here.')}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {requests.map((req) => (
              <button
                key={req.id}
                type="button"
                className={`w-full border p-4 text-left transition-colors md:p-5 ${
                  req.status === 'new'
                    ? 'border-[#FBFF63]/60 bg-black'
                    : 'border-white/15 hover:border-white/30'
                }`}
                onClick={() => openModal(req)}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-white">{req.name}</h3>
                      <span
                        className={`border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${getStatusBadge(
                          req.status
                        )}`}
                      >
                        {req.status.replace('_', ' ')}
                      </span>
                      {req.inquiry_type && (
                        <span className="border border-white/20 px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider text-white/60">
                          {req.inquiry_type}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-white/60">{req.email}</p>
                    <div className="mt-2 flex flex-wrap gap-3 font-mono text-xs">
                      {req.role && <span className="text-[#A9EBF9]">Role: {req.role}</span>}
                      {req.company && <span className="text-white/40">Org: {req.company}</span>}
                      {req.fleet_size && <span className="text-white/40">Scope: {req.fleet_size}</span>}
                    </div>
                  </div>
                  <div className="text-right">
                    <span className="font-mono text-[10px] text-white/35 block">
                      {new Date(req.created_at).toLocaleDateString()} {new Date(req.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </span>
                    {req.source_page && (
                      <span className="font-mono text-[10px] text-white/20 block mt-1">
                        {req.source_page}
                      </span>
                    )}
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* DETAIL MODAL */}
      {selectedReq && (
        <div className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center p-4">
          <div className="bg-[#0D0D0D] border-2 border-white p-6 md:p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-6">
              <div>
                <span
                  className={`border px-2 py-0.5 font-mono text-xs uppercase tracking-wider ${getStatusBadge(
                    selectedReq.status
                  )}`}
                >
                  {selectedReq.status.replace('_', ' ')}
                </span>
                <h2 className="text-2xl md:text-3xl font-light mt-2 text-white">
                  {selectedReq.name}
                </h2>
              </div>
              <button
                onClick={closeModal}
                className="text-white hover:text-[#FE5934] transition-colors text-xl md:text-2xl ml-4"
              >
                ✕
              </button>
            </div>

            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">
                    Email
                  </label>
                  <a
                    href={`mailto:${selectedReq.email}?subject=Pegasus Platform Inquiry`}
                    className="text-lg font-light mt-1 block text-[#A9EBF9] hover:underline break-all"
                  >
                    {selectedReq.email}
                  </a>
                </div>
                <div>
                  <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">
                    Role / Position
                  </label>
                  <p className="text-lg font-light mt-1 text-white">{selectedReq.role || '—'}</p>
                </div>
                <div>
                  <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">
                    Company
                  </label>
                  <p className="text-lg font-light mt-1 text-white">{selectedReq.company || '—'}</p>
                </div>
                <div>
                  <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">
                    Scope / Fleet Size
                  </label>
                  <p className="text-lg font-light mt-1 text-white">{selectedReq.fleet_size || '—'}</p>
                </div>
              </div>

              {selectedReq.message && (
                <div>
                  <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">
                    Operational Message
                  </label>
                  <p className="text-sm md:text-base mt-2 leading-relaxed bg-black p-4 border border-white/20 whitespace-pre-wrap text-white/90 font-sans">
                    {selectedReq.message}
                  </p>
                </div>
              )}

              {/* CRM STATUS WORKFLOW */}
              <div className="border border-white/20 p-4 bg-white/[0.02]">
                <label className="text-xs text-white/70 uppercase tracking-wider font-mono block mb-2">
                  Workflow Status
                </label>
                <div className="flex flex-wrap gap-2">
                  {(['new', 'in_review', 'contacted', 'qualified', 'closed'] as RequestStatus[]).map(
                    (st) => (
                      <button
                        key={st}
                        disabled={isUpdating || selectedReq.status === st}
                        onClick={() => updateStatus(selectedReq.id, st)}
                        className={`border px-3 py-1.5 font-mono text-xs uppercase tracking-wider transition-colors ${
                          selectedReq.status === st
                            ? 'border-white bg-white text-black font-semibold'
                            : 'border-white/20 text-white/60 hover:border-white/50 hover:text-white'
                        }`}
                      >
                        {st.replace('_', ' ')}
                      </button>
                    )
                  )}
                </div>
              </div>

              {/* ADMIN NOTES */}
              <div>
                <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">
                  Internal Notes / Next Action
                </label>
                <div className="mt-2 flex gap-2">
                  <input
                    type="text"
                    value={adminNoteInput}
                    onChange={(e) => setAdminNoteInput(e.target.value)}
                    placeholder="Add follow-up notes or operator assigned..."
                    className="flex-1 bg-black border border-white/30 px-3 py-2 text-sm text-white focus:outline-none focus:border-[#A9EBF9]"
                  />
                  <button
                    onClick={() => updateStatus(selectedReq.id, selectedReq.status)}
                    disabled={isUpdating}
                    className="border border-white/30 px-4 py-2 text-xs font-mono uppercase tracking-wider hover:bg-white hover:text-black transition-colors"
                  >
                    Save
                  </button>
                </div>
              </div>

              <div className="flex flex-wrap gap-4 pt-4 border-t border-white/10">
                <a
                  href={`mailto:${selectedReq.email}?subject=Pegasus Demo Follow-up`}
                  className="editorial-btn editorial-btn--inverted flex-1 text-center"
                >
                  Reply via Email
                </a>
                <button
                  type="button"
                  onClick={() => deleteRequest(selectedReq.id)}
                  className="editorial-btn flex-1 text-[#FE5934] hover:border-[#FE5934]"
                >
                  Delete Request
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}