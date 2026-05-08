'use client';

import { useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  actions?: React.ReactNode;
}

export default function Dialog({ open, onClose, title, children, actions }: DialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handler);
    dialogRef.current?.focus();
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handler);
      document.body.style.overflow = '';
    };
  }, [open, onClose]);

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-100 flex items-center justify-center p-4">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0"
            style={{ background: 'color-mix(in srgb, var(--desk-text-primary) 40%, transparent)' }}
            onClick={onClose}
          />
          <motion.div
            ref={dialogRef}
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="w-full max-w-lg overflow-hidden relative z-10"
            style={{
              background: 'var(--desk-surface)',
              border: '1px solid var(--desk-border)',
              borderRadius: 'var(--radius-lg)',
              boxShadow: '0 16px 36px rgba(15, 23, 42, 0.18)',
            }}
            role="dialog"
            aria-modal="true"
            aria-labelledby="dialog-title"
            tabIndex={-1}
          >
            <div
              className="px-8 py-6 border-b flex items-center justify-between"
              style={{ borderColor: 'var(--desk-border)', background: 'var(--desk-surface-subtle)' }}
            >
              <h2 id="dialog-title" className="text-2xl font-semibold tracking-tight" style={{ color: 'var(--desk-text-primary)' }}>{title}</h2>
              <button 
                onClick={onClose}
                className="w-10 h-10 rounded-full flex items-center justify-center transition-colors active-press"
                style={{ color: 'var(--desk-text-secondary)' }}
              >
                <span className="material-symbols-outlined">close</span>
              </button>
            </div>
            
            <div className="px-8 py-8 md-typescale-body-large leading-relaxed" style={{ color: 'var(--desk-text-secondary)' }}>
              {children}
            </div>
            
            {actions && (
              <div
                className="px-8 py-6 border-t flex items-center justify-end gap-3"
                style={{ borderColor: 'var(--desk-border)', background: 'var(--desk-surface-subtle)' }}
              >
                {actions}
              </div>
            )}
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
