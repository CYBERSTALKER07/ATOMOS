'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';

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
  const headerRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const notificationRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    document.title = 'Customer Messages | Admin Dashboard';
    
    // Animate header
    gsap.fromTo(headerRef.current,
      { opacity: 0, y: -50 },
      { opacity: 1, y: 0, duration: 1, ease: 'power3.out' }
    );

    // Animate list
    gsap.fromTo(listRef.current,
      { opacity: 0, x: -50 },
      { opacity: 1, x: 0, duration: 1, delay: 0.3, ease: 'power3.out' }
    );

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
    <div className="min-h-screen bg-black text-white relative overflow-hidden">
      {/* Navigation */}
      <nav className="fixed top-8 left-8 z-50 flex gap-4">
        <Link href="/admin" className="editorial-btn editorial-btn--sm editorial-btn--inverted">
          <span>←</span>
          <span>Applications</span>
        </Link>
        <Link href="/" className="editorial-btn editorial-btn--sm">
          Home
        </Link>
      </nav>

      {/* Custom Notification */}
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
              <h4 className="font-bold text-lg mb-1 text-[#A9EBF9]">New Message!</h4>
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

      {/* Header */}
      <div ref={headerRef} className="container mx-auto px-4 pt-32 pb-12">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-5xl md:text-6xl font-bold mb-4">
                Customer Messages
              </h1>
              <div className="w-24 h-1 bg-white mb-4" />
              <p className="text-xl text-gray-400">
                View and manage customer inquiries
              </p>
            </div>
            
            {unreadCount > 0 && (
              <div className="bg-[#A9EBF9] text-black px-6 py-3 rounded-2xl border-2 border-[#A9EBF9]">
                <span className="text-2xl font-bold">{unreadCount}</span>
                <span className="ml-2 text-sm">Unread</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Messages List */}
      <div ref={listRef} className="container mx-auto px-4 pb-20">
        <div className="max-w-7xl mx-auto">
          {messages.length === 0 ? (
            <div className="border-2 border-white rounded-2xl p-12 text-center">
              <div className="text-6xl mb-4">📬</div>
              <h3 className="text-2xl font-bold mb-2">No Messages Yet</h3>
              <p className="text-gray-400">
                Customer messages will appear here when they contact you
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {messages.map((msg, index) => (
                <div
                  key={msg.id}
                  className={`border-2 rounded-2xl p-6 cursor-pointer transition-all duration-300 ${
                    msg.read 
                      ? 'border-white hover:bg-[#0D0D0D]' 
                      : 'border-[#A9EBF9] bg-[#0D0D0D] hover:border-[#BDE7FF]'
                  }`}
                  onClick={() => viewMessage(msg)}
                  style={{
                    animation: `slideIn 0.5s ease-out ${index * 0.1}s both`
                  }}
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-2xl font-bold">{msg.name}</h3>
                        {!msg.read && (
                          <span className="bg-[#A9EBF9] text-black px-3 py-1 rounded-lg text-xs font-bold">
                            NEW
                          </span>
                        )}
                      </div>
                      <p className="text-gray-400 mb-2">{msg.email}</p>
                      <p className="text-lg font-semibold mb-2">{msg.subject}</p>
                      <p className="text-gray-400 text-sm line-clamp-2">{msg.message}</p>
                      <p className="text-gray-500 text-sm mt-2">
                        {new Date(msg.timestamp).toLocaleString()}
                      </p>
                    </div>
                    <div className="text-4xl">💬</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Modal */}
      {selectedMsg && (
        <div className="fixed inset-0 bg-black bg-opacity-90 z-50 flex items-center justify-center p-4">
          <div className="modal-content bg-[#0D0D0D] border-2 border-white rounded-2xl p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-6">
              <h2 className="text-3xl font-bold">Message Details</h2>
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
                <p className="text-xl font-bold mt-1">{selectedMsg.name}</p>
              </div>

              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">Email</label>
                <a 
                  href={`mailto:${selectedMsg.email}`}
                  className="text-xl font-bold mt-1 block text-[#A9EBF9] hover:text-white transition-colors"
                >
                  {selectedMsg.email}
                </a>
              </div>

              <div>
                <label className="text-sm text-gray-400 uppercase tracking-wider">Subject</label>
                <p className="text-xl font-bold mt-1">{selectedMsg.subject}</p>
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

      <style jsx>{`
        @keyframes slideIn {
          from {
            opacity: 0;
            transform: translateX(-30px);
          }
          to {
            opacity: 1;
            transform: translateX(0);
          }
        }
      `}</style>
    </div>
  );
}