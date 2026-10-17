'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { gsap } from 'gsap';
import AdminShell from '../components/AdminShell';
import { useLanguage } from '@/app/context/LanguageContext';
import type { EnterpriseRequest, RequestStatus } from '@/app/lib/supabase/types';

export default function MessagesPage() {
  const { t } = useLanguage();
  const [messages, setMessages] = useState<EnterpriseRequest[]>([]);
  const [selectedMsg, setSelectedMsg] = useState<EnterpriseRequest | null>(null);
  const [activeFilter, setActiveFilter] = useState<string>('all');
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isUpdating, setIsUpdating] = useState<boolean>(false);
  const listRef = useRef<HTMLDivElement>(null);

  const loadMessages = useCallback(async () => {
    try {
      const url =
        activeFilter === 'all'
          ? '/api/admin/requests?inquiry_type=general'
          : `/api/admin/requests?inquiry_type=general&status=${activeFilter}`;

      const res = await fetch(url);
      if (res.ok) {
        const json = await res.json();
        if (json.data) {
          setMessages(json.data);
          return;
        }
      }
    } catch (err) {
      console.warn('[ADMIN_MESSAGES] Could not fetch from Supabase API, falling back:', err);
    }

    // Fallback to localStorage if offline
    try {
      const stored = localStorage.getItem('customer_messages');
      if (stored) {
        const msgs = JSON.parse(stored);
        setMessages(
          msgs.map((m: any) => ({
            id: m.id || crypto.randomUUID(),
            created_at: m.timestamp || m.created_at || new Date().toISOString(),
            updated_at: m.timestamp || m.created_at || new Date().toISOString(),
            name: m.name || 'Anonymous',
            email: m.email || '',
            company: m.company || null,
            role: null,
            inquiry_type: 'general',
            subject: m.subject || 'General Inquiry',
            message: m.message || '',
            status: m.read ? 'contacted' : 'new',
            priority: 'normal',
          }))
        );
      }
    } catch {}
  }, [activeFilter]);

  useEffect(() => {
    document.title = `${t('admin_nav_messages', 'Messages')} | Pegasus Admin`;
    setIsLoading(true);
    loadMessages().finally(() => setIsLoading(false));

    const interval = setInterval(() => {
      loadMessages();
    }, 6000);

    return () => clearInterval(interval);
  }, [loadMessages, t]);

  useEffect(() => {
    if (listRef.current) {
      gsap.fromTo(listRef.current, { opacity: 0, y: 15 }, { opacity: 1, y: 0, duration: 0.4, ease: 'power2.out' });
    }
  }, [activeFilter]);

  const updateStatus = async (id: string, newStatus: RequestStatus) => {
    setIsUpdating(true);
    try {
      const res = await fetch('/api/admin/requests', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id, status: newStatus }),
      });

      if (res.ok) {
        setMessages((prev) =>
          prev.map((m) => (m.id === id ? { ...m, status: newStatus, updated_at: new Date().toISOString() } : m))
        );

        if (selectedMsg && selectedMsg.id === id) {
          setSelectedMsg((prev) => (prev ? { ...prev, status: newStatus } : null));
        }
      }
    } catch (err) {
      console.error('[ADMIN_MESSAGES] Error updating message status:', err);
    } finally {
      setIsUpdating(false);
    }
  };

  const deleteMessage = async (id: string) => {
    if (!confirm('Are you sure you want to delete this message?')) return;

    try {
      await fetch(`/api/admin/requests?id=${id}`, { method: 'DELETE' });
      setMessages((prev) => prev.filter((m) => m.id !== id));
      setSelectedMsg(null);
    } catch (err) {
      console.error('[ADMIN_MESSAGES] Error deleting message:', err);
    }
  };

  const viewMessage = (msg: EnterpriseRequest) => {
    setSelectedMsg(msg);
    if (msg.status === 'new') {
      updateStatus(msg.id, 'in_review');
    }
  };

  const closeModal = () => {
    setSelectedMsg(null);
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

  const unreadCount = messages.filter((m) => m.status === 'new').length;

  return (
    <AdminShell
      title={t('admin_messages_title', 'Customer Inquiries & Messages')}
      subtitle={t('admin_messages_sub', 'Live Supabase inbox for website contacts, partnerships, and general queries.')}
      badge={
        unreadCount > 0 ? (
          <span className="border border-[#A9EBF9] bg-[#A9EBF9]/10 px-4 py-2 font-mono text-sm text-[#A9EBF9]">
            {unreadCount} {t('admin_badge_new', 'UNREAD')}
          </span>
        ) : undefined
      }
      nav={[
        { label: t('admin_nav_apps', 'Demo Requests'), href: '/admin' },
        { label: t('admin_nav_messages', 'General Inquiries'), href: '/admin/messages', active: true },
      ]}
    >
      {/* FILTER BUTTONS */}
      <div className="mb-6 flex flex-wrap gap-2 font-mono text-xs">
        {['all', 'new', 'in_review', 'contacted', 'closed'].map((filter) => (
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
            Loading messages from Supabase...
          </div>
        ) : messages.length === 0 ? (
          <div className="border border-white/15 p-8 text-center md:p-12">
            <h3 className="text-xl font-semibold">{t('admin_no_msgs', 'No messages found')}</h3>
            <p className="mt-2 text-sm text-white/50">
              {t('admin_no_msgs_desc', 'Customer messages and inquiries will appear here.')}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {messages.map((msg) => (
              <button
                key={msg.id}
                type="button"
                onClick={() => viewMessage(msg)}
                className={`w-full border p-4 text-left transition-colors md:p-5 ${
                  msg.status === 'new'
                    ? 'border-[#A9EBF9]/70 bg-black'
                    : 'border-white/15 hover:border-white/30'
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-white">{msg.name}</h3>
                      <span
                        className={`border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${getStatusBadge(
                          msg.status
                        )}`}
                      >
                        {msg.status.replace('_', ' ')}
                      </span>
                      {msg.company && (
                        <span className="font-mono text-xs text-white/40">({msg.company})</span>
                      )}
                    </div>
                    <p className="mt-1 text-sm font-medium text-white/80">
                      {msg.subject || 'General Inquiry'}
                    </p>
                    <p className="mt-1 line-clamp-2 text-sm text-white/45">{msg.message}</p>
                  </div>
                  <span className="shrink-0 font-mono text-[10px] text-white/35">
                    {new Date(msg.created_at).toLocaleDateString()}
                  </span>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* MODAL */}
      {selectedMsg && (
        <div className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center p-4">
          <div className="bg-[#0D0D0D] border-2 border-white p-6 md:p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-6">
              <div>
                <span
                  className={`border px-2 py-0.5 font-mono text-xs uppercase tracking-wider ${getStatusBadge(
                    selectedMsg.status
                  )}`}
                >
                  {selectedMsg.status.replace('_', ' ')}
                </span>
                <h2 className="text-2xl md:text-3xl font-light mt-2 text-white">{selectedMsg.name}</h2>
              </div>
              <button
                onClick={closeModal}
                className="text-white hover:text-[#FE5934] transition-colors text-2xl"
              >
                ✕
              </button>
            </div>

            <div className="space-y-6">
              <div>
                <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">Email</label>
                <a
                  href={`mailto:${selectedMsg.email}?subject=Re: ${selectedMsg.subject || 'Pegasus Inquiry'}`}
                  className="text-xl font-light mt-1 block text-[#A9EBF9] hover:underline break-all"
                >
                  {selectedMsg.email}
                </a>
              </div>

              {selectedMsg.company && (
                <div>
                  <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">Company</label>
                  <p className="text-lg font-light mt-1 text-white">{selectedMsg.company}</p>
                </div>
              )}

              <div>
                <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">Subject</label>
                <p className="text-lg font-light mt-1 text-white">{selectedMsg.subject || 'General Inquiry'}</p>
              </div>

              <div>
                <label className="text-xs text-gray-400 uppercase tracking-wider font-mono">Message</label>
                <p className="text-base mt-2 leading-relaxed bg-black p-4 border border-white/20 whitespace-pre-wrap text-white font-sans">
                  {selectedMsg.message}
                </p>
              </div>

              {/* WORKFLOW STATUS BUTTONS */}
              <div className="border border-white/20 p-4 bg-white/[0.02]">
                <label className="text-xs text-white/70 uppercase tracking-wider font-mono block mb-2">
                  Update Status
                </label>
                <div className="flex flex-wrap gap-2">
                  {(['new', 'in_review', 'contacted', 'closed'] as RequestStatus[]).map((st) => (
                    <button
                      key={st}
                      disabled={isUpdating || selectedMsg.status === st}
                      onClick={() => updateStatus(selectedMsg.id, st)}
                      className={`border px-3 py-1.5 font-mono text-xs uppercase tracking-wider transition-colors ${
                        selectedMsg.status === st
                          ? 'border-white bg-white text-black font-semibold'
                          : 'border-white/20 text-white/60 hover:border-white/50 hover:text-white'
                      }`}
                    >
                      {st.replace('_', ' ')}
                    </button>
                  ))}
                </div>
              </div>

              <div className="flex gap-4 pt-4 border-t border-white/10">
                <a
                  href={`mailto:${selectedMsg.email}?subject=Re: ${selectedMsg.subject || 'Pegasus Inquiry'}`}
                  className="editorial-btn editorial-btn--inverted flex-1 text-center"
                >
                  Reply via Email
                </a>
                <button
                  type="button"
                  onClick={() => deleteMessage(selectedMsg.id)}
                  className="editorial-btn flex-1 text-[#FE5934] hover:border-[#FE5934]"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}