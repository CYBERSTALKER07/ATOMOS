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
        <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/60 backdrop-blur-md"
            onClick={onClose}
          />
          <motion.div
            ref={dialogRef}
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="glass-premium w-full max-w-lg rounded-3xl overflow-hidden relative z-10"
            role="dialog"
            aria-modal="true"
            aria-labelledby="dialog-title"
            tabIndex={-1}
          >
            <div className="px-8 py-6 border-b border-white/10 flex items-center justify-between bg-white/[0.03]">
              <h2 id="dialog-title" className="text-2xl font-bold tracking-tight text-white">{title}</h2>
              <button 
                onClick={onClose}
                className="w-10 h-10 rounded-full flex items-center justify-center hover:bg-white/10 transition-colors active-press"
              >
                <span className="material-symbols-outlined text-white/60">close</span>
              </button>
            </div>
            
            <div className="px-8 py-8 md-typescale-body-large text-white/80 leading-relaxed">
              {children}
            </div>
            
            {actions && (
              <div className="px-8 py-6 border-t border-white/10 bg-white/[0.03] flex items-center justify-end gap-3">
                {actions}
              </div>
            )}
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
