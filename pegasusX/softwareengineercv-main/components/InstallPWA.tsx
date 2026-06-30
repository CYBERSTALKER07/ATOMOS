'use client';

import React, { useEffect, useRef, useState } from 'react';
import gsap from 'gsap';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
} 

const InstallPWA: React.FC = () => {
  const [deferredPrompt, setDeferredPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [isInstallable, setIsInstallable] = useState(false);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const iconRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    // Check if already installed
    if (window.matchMedia('(display-mode: standalone)').matches) {
      setIsInstallable(false);
      return;
    }

    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e as BeforeInstallPromptEvent);
      setIsInstallable(true);
      
      // Animate button entrance
      if (buttonRef.current) {
        gsap.fromTo(
          buttonRef.current,
          { opacity: 0, scale: 0.8, y: 20 },
          { opacity: 1, scale: 1, y: 0, duration: 0.6, ease: 'back.out(1.7)' }
        );
      }
    };

    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);

    return () => {
      window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
    };
  }, []);

  const handleInstallClick = async () => {
    if (!deferredPrompt) return;

    // Animate click
    if (buttonRef.current) {
      gsap.to(buttonRef.current, {
        scale: 0.95,
        duration: 0.1,
        yoyo: true,
        repeat: 1,
      });
    }

    deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;

    if (outcome === 'accepted') {
      setIsInstallable(false);
      // Animate button exit
      if (buttonRef.current) {
        gsap.to(buttonRef.current, {
          opacity: 0,
          scale: 0.8,
          y: -20,
          duration: 0.4,
          ease: 'back.in(1.7)',
        });
      }
    }

    setDeferredPrompt(null);
  };

  const handleMouseEnter = () => {
    if (iconRef.current) {
      gsap.to(iconRef.current, {
        rotation: 360,
        duration: 0.6,
        ease: 'back.out(1.7)',
      });
    }
  };

  if (!isInstallable) return null;

  return (
    <button
      ref={buttonRef}
      onClick={handleInstallClick}
      onMouseEnter={handleMouseEnter}
      className="editorial-btn editorial-btn--shadow fixed bottom-8 right-8 z-50 flex items-center gap-3 group"
      aria-label="Install App"
    >
      <span
        ref={iconRef}
        className="text-2xl"
        role="img"
        aria-hidden="true"
      >
        💾
      </span>
      <span className="font-semibold text-sm md:text-base whitespace-nowrap">
        Install App
      </span>
    </button>
  );
};

export default InstallPWA;
