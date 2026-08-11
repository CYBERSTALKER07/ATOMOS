import { useEffect, useState } from 'react';

import { API_BASE } from '../api';
import {
  clearPayloaderSession,
  savePayloaderSession,
  setTokenRefreshListener,
} from '../authSession';
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from '../firebaseAuth';
import { extractProblemMessage } from '../localization';
import { registerPayloadPushTokens } from '../pushRegistration';
import type { Locale } from '../../../packages/i18n/locales';
import type { ShowToast } from './useToast';

// ─── Payloader auth (OTP / PIN login, logout, token refresh) ──────────────────

export function usePayloaderAuth({
  locale,
  tx,
  showToast,
}: {
  locale: Locale;
  tx: (key: string) => string;
  showToast: ShowToast;
}) {
  // Auth state
  const [token, setToken] = useState<string | null>(null);
  const [workerName, setWorkerName] = useState('');
  const [phoneInput, setPhoneInput] = useState('');
  const [pinInput, setPinInput] = useState('');
  const [otpInput, setOtpInput] = useState('');
  const [loginMode, setLoginMode] = useState<'otp' | 'pin'>('otp');
  const [otpSent, setOtpSent] = useState(false);
  const [isLoggingIn, setIsLoggingIn] = useState(false);
  const [authLoading, setAuthLoading] = useState(true);

  // Supplier context
  const [supplierId, setSupplierId] = useState<string | null>(null);

  // Keep the app-level token in sync with silent refreshes from authSession
  useEffect(() => {
    setTokenRefreshListener(setToken);
    return () => setTokenRefreshListener(null);
  }, []);

  // ── Payloader Login ──────────────────────────────────────────────────────
  const completeLogin = async (data: Record<string, unknown>) => {
    await savePayloaderSession(data as Parameters<typeof savePayloaderSession>[0]);
    setToken(String(data.token ?? ''));
    setWorkerName(String(data.name ?? 'Payloader'));
    if (data.supplier_id) setSupplierId(String(data.supplier_id));
    void registerPayloadPushTokens();
  };

  const handleLoginPin = async () => {
    if (!phoneInput || !pinInput) return;
    setIsLoggingIn(true);
    try {
      const res = await fetch(`${API_BASE}/v1/auth/payloader/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone: phoneInput, pin: pinInput }),
      });
      if (!res.ok) {
        throw new Error(await extractProblemMessage(res, locale));
      }
      const data = await res.json();
      await completeLogin(data);
    } catch (e: unknown) {
      showToast(tx('auth.error.login_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleSendOtp = async () => {
    if (!phoneInput.trim()) return;
    setIsLoggingIn(true);
    try {
      await sendPhoneOtp(phoneInput.trim());
      setOtpSent(true);
    } catch (e: unknown) {
      showToast(tx('auth.error.login_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleVerifyOtp = async () => {
    if (otpInput.trim().length < 6) return;
    setIsLoggingIn(true);
    try {
      const idToken = await verifyPhoneOtp(otpInput.trim());
      const res = await fetch(`${API_BASE}/v1/auth/payloader/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id_token: idToken }),
      });
      if (!res.ok) {
        throw new Error(await extractProblemMessage(res, locale));
      }
      const data = await res.json();
      await completeLogin(data);
      setOtpSent(false);
      setOtpInput('');
      resetPhoneOtpFlow();
    } catch (e: unknown) {
      showToast(tx('auth.error.login_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleLogout = async () => {
    await clearPayloaderSession();
    setToken(null);
    setWorkerName('');
    setSupplierId(null);
  };

  const handleToggleLoginMode = () => {
    setLoginMode(loginMode === 'otp' ? 'pin' : 'otp');
    setOtpSent(false);
    setOtpInput('');
    resetPhoneOtpFlow();
  };

  return {
    token,
    setToken,
    workerName,
    setWorkerName,
    supplierId,
    setSupplierId,
    authLoading,
    setAuthLoading,
    phoneInput,
    setPhoneInput,
    pinInput,
    setPinInput,
    otpInput,
    setOtpInput,
    loginMode,
    otpSent,
    isLoggingIn,
    completeLogin,
    handleLoginPin,
    handleSendOtp,
    handleVerifyOtp,
    handleLogout,
    handleToggleLoginMode,
  };
}
