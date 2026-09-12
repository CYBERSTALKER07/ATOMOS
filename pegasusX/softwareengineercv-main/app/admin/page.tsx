'use client';

import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import AdminShell from './components/AdminShell';
import { useLanguage } from '../context/LanguageContext';

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
  const listRef = useRef<HTMLDivElement>(null);
  const notificationRef = useRef<HTMLDivElement>(null);
  const { t } = useLanguage();

  useEffect(() => {
    document.title = 'Admin Dashboard | Applications';

    if (listRef.current) {
      gsap.fromTo(
        listRef.current,
        { opacity: 0, y: 20 },
        { opacity: 1, y: 0, duration: 0.6, ease: 'power3.out' }
      );
    }

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
    <AdminShell
      title={t('admin_title')}
      subtitle={t('admin_subtitle')}
      badge={
        unreadCount > 0 ? (
          <span className="border border-[#FBFF63] bg-[#FBFF63]/10 px-4 py-2 font-mono text-sm text-[#FBFF63]">
            {unreadCount} {t('admin_new')}
          </span>
        ) : undefined
      }
      nav={[
        { label: t('admin_nav_apps'), href: '/admin', active: true },
        { label: t('admin_nav_messages'), href: '/admin/messages' },
      ]}
    >
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
              <h4 className="font-light text-base md:text-lg mb-1 text-[#8DDC96]">{t('admin_new_app')}</h4>
              <p className="text-white font-semibold text-sm md:text-base truncate">{notification.name}</p>
              <p className="text-gray-400 text-xs md:text-sm truncate">{notification.position}</p>
              <button
                onClick={() => viewApplication(notification)}
                className="mt-2 md:mt-3 text-[#8DDC96] hover:text-white transition-colors text-xs md:text-sm font-semibold"
              >
                {t('admin_view_details')}
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

      <div ref={listRef}>
          {applications.length === 0 ? (
            <div className="border border-white/15 p-8 text-center md:p-12">
              <h3 className="text-xl font-semibold">{t('admin_no_apps')}</h3>
              <p className="mt-2 text-sm text-white/50">
                {t('admin_no_apps_desc')}
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {applications.map((app) => (
                <button
                  key={app.id}
                  type="button"
                  className={`w-full border p-4 text-left transition-colors md:p-5 ${
                    app.read
                      ? 'border-white/15 hover:border-white/30'
                      : 'border-[#FBFF63]/50 bg-[#0a0a0a]'
                  }`}
                  onClick={() => viewApplication(app)}
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold">{app.name}</h3>
                        {!app.read ? (
                          <span className="bg-[#FBFF63] px-2 py-0.5 font-mono text-[10px] text-black">
                            {t('admin_badge_new')}
                          </span>
                        ) : null}
                      </div>
                      <p className="mt-1 text-sm text-white/50">{app.email}</p>
                      <p className="mt-2 font-mono text-xs text-[#A9EBF9]">{app.position}</p>
                    </div>
                    <span className="font-mono text-[10px] text-white/35">
                      {new Date(app.timestamp).toLocaleString()}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          )}
      </div>

      {/* Modal */}
      {selectedApp && (
        <div className="fixed inset-0 bg-black bg-opacity-90 z-50 flex items-center justify-center p-4">
          <div className="modal-content bg-[#0D0D0D] border-2 border-white rounded-2xl p-6 md:p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-6">
              <h2 className="text-2xl md:text-3xl font-light">{t('admin_modal_title')}</h2>
              <button
                onClick={closeModal}
                className="text-white hover:text-[#FE5934] transition-colors text-xl md:text-2xl flex-shrink-0 ml-4"
              >
                ✕
              </button>
            </div>

            <div className="space-y-6">
              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">{t('admin_label_name')}</label>
                <p className="text-lg md:text-xl font-light mt-1 break-words">{selectedApp.name}</p>
              </div>

              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">{t('admin_label_email')}</label>
                <a 
                  href={`mailto:${selectedApp.email}`}
                  className="text-lg md:text-xl font-light mt-1 block text-[#A9EBF9] hover:text-white transition-colors break-all"
                >
                  {selectedApp.email}
                </a>
              </div>

              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">{t('admin_label_position')}</label>
                <p className="text-lg md:text-xl font-light mt-1 break-words">{selectedApp.position}</p>
              </div>

              {selectedApp.portfolio && (
                <div>
                  <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">{t('admin_label_portfolio')}</label>
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
                  <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">{t('admin_label_message')}</label>
                  <p className="text-sm md:text-lg mt-2 leading-relaxed bg-black p-4 rounded-xl border-2 border-white break-words">
                    {selectedApp.message}
                  </p>
                </div>
              )}

              <div>
                <label className="text-xs md:text-sm text-gray-400 uppercase tracking-wider">{t('admin_label_submitted')}</label>
                <p className="text-sm md:text-lg mt-1">{new Date(selectedApp.timestamp).toLocaleString()}</p>
              </div>

              <div className="flex flex-col sm:flex-row gap-4 pt-4">
                <button type="button" onClick={closeModal} className="editorial-btn editorial-btn--inverted flex-1">
                  {t('admin_btn_close')}
                </button>
                <button type="button" onClick={() => deleteApplication(selectedApp.id)} className="editorial-btn flex-1">
                  {t('admin_btn_delete')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

    </AdminShell>
  );
}