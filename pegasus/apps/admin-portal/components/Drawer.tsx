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

  useEffect(() => {
    if (open) {
      const handleEsc = (e: KeyboardEvent) => {
        if (e.key === 'Escape') onClose();
      };
      document.addEventListener('keydown', handleEsc);
      document.body.style.overflow = 'hidden';
      return () => {
        document.removeEventListener('keydown', handleEsc);
        document.body.style.overflow = '';
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
            className="fixed inset-0 z-[90] bg-black/60 backdrop-blur-sm"
          />

          {/* Panel */}
          <motion.aside
            ref={panelRef}
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 30, stiffness: 300 }}
            aria-label={title || 'Detail panel'}
            className="fixed top-0 right-0 z-[100] h-full w-full md:w-[480px] flex flex-col glass-premium border-l border-white/10"
          >
            {/* Header */}
            <div className="flex items-center justify-between px-8 py-6 border-b border-white/10 bg-white/[0.03]">
              <h2 className="text-2xl font-bold tracking-tight text-white">{title}</h2>
              <Button
                variant="ghost"
                isIconOnly
                onPress={onClose}
                aria-label="Close"
                className="w-12 h-12 rounded-full text-white/60 hover:bg-white/10 transition-colors active-press"
              >
                <Icon name="close" className="w-6 h-6" />
              </Button>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-y-auto px-8 py-8 md-typescale-body-large text-white/80 leading-relaxed custom-scrollbar">
              {children}
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  );
}
