'use client';

import React, { useCallback, useState } from 'react';
import { useRouter } from 'next/navigation';
import { SupplierShiftProvider, useSupplierShift } from '@/hooks/useSupplierShift';
import { firebaseSignOut } from '../../lib/firebase';

function clearAuthCookies() {
  document.cookie = 'admin_jwt=; path=/; max-age=0; SameSite=Lax';
  document.cookie = 'supplier_jwt=; path=/; max-age=0; SameSite=Lax';
  document.cookie = 'supplier_name=; path=/; max-age=0; SameSite=Lax';
}

function SupplierLayoutInner({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { isActive, toggleShift } = useSupplierShift();
  const [isToggling, setIsToggling] = useState(false);

  const handleLogout = useCallback(() => {
    clearAuthCookies();
    firebaseSignOut().catch(() => {});
    router.push('/auth/login');
  }, [router]);

  const handleShiftToggle = useCallback(async () => {
    if (isToggling) return;
    setIsToggling(true);
    try { await toggleShift(); } finally { setIsToggling(false); }
  }, [isToggling, toggleShift]);

  return (
    <div className="min-h-full relative">
      {/* Top-right controls */}
      <div className="absolute top-4 right-4 z-50 flex items-center gap-2">
        {/* Shift Status Toggle */}
        {isActive !== null && (
          <button
            onClick={handleShiftToggle}
            disabled={isToggling}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full md-typescale-label-small transition-all cursor-pointer select-none"
            style={{
              background: isActive
                ? 'color-mix(in srgb, var(--success) 15%, transparent)'
                : 'color-mix(in srgb, var(--danger) 15%, transparent)',
              color: isActive ? 'var(--success)' : 'var(--danger)',
              border: `1px solid ${isActive ? 'color-mix(in srgb, var(--success) 30%, transparent)' : 'color-mix(in srgb, var(--danger) 30%, transparent)'}`,
              opacity: isToggling ? 0.6 : 1,
            }}
            title={isActive ? 'Click to go off-shift' : 'Click to go on-shift'}
          >
            <span
              className="inline-block rounded-full"
              style={{
                width: 8,
                height: 8,
                background: isActive ? 'var(--success)' : 'var(--danger)',
              }}
            />
            {isActive ? 'ON SHIFT' : 'OFF SHIFT'}
          </button>
        )}

        {/* Sign Out */}
        <button
          onClick={handleLogout}
          className="px-3 py-1.5 rounded-full md-typescale-label-small transition-colors cursor-pointer"
          style={{ background: 'var(--surface)', color: 'var(--muted)' }}
        >
          Sign Out
        </button>
      </div>
      {children}
    </div>
  );
}

export default function SupplierLayout({ children }: { children: React.ReactNode }) {
  return (
    <SupplierShiftProvider>
      <SupplierLayoutInner>{children}</SupplierLayoutInner>
    </SupplierShiftProvider>
  );
}
