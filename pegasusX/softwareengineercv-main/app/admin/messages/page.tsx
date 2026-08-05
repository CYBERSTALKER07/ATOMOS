'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import AdminShell from '../components/AdminShell';

interface Message {
  id: string;
  name: string;
  email: string;
  subject: string;
  message: string;
  timestamp: string;
  read: boolean;
}

export default function MessagesPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [selectedMsg, setSelectedMsg] = useState<Message | null>(null);
  const [notification, setNotification] = useState<Message | null>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const notificationRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    document.title = 'Customer Messages | Admin Dashboard';

    if (listRef.current) {
      gsap.fromTo(listRef.current, { opacity: 0, y: 20 }, { opacity: 1, y: 0, duration: 0.6, ease: 'power3.out' });
    }

    // Load messages from localStorage
    loadMessages();

    // Poll for new messages every 3 seconds
    const interval = setInterval(() => {
      loadMessages();
    }, 3000);

    return () => clearInterval(interval);
  }, []);

  const loadMessages = () => {
    const stored = localStorage.getItem('customer_messages');
    if (stored) {
      const msgs = JSON.parse(stored);
      setMessages(msgs);
      
      // Check for new unread messages
      const newMsgs = msgs.filter((msg: Message) => !msg.read);
      if (newMsgs.length > 0 && !notification) {
        showNotification(newMsgs[0]);
      }
    }
  };

  const showNotification = (msg: Message) => {
    setNotification(msg);
    
    // Animate notification
    gsap.fromTo(notificationRef.current,
      { x: 400, opacity: 0, scale: 0.8 },
      { 
        x: 0, 
        opacity: 1, 
        scale: 1, 
        duration: 0.8, 
        ease: 'back.out(1.7)',
        onComplete: () => {
          // Auto-hide after 5 seconds
          setTimeout(() => {
            gsap.to(notificationRef.current, {
              x: 400,
              opacity: 0,
              duration: 0.5,
              ease: 'power2.in',
              onComplete: () => setNotification(null)
            });
          }, 5000);
        }
      }
    );
  };

  const markAsRead = (id: string) => {
    const updated = messages.map(msg => 
      msg.id === id ? { ...msg, read: true } : msg
    );
    setMessages(updated);
    localStorage.setItem('customer_messages', JSON.stringify(updated));
  };

  const viewMessage = (msg: Message) => {
    setSelectedMsg(msg);
    markAsRead(msg.id);
    
    // Animate modal
    gsap.fromTo('.modal-content',
      { scale: 0.8, opacity: 0 },
      { scale: 1, opacity: 1, duration: 0.5, ease: 'back.out(1.7)' }
    );
  };

  const closeModal = () => {
    gsap.to('.modal-content', {
      scale: 0.8,
      opacity: 0,
      duration: 0.3,
      ease: 'power2.in',
      onComplete: () => setSelectedMsg(null)
    });
  };

  const deleteMessage = (id: string) => {
    const updated = messages.filter(msg => msg.id !== id);
    setMessages(updated);
    localStorage.setItem('customer_messages', JSON.stringify(updated));
    setSelectedMsg(null);
  };

  const unreadCount = messages.filter(msg => !msg.read).length;

  return (
    <AdminShell
      title="Customer messages"
      subtitle="View and manage contact form inquiries."
      badge={
        unreadCount > 0 ? (
          <span className="border border-[#A9EBF9] bg-[#A9EBF9]/10 px-4 py-2 font-mono text-sm text-[#A9EBF9]">
            {unreadCount} unread
          </span>
        ) : undefined
      }
      nav={[
        { label: 'Applications', href: '/admin' },
        { label: 'Messages', href: '/admin/messages', active: true },
      ]}
    >
      {notification && (
        <div 
          ref={notificationRef}
          className="fixed top-8 right-8 z-50 bg-[#0D0D0D] border-2 border-[#A9EBF9] rounded-2xl p-6 shadow-2xl w-96"
        >
          <div className="flex items-start gap-4">
            <div className="w-12 h-12 bg-[#A9EBF9] rounded-xl flex items-center justify-center flex-shrink-0">
              <span className="text-2xl">💬</span>
            </div>
            <div className="flex-1">
              <h4 className="font-light text-lg mb-1 text-[#A9EBF9]">New Message!</h4>
              <p className="text-white font-semibold">{notification.name}</p>
              <p className="text-gray-400 text-sm truncate">{notification.subject}</p>
              <button
                onClick={() => viewMessage(notification)}
                className="mt-3 text-[#A9EBF9] hover:text-white transition-colors text-sm font-semibold"
              >
                Read Message →
              </button>
            </div>
            <button
              onClick={() => {
                gsap.to(notificationRef.current, {
                  x: 400,
                  opacity: 0,
                  duration: 0.3,
                  onComplete: () => setNotification(null)
                });
              }}
              className="text-white hover:text-[#FE5934] transition-colors"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      <div ref={listRef}>
        {messages.length === 0 ? (
          <div className="border border-white/15 p-8 text-center md:p-12">
            <h3 className="text-xl font-semibold">No messages yet</h3>
            <p className="mt-2 text-sm text-white/50">
              Customer messages appear here from the contact form.
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
                  msg.read ? 'border-white/15 hover:border-white/30' : 'border-[#A9EBF9]/50 bg-[#0a0a0a]'
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold">{msg.name}</h3>
                      {!msg.read ? (
                        <span className="bg-[#A9EBF9] px-2 py-0.5 font-mono text-[10px] text-black">NEW</span>
                      ) : null}
                    </div>
                    <p className="mt-1 text-sm font-medium text-white/80">{msg.subject}</p>
                    <p className="mt-1 line-clamp-2 text-sm text-white/45">{msg.message}</p>
                  </div>
                  <span className="shrink-0 font-mono text-[10px] text-white/35">
                    {new Date(msg.timestamp).toLocaleDateString()}
                  </span>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Modal */}
      {selectedMsg && (
        <div className="fixed inset-0 bg-black bg-opacity-90 z-50 flex items-center justify-center p-4">
          <div className="modal-content bg-[#0D0D0D] border-2 border-white rounded-2xl p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-6">
              <h2 className="text-3xl font-light">Message Details</h2>
              <button
                onClick={closeModal}
                className="text-white hover:text-[#FE5934] transition-colors text-2xl"
              >
                ✕
              </button>
            </div>

            <div className="space-y-6">
              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">From</label>
                <p className="text-xl font-light mt-1">{selectedMsg.name}</p>
              </div>

              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">Email</label>
                <a 
                  href={`mailto:${selectedMsg.email}`}
                  className="text-xl font-light mt-1 block text-[#A9EBF9] hover:text-white transition-colors"
                >
                  {selectedMsg.email}
                </a>
              </div>

              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">Subject</label>
                <p className="text-xl font-light mt-1">{selectedMsg.subject}</p>
              </div>

              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">Message</label>
                <p className="text-lg mt-2 leading-relaxed bg-black p-4 rounded-xl border-2 border-white whitespace-pre-wrap">
                  {selectedMsg.message}
                </p>
              </div>

              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">Received</label>
                <p className="text-lg mt-1">{new Date(selectedMsg.timestamp).toLocaleString()}</p>
              </div>

              <div className="flex gap-4 pt-4">
                <a
                  href={`mailto:${selectedMsg.email}?subject=Re: ${selectedMsg.subject}`}
                  className="editorial-btn editorial-btn--inverted flex-1"
                >
                  Reply via Email
                </a>
                <button type="button" onClick={() => deleteMessage(selectedMsg.id)} className="editorial-btn flex-1">
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