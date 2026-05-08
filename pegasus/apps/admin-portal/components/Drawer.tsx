'use client';

import React, { useEffect, useRef } from 'react';
import { Button } from '@heroui/react';
import { motion, AnimatePresence } from 'framer-motion';
import Icon from './Icon';

interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export default function Drawer({ open, onClose, title, children }: DrawerProps) {
  const panelRef = useRef<HTMLElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (open) {
      previousFocusRef.current = document.activeElement as HTMLElement;

      const handleEsc = (e: KeyboardEvent) => {
        if (e.key === 'Escape') {
          onClose();
          return;
        }

        if (e.key !== 'Tab' || !panelRef.current) return;

        const focusable = Array.from(
          panelRef.current.querySelectorAll<HTMLElement>(
            'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
          )
        ).filter((el) => !el.hasAttribute('aria-hidden'));

        if (focusable.length === 0) return;

        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      };

      document.addEventListener('keydown', handleEsc);
      document.body.style.overflow = 'hidden';

      const id = window.requestAnimationFrame(() => {
        panelRef.current?.focus();
      });

      return () => {
        window.cancelAnimationFrame(id);
        document.removeEventListener('keydown', handleEsc);
        document.body.style.overflow = '';

        if (previousFocusRef.current && document.contains(previousFocusRef.current)) {
          previousFocusRef.current.focus();
        }
      };
    }
  }, [open, onClose]);

  return (
    <AnimatePresence>
      {open && (
        <>
          {/* Scrim */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 z-90 bg-black/35"
          />

          {/* Panel */}
          <motion.aside
            ref={panelRef}
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 30, stiffness: 300 }}
            role="dialog"
            aria-modal="true"
            aria-label={title || 'Detail panel'}
            tabIndex={-1}
            className="fixed top-0 right-0 z-100 h-full w-full md:w-100 md:min-w-90 md:max-w-110 flex flex-col desk-inspector rounded-none"
          >
            {/* Header */}
            <div className="desk-inspector-header px-5 py-4">
              <h2 className="md-typescale-title-medium" style={{ color: 'var(--desk-text-primary)' }}>
                {title || 'Details'}
              </h2>
              <Button
                variant="ghost"
                isIconOnly
                onPress={onClose}
                aria-label="Close"
                className="desk-btn-ghost w-9 h-9 p-0"
              >
                <Icon name="close" className="w-5 h-5" />
              </Button>
            </div>

            {/* Content */}
            <div className="desk-inspector-body md-typescale-body-medium" style={{ color: 'var(--desk-text-secondary)' }}>
              {children}
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  );
}
