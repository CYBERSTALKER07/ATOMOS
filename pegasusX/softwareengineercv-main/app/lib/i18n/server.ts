import { cookies } from 'next/headers';
import type { Language } from './translations';

export const SUPPORTED_LANGUAGES: Language[] = [
  'en',
  'ru',
  'es',
  'de',
  'fr',
  'zh',
  'ja',
  'ar',
  'pt',
  'tr',
  'uz',
];

export async function getServerLanguage(): Promise<Language> {
  const cookieStore = await cookies();
  const lang = cookieStore.get('pegasus_lang')?.value as Language | undefined;
  if (lang && SUPPORTED_LANGUAGES.includes(lang)) {
    return lang;
  }
  return 'en';
}
