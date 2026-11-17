export interface AuthCountry {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}

export const AUTH_COUNTRIES: AuthCountry[] = [
  { code: "UZ", name: "Uzbekistan", dialCode: "+998", currency: "UZS" },
  { code: "KZ", name: "Kazakhstan", dialCode: "+7", currency: "KZT" },
  { code: "KG", name: "Kyrgyzstan", dialCode: "+996", currency: "KGS" },
  { code: "TJ", name: "Tajikistan", dialCode: "+992", currency: "TJS" },
  { code: "TM", name: "Turkmenistan", dialCode: "+993", currency: "TMT" },
  { code: "AE", name: "United Arab Emirates", dialCode: "+971", currency: "AED" },
  { code: "TR", name: "Türkiye", dialCode: "+90", currency: "TRY" },
  { code: "RU", name: "Russia", dialCode: "+7", currency: "RUB" },
  { code: "US", name: "United States", dialCode: "+1", currency: "USD" },
  { code: "GB", name: "United Kingdom", dialCode: "+44", currency: "GBP" },
];

export function dialCodeForCountry(code: string): string {
  return AUTH_COUNTRIES.find((c) => c.code === code)?.dialCode ?? "";
}
