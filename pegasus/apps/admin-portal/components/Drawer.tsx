'use client';

import React, { useEffect, useRef, useCallback } from 'react';
import { Button } from '@heroui/react';
import Icon from './Icon';

interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export default function Drawer({ open, onClose, title, children }: DrawerProps) {
  const panelRef = useRef<HTMLElement>(null);

  const handleEsc = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
  }, [onClose]);

  useEffect(() => {
    if (open) {
      document.addEventListener('keydown', handleEsc);
      return () => document.removeEventListener('keydown', handleEsc);
    }
  }, [open, handleEsc]);

  return (
    <>
      {/* Scrim */}
      <div
        onClick={onClose}
        className="fixed inset-0 z-40 transition-opacity duration-300"
        style={{
          background: 'var(--backdrop)',
          opacity: open ? 1 : 0,
          pointerEvents: open ? 'auto' : 'none',
        }}
      />

      {/* Panel */}
      <aside
        ref={panelRef}
        aria-label={title || 'Detail panel'}
        className="fixed top-0 right-0 z-50 h-full w-full md:w-120 flex flex-col transition-transform duration-300 ease-out bg-surface"
        style={{
          transform: open ? 'translateX(0)' : 'translateX(100%)',
          borderLeft: '1px solid var(--border)',
        }}
      >
        {/* Pinned header */}
        <div className="flex items-center justify-between px-6 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
          <h2 className="md-typescale-title-large text-foreground">{title}</h2>
          <Button
            variant="ghost"
            isIconOnly
            onPress={onClose}
            aria-label="Close"
            className="w-10 h-10 min-w-0 text-muted"
          >
            <Icon name="close" className="w-5 h-5" />
          </Button>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto">
          {children}
        </div>
      </aside>
    </>
  );
}
