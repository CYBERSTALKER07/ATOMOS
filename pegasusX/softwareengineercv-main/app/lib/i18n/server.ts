import { cookies } from 'next/headers';
import type { Language } from './translations';

export async function getServerLanguage(): Promise<Language> {
  const cookieStore = await cookies();
  const lang = cookieStore.get('pegasus_lang')?.value;
  if (lang === 'en' || lang === 'ru') {
    return lang as Language;
  }
  return 'en';
}
