export type CookieCategory = 'necessary' | 'functional' | 'analytics' | 'marketing';

export interface ConsentCategories {
  necessary: boolean; // Always true (Strictly Necessary)
  functional: boolean;
  analytics: boolean;
  marketing: boolean;
}

export type JurisdictionCode = 'GDPR' | 'CCPA' | 'UZBEKISTAN' | 'GLOBAL';

export interface CookieConsentRecord {
  version: string;
  consentId: string;
  timestamp: string;
  categories: ConsentCategories;
  jurisdiction: JurisdictionCode;
  gpcApplied: boolean;
}

export type CookieParty = 'First-Party' | 'Third-Party';

export interface CookieInventoryItem {
  name: string;
  category: CookieCategory;
  provider: string;
  party: CookieParty;
  domain: string;
  expiryEn: string;
  expiryRu: string;
  purposeEn: string;
  purposeRu: string;
  retentionType: 'Session' | 'Persistent';
}

export const CONSENT_STORAGE_KEY = 'pegasus_cookie_consent_v1';
export const CONSENT_COOKIE_NAME = 'pegasus_consent';
export const CURRENT_POLICY_VERSION = '2026.1';
