'use client';

import React, { createContext, useContext, useEffect, useState, useMemo, useCallback } from 'react';
import {
 ConsentCategories,
 CookieCategory,
 CookieConsentRecord,
} from '../lib/cookies/cookieTypes';
import {
 getStoredConsent,
 saveConsentRecord,
 resetConsentRecord,
 ACCEPT_ALL_CONSENT,
 REJECT_NON_ESSENTIAL_CONSENT,
 isGPCActive,
} from '../lib/cookies/cookieRegistry';

interface CookieConsentContextType {
 consent: CookieConsentRecord | null;
 isBannerOpen: boolean;
 isModalOpen: boolean;
 isGPC: boolean;
 hasConsent: (category: CookieCategory) => boolean;
 openModal: () => void;
 closeModal: () => void;
 acceptAll: () => void;
 rejectNonEssential: () => void;
 savePreferences: (categories: ConsentCategories) => void;
 resetConsent: () => void;
}

const CookieConsentContext = createContext<CookieConsentContextType | undefined>(undefined);

export function CookieConsentProvider({ children }: { children: React.ReactNode }) {
 const [consent, setConsent] = useState<CookieConsentRecord | null>(null);
 const [isBannerOpen, setIsBannerOpen] = useState(false);
 const [isModalOpen, setIsModalOpen] = useState(false);
 const [isGPC, setIsGPC] = useState(false);
 const [mounted, setMounted] = useState(false);

 // Initialize consent on mount
 useEffect(() => {
 setMounted(true);
 const gpcActive = isGPCActive();
 setIsGPC(gpcActive);

 const stored = getStoredConsent();
 if (stored) {
 setConsent(stored);
 setIsBannerOpen(false);
 } else {
 // First visit: show tactical bottom consent banner
 setIsBannerOpen(true);
 }
 }, []);

 const openModal = useCallback(() => {
 setIsModalOpen(true);
 }, []);

 const closeModal = useCallback(() => {
 setIsModalOpen(false);
 }, []);

 const acceptAll = useCallback(() => {
 const record = saveConsentRecord(ACCEPT_ALL_CONSENT);
 setConsent(record);
 setIsBannerOpen(false);
 setIsModalOpen(false);
 }, []);

 const rejectNonEssential = useCallback(() => {
 const record = saveConsentRecord(REJECT_NON_ESSENTIAL_CONSENT);
 setConsent(record);
 setIsBannerOpen(false);
 setIsModalOpen(false);
 }, []);

 const savePreferences = useCallback((categories: ConsentCategories) => {
 const record = saveConsentRecord(categories);
 setConsent(record);
 setIsBannerOpen(false);
 setIsModalOpen(false);
 }, []);

 const resetConsent = useCallback(() => {
 resetConsentRecord();
 setConsent(null);
 setIsBannerOpen(true);
 setIsModalOpen(false);
 }, []);

 const hasConsent = useCallback(
 (category: CookieCategory): boolean => {
 if (category === 'necessary') return true;
 if (!consent) return false;
 return Boolean(consent.categories[category]);
 },
 [consent]
 );

 const value = useMemo(
 () => ({
 consent,
 isBannerOpen: mounted ? isBannerOpen : false,
 isModalOpen: mounted ? isModalOpen : false,
 isGPC,
 hasConsent,
 openModal,
 closeModal,
 acceptAll,
 rejectNonEssential,
 savePreferences,
 resetConsent,
 }),
 [
 consent,
 isBannerOpen,
 isModalOpen,
 isGPC,
 mounted,
 hasConsent,
 openModal,
 closeModal,
 acceptAll,
 rejectNonEssential,
 savePreferences,
 resetConsent,
 ]
 );

 return (
 <CookieConsentContext.Provider value={value}>
 {children}
 </CookieConsentContext.Provider>
 );
}

export function useCookieConsent() {
 const context = useContext(CookieConsentContext);
 if (!context) {
 return {
 consent: null,
 isBannerOpen: false,
 isModalOpen: false,
 isGPC: false,
 hasConsent: (cat: CookieCategory) => cat === 'necessary',
 openModal: () => {},
 closeModal: () => {},
 acceptAll: () => {},
 rejectNonEssential: () => {},
 savePreferences: () => {},
 resetConsent: () => {},
 };
 }
 return context;
}
