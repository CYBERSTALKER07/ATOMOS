'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import Link from 'next/link';

interface Application {
  id: string;
  name: string;
  email: string;
  position: string;
  portfolio?: string;
  message?: string;
  timestamp: string;
  read: boolean;
}

export default function AdminPage() {
  const [applications, setApplications] = useState<Application[]>([]);
  const [selectedApp, setSelectedApp] = useState<Application | null>(null);
  const [notification, setNotification] = useState<Application | null>(null);
  const headerRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const notificationRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    document.title = 'Admin Dashboard | Applications';
    
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

    // Load applications from localStorage
    loadApplications();

    // Poll for new applications every 3 seconds
    const interval = setInterval(() => {
      loadApplications();
    }, 3000);

    return () => clearInterval(interval);
  }, []);

  const loadApplications = () => {
    const stored = localStorage.getItem('team_applications');
    if (stored) {
      const apps = JSON.parse(stored);
      setApplications(apps);
      
      // Check for new unread applications
      const newApps = apps.filter((app: Application) => !app.read);
      if (newApps.length > 0 && !notification) {
        showNotification(newApps[0]);
      }
    }
  };

  const showNotification = (app: Application) => {
    setNotification(app);
    
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
    const updated = applications.map(app => 
      app.id === id ? { ...app, read: true } : app
    );
    setApplications(updated);
    localStorage.setItem('team_applications', JSON.stringify(updated));
  };

  const viewApplication = (app: Application) => {
    setSelectedApp(app);
    markAsRead(app.id);
    
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
      onComplete: () => setSelectedApp(null)
    });
  };

  const deleteApplication = (id: string) => {
    const updated = applications.filter(app => app.id !== id);
    setApplications(updated);
    localStorage.setItem('team_applications', JSON.stringify(updated));
    setSelectedApp(null);
  };

  const unreadCount = applications.filter(app => !app.read).length;

  return (
    <div className="min-h-screen bg-black text-white relative overflow-hidden">
      {/* Navigation */}
      <nav className="fixed top-4 left-4 md:top-8 md:left-8 z-50">
        <Link href="/" className="editorial-btn editorial-btn--sm">
          <span>←</span>
          <span className="hidden sm:inline">Back to Home</span>
          <span className="sm:hidden">Home</span>
        </Link>
      </nav>

      {/* Custom Notification */}
      {notification && (
        <div 
          ref={notificationRef}
          className="fixed top-4 right-4 md:top-8 md:right-8 z-50 bg-[#0D0D0D] border-2 border-[#8DDC96] rounded-2xl p-4 md:p-6 shadow-2xl w-80 md:w-96 max-w-[calc(100vw-2rem)]"
        >
          <div className="flex items-start gap-3 md:gap-4">
            <div className="w-10 h-10 md:w-12 md:h-12 bg-[#8DDC96] rounded-xl flex items-center justify-center flex-shrink-0">
              <span className="text-xl md:text-2xl">🎉</span>
            </div>
            <div className="flex-1 min-w-0">
              <h4 className="font-light text-base md:text-lg mb-1 text-[#8DDC96]">New Application!</h4>
              <p className="text-white font-semibold text-sm md:text-base truncate">{notification.name}</p>
              <p className="text-gray-400 text-xs md:text-sm truncate">{notification.position}</p>
              <button
                onClick={() => viewApplication(notification)}
                className="mt-2 md:mt-3 text-[#8DDC96] hover:text-white transition-colors text-xs md:text-sm font-semibold"
              >
                View Details →
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
              className="text-white hover:text-[#FE5934] transition-colors text-lg md:text-xl flex-shrink-0"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {/* Header */}
      <div ref={headerRef} className="container mx-auto px-4 pt-20 md:pt-32 pb-8 md:pb-12">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <h1 className="text-3xl sm:text-4xl md:text-5xl lg:text-6xl font-light mb-4">
                Admin Dashboard
              </h1>
              <div className="w-24 h-1 bg-white mb-4" />
              <p className="text-lg md:text-xl text-gray-400">
                Manage team applications
              </p>
            </div>
            
            {unreadCount > 0 && (
              <div className="bg-[#FE5934] text-white px-4 py-2 md:px-6 md:py-3 rounded-2xl border-2 border-[#FE5934] self-start sm:self-center">
                <span className="text-xl md:text-2xl font-light">{unreadCount}</span>
                <span className="ml-2 text-xs md:text-sm">New</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Applications List */}
      <div ref={listRef} className="container mx-auto px-4 pb-20">
        <div className="max-w-7xl mx-auto">
          {applications.length === 0 ? (
            <div className="border-2 border-white rounded-2xl p-8 md:p-12 text-center">
              <div className="text-4xl md:text-6xl mb-4">📭</div>
              <h3 className="text-xl md:text-2xl font-light mb-2">No Applications Yet</h3>
              <p className="text-gray-400 text-sm md:text-base">
                Applications will appear here when someone submits the form
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {applications.map((app, index) => (
                <div
                  key={app.id}
                  className={`border-2 rounded-2xl p-4 md:p-6 cursor-pointer transition-all duration-300 ${
                    app.read 
                      ? 'border-white hover:bg-[#0D0D0D]' 
                      : 'border-[#FBFF63] bg-[#0D0D0D] hover:border-[#FFA500]'
                  }`}
                  onClick={() => viewApplication(app)}
                  style={{
                    animation: `slideIn 0.5s ease-out ${index * 0.1}s both`
                  }}
                >
                  <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 mb-2">
                        <h3 className="text-xl md:text-2xl font-light truncate">{app.name}</h3>
                        {!app.read && (
                          <span className="bg-[#FBFF63] text-black px-3 py-1 rounded-lg text-xs font-light self-start">
                            NEW
                          </span>
                        )}
                      </div>
                      <p className="text-gray-400 mb-2 text-sm md:text-base break-all">{app.email}</p>
                      <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 text-sm">
                        <span className="bg-[#A9EBF9] text-black px-3 py-1 rounded-lg font-semibold self-start">
                          {app.position}
                        </span>
                        <span className="text-gray-500 text-xs sm:text-sm">
                          {new Date(app.timestamp).toLocaleString()}
                        </span>
                      </div>
                    </div>
                    <div className="text-3xl md:text-4xl self-center sm:self-start flex-shrink-0">
                      {app.position.includes('Frontend') && '🎨'}
                      {app.position.includes('Backend') && '⚙️'}
                      {app.position.includes('Full Stack') && '🚀'}
                      {app.position.includes('Designer') && '✨'}
                      {app.position.includes('DevOps') && '🔧'}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Modal */}
      {selectedApp && (
        <div className="fixed inset-0 bg-black bg-opacity-90 z-50 flex items-center justify-center p-4">
          <div className="modal-content bg-[#0D0D0D] border-2 border-white rounded-2xl p-6 md:p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-6">
              <h2 className="text-2xl md:text-3xl font-light">Application Details</h2>
              <button
                onClick={closeModal}
                className="text-white hover:text-[#FE5934] transition-colors text-xl md:text-2xl flex-shrink-0 ml-4"
              >
                ✕
              </button>
            </div>

            <div className="space-y-6">
              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">Name</label>
                <p className="text-lg md:text-xl font-light mt-1 break-words">{selectedApp.name}</p>
              </div>

              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">Email</label>
                <a 
                  href={`mailto:${selectedApp.email}`}
                  className="text-lg md:text-xl font-light mt-1 block text-[#A9EBF9] hover:text-white transition-colors break-all"
                >
                  {selectedApp.email}
                </a>
              </div>

              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">Position</label>
                <p className="text-lg md:text-xl font-light mt-1 break-words">{selectedApp.position}</p>
              </div>

              {selectedApp.portfolio && (
                <div>
                  <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">Portfolio</label>
                  <a 
                    href={selectedApp.portfolio}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-lg md:text-xl font-light mt-1 block text-[#A9EBF9] hover:text-white transition-colors break-all"
                  >
                    {selectedApp.portfolio}
                  </a>
                </div>
              )}

              {selectedApp.message && (
                <div>
                  <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">Message</label>
                  <p className="text-sm md:text-lg mt-2 leading-relaxed bg-black p-4 rounded-xl border-2 border-white break-words">
                    {selectedApp.message}
                  </p>
                </div>
              )}

              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">Submitted</label>
                <p className="text-sm md:text-lg mt-1">{new Date(selectedApp.timestamp).toLocaleString()}</p>
              </div>

              <div className="flex flex-col sm:flex-row gap-4 pt-4">
                <button type="button" onClick={closeModal} className="editorial-btn editorial-btn--inverted flex-1">
                  Close
                </button>
                <button type="button" onClick={() => deleteApplication(selectedApp.id)} className="editorial-btn flex-1">
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