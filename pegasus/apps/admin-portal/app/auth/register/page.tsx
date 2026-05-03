'use client';

import { useState, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import dynamic from 'next/dynamic';
import { Button } from '@heroui/react';
import { translateProblemDetail } from '@pegasus/i18n';
import { useLocale } from '@/hooks/useLocale';
import { exchangeCustomToken } from '../../../lib/firebase';
import { isTauri, storeToken } from '../../../lib/bridge';
import { COUNTRIES, DEFAULT_COUNTRY, findCountry, dialingPrefix } from '../../../lib/constants/countries';

function LocationPickerLoading() {
  const { t } = useLocale();

  return (
    <div
      className="w-full h-72 flex items-center justify-center"
      style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 12 }}
    >
      <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
        {t('supplier_portal.auth.register.step2.map_loading')}
      </span>
    </div>
  );
}

const LocationPicker = dynamic(
  () => import('../../../components/location-picker/location-picker'),
  { ssr: false, loading: () => <LocationPickerLoading /> }
);

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// ─── All canonical categories — Walmart-scale shelf coverage ─────────────────
const CATEGORY_IDS = [
  'cat-water',
  'cat-sparkling-water',
  'cat-soft-drinks',
  'cat-juice',
  'cat-tea-coffee',
  'cat-energy-sports',
  'cat-powdered-drinks',
  'cat-kombucha',
  'cat-plant-milk',
  'cat-milk',
  'cat-yogurt',
  'cat-cheese',
  'cat-butter-margarine',
  'cat-eggs',
  'cat-deli-meats',
  'cat-hummus-dips',
  'cat-tofu-tempeh',
  'cat-fresh-pasta-sauces',
  'cat-bread',
  'cat-tortillas-wraps',
  'cat-bagels-english-muffins',
  'cat-cakes-pastries',
  'cat-cookies-brownies',
  'cat-pita-naan',
  'cat-baking-mixes',
  'cat-cereal',
  'cat-oatmeal-porridge',
  'cat-granola-bars',
  'cat-pancake-waffle',
  'cat-syrup-honey',
  'cat-rice',
  'cat-pasta-noodles',
  'cat-flour-baking',
  'cat-sugar-sweeteners',
  'cat-cooking-oils',
  'cat-vinegar',
  'cat-canned-vegetables',
  'cat-canned-fruit',
  'cat-canned-beans-legumes',
  'cat-canned-meat-fish',
  'cat-canned-soup',
  'cat-tomato-products',
  'cat-condiments',
  'cat-mayonnaise-dressings',
  'cat-mustard-hot-sauce',
  'cat-soy-asian-sauces',
  'cat-bbq-marinades',
  'cat-spices-herbs',
  'cat-salt-pepper',
  'cat-dried-beans-lentils',
  'cat-grains-couscous',
  'cat-chips-crisps',
  'cat-popcorn',
  'cat-pretzels-crackers',
  'cat-nuts-seeds',
  'cat-dried-fruit',
  'cat-jerky-meat-snacks',
  'cat-protein-bars',
  'cat-rice-cakes',
  'cat-chocolate',
  'cat-gummy-candy',
  'cat-hard-candy-mints',
  'cat-chewing-gum',
  'cat-biscuits-cookies',
  'cat-ice-cream-toppings',
  'cat-fresh-fruit',
  'cat-fresh-vegetables',
  'cat-herbs-salads',
  'cat-organic-produce',
  'cat-mushrooms',
  'cat-potatoes-onions',
  'cat-beef',
  'cat-chicken-turkey',
  'cat-pork',
  'cat-lamb',
  'cat-ground-meat',
  'cat-sausages-hotdogs',
  'cat-fresh-fish',
  'cat-shrimp-shellfish',
  'cat-smoked-fish',
  'cat-frozen-meals',
  'cat-frozen-pizza',
  'cat-frozen-vegetables',
  'cat-frozen-fruit',
  'cat-frozen-meat-seafood',
  'cat-frozen-snacks',
  'cat-frozen-breakfast',
  'cat-ice-cream',
  'cat-frozen-fries-potatoes',
  'cat-mexican-food',
  'cat-asian-food',
  'cat-indian-food',
  'cat-middle-eastern-food',
  'cat-italian-food',
  'cat-kosher',
  'cat-halal',
  'cat-gluten-free',
  'cat-vegan-plant-based',
  'cat-organic-natural',
  'cat-baby-formula',
  'cat-baby-food',
  'cat-diapers',
  'cat-baby-bath-skin',
  'cat-baby-feeding',
  'cat-medicine-otc',
  'cat-cold-flu',
  'cat-digestive-health',
  'cat-first-aid',
  'cat-vitamins',
  'cat-protein-powder',
  'cat-eye-ear-care',
  'cat-diabetes-care',
  'cat-mobility-aids',
  'cat-shampoo-conditioner',
  'cat-hair-styling',
  'cat-hair-color',
  'cat-body-wash-soap',
  'cat-deodorant',
  'cat-lotion-moisturizer',
  'cat-sunscreen',
  'cat-face-care',
  'cat-lip-care',
  'cat-oral-care',
  'cat-mouthwash',
  'cat-shaving',
  'cat-mens-grooming',
  'cat-feminine-care',
  'cat-cotton-pads',
  'cat-cosmetics-face',
  'cat-cosmetics-eyes',
  'cat-cosmetics-lips',
  'cat-nail-care',
  'cat-fragrance',
  'cat-hair-tools',
  'cat-all-purpose-cleaners',
  'cat-bathroom-cleaners',
  'cat-kitchen-cleaners',
  'cat-glass-cleaners',
  'cat-floor-care',
  'cat-disinfectants',
  'cat-laundry-detergent',
  'cat-fabric-softener',
  'cat-stain-removers',
  'cat-dishwashing',
  'cat-trash-bags',
  'cat-sponges-cloths',
  'cat-air-fresheners',
  'cat-pest-control',
  'cat-toilet-paper',
  'cat-paper-towels',
  'cat-facial-tissue',
  'cat-napkins',
  'cat-plates-cups-disposable',
  'cat-food-wrap-bags',
  'cat-cookware',
  'cat-bakeware',
  'cat-kitchen-utensils',
  'cat-knives-cutting',
  'cat-food-storage',
  'cat-water-bottles',
  'cat-dinnerware',
  'cat-small-appliances',
  'cat-bedding',
  'cat-pillows-blankets',
  'cat-towels',
  'cat-curtains-blinds',
  'cat-home-decor',
  'cat-candles-home-fragrance',
  'cat-storage-organization',
  'cat-hangers-laundry-supplies',
  'cat-light-bulbs',
  'cat-batteries',
  'cat-extension-cords',
  'cat-plants-seeds',
  'cat-soil-fertilizer',
  'cat-garden-tools',
  'cat-outdoor-furniture',
  'cat-grills-charcoal',
  'cat-dog-food',
  'cat-cat-food',
  'cat-pet-treats',
  'cat-pet-toys',
  'cat-pet-grooming',
  'cat-litter-waste',
  'cat-pet-beds-carriers',
  'cat-pens-pencils',
  'cat-notebooks-paper',
  'cat-binders-folders',
  'cat-tape-glue-scissors',
  'cat-printer-ink',
  'cat-backpacks-bags',
  'cat-action-figures',
  'cat-building-sets',
  'cat-board-games-puzzles',
  'cat-outdoor-play',
  'cat-arts-crafts',
  'cat-stuffed-animals',
  'cat-phone-accessories',
  'cat-chargers-cables',
  'cat-headphones-earbuds',
  'cat-memory-cards-usb',
  'cat-smart-home',
  'cat-motor-oil',
  'cat-car-cleaning',
  'cat-car-fresheners',
  'cat-car-accessories',
  'cat-party-supplies',
  'cat-gift-wrap',
  'cat-greeting-cards',
  'cat-seasonal',
  'cat-tobacco',
  'cat-lighters-matches',
  'cat-other',
] as const;

const STEPS = ['account', 'location', 'business', 'categories'] as const;

const STEP_ICONS: Record<number, string> = {
  0: 'M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z',
  1: 'M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z',
  2: 'M12 7V3H2v18h20V7H12zM6 19H4v-2h2v2zm0-4H4v-2h2v2zm0-4H4V9h2v2zm0-4H4V5h2v2zm4 12H8v-2h2v2zm0-4H8v-2h2v2zm0-4H8V9h2v2zm0-4H8V5h2v2zm10 12h-8v-2h2v-2h-2v-2h2v-2h-2V9h8v10zm-2-8h-2v2h2v-2zm0 4h-2v2h2v-2z',
  3: 'M4 10v7h3v-7H4zm6 0v7h3v-7h-3zM2 22h19v-3H2v3zm14-12v7h3v-7h-3zm-4.5-9L2 6v2h19V6l-9.5-5z',
};

// ─── Shared Input Component ───────────────────────────────────────────────────
function InputField({ label, ...props }: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  const { t } = useLocale();
  const [showPw, setShowPw] = useState(false);
  const isPassword = props.type === 'password';

  return (
    <div>
      <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>{label}</label>
      <div className="relative">
        <input {...props} type={isPassword && showPw ? 'text' : props.type} className={`md-input-outlined w-full${isPassword ? ' pr-12' : ''}`} />
        {isPassword && (
          <button
            type="button"
            onClick={() => setShowPw(v => !v)}
            className="absolute right-3 top-1/2 -translate-y-1/2 p-1 rounded-full"
            style={{ color: 'var(--muted)' }}
            aria-label={showPw ? t('common.action.hide_password') : t('common.action.show_password')}
          >
            {showPw ? (
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78 3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z"/></svg>
            ) : (
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/></svg>
            )}
          </button>
        )}
      </div>
    </div>
  );
}

// ─── Step 1: Account ──────────────────────────────────────────────────────────
function Step1({
  data, onChange, country, onCountryChange,
}: {
  data: { companyName: string; contactPerson: string; email: string; phone: string; password: string; confirmPassword: string };
  onChange: (k: string, v: string) => void;
  country: string;
  onCountryChange: (code: string) => void;
}) {
  const { t } = useLocale();
  const [countrySearch, setCountrySearch] = useState('');
  const [countryOpen, setCountryOpen] = useState(false);
  const selected = findCountry(country);
  const prefix = selected?.code ?? '+998';

  const filteredCountries = useMemo(() => {
    if (!countrySearch) return COUNTRIES;
    const q = countrySearch.toLowerCase();
    return COUNTRIES.filter(c => c.label.toLowerCase().includes(q) || c.value.toLowerCase().includes(q) || c.code.includes(q));
  }, [countrySearch]);

  return (
    <div className="space-y-4">
      {/* Country Selector */}
      <div>
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>
          {t('supplier_portal.auth.register.step1.country_label')}
        </label>
        <div className="relative">
          <button
            type="button"
            onClick={() => setCountryOpen(v => !v)}
            className="md-input-outlined w-full text-left flex items-center justify-between"
          >
            <span>{selected ? `${selected.label} (${selected.code})` : t('supplier_portal.auth.register.step1.country_placeholder')}</span>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--muted)' }}>
              <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6z"/>
            </svg>
          </button>
          {countryOpen && (
            <div className="absolute z-50 mt-1 w-full max-h-60 overflow-auto md-shape-md" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
              <div className="sticky top-0 p-2" style={{ background: 'var(--surface)' }}>
                <input
                  type="text"
                  value={countrySearch}
                  onChange={e => setCountrySearch(e.target.value)}
                  placeholder={t('supplier_portal.auth.register.step1.search_countries')}
                  className="md-input-outlined w-full"
                  autoFocus
                />
              </div>
              {filteredCountries.map(c => (
                <button
                  key={c.value}
                  type="button"
                  className="w-full text-left px-3 py-2 md-typescale-body-small hover:opacity-80 flex items-center justify-between"
                  style={{ background: c.value === country ? 'var(--accent-soft)' : 'var(--surface)', color: 'var(--foreground)' }}
                  onClick={() => { onCountryChange(c.value); setCountryOpen(false); setCountrySearch(''); }}
                >
                  <span>{c.label}</span>
                  <span style={{ color: 'var(--muted)' }}>{c.code}</span>
                </button>
              ))}
              {filteredCountries.length === 0 && (
                <p className="px-3 py-4 md-typescale-label-small text-center" style={{ color: 'var(--muted)' }}>
                  {t('supplier_portal.auth.register.step1.no_countries')}
                </p>
              )}
            </div>
          )}
        </div>
      </div>

      <InputField label={t('supplier_portal.auth.register.step1.company_name_label')} type="text" value={data.companyName} onChange={e => onChange('companyName', e.target.value)} placeholder={t('supplier_portal.auth.register.step1.company_name_placeholder')} required autoFocus />
      <InputField label={t('supplier_portal.auth.register.step1.contact_person_label')} type="text" value={data.contactPerson} onChange={e => onChange('contactPerson', e.target.value)} placeholder={t('supplier_portal.auth.register.step1.contact_person_placeholder')} required />
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <InputField label={t('supplier_portal.auth.register.step1.email_label')} type="email" value={data.email} onChange={e => onChange('email', e.target.value)} placeholder={t('supplier_portal.auth.register.step1.email_placeholder')} required />
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>
            {t('supplier_portal.auth.register.step1.phone_label')}
          </label>
          <div className="md-input-outlined flex items-center !p-0 overflow-hidden w-full focus-within:!border-[var(--color-md-on-surface)]">
            <span className="flex items-center justify-center px-3 h-full shrink-0 md-typescale-body-small border-r" style={{ color: 'var(--muted)', borderColor: 'var(--border)', minWidth: '64px' }}>{prefix}</span>
            <input type="tel" value={data.phone} onChange={e => onChange('phone', e.target.value)} placeholder={t('supplier_portal.auth.register.step1.phone_placeholder')} required className="w-full h-full bg-transparent outline-none px-3 md-typescale-body-small" style={{ color: 'var(--foreground)' }} />
          </div>
        </div>
      </div>
      <InputField label={t('supplier_portal.auth.register.step1.password_label')} type="password" value={data.password} onChange={e => onChange('password', e.target.value)} placeholder={t('supplier_portal.auth.register.step1.password_placeholder')} required />
      <InputField label={t('supplier_portal.auth.register.step1.confirm_password_label')} type="password" value={data.confirmPassword} onChange={e => onChange('confirmPassword', e.target.value)} placeholder={t('supplier_portal.auth.register.step1.confirm_password_placeholder')} required />
    </div>
  );
}

// ─── Step 2: Location ─────────────────────────────────────────────────────────
function Step2({
  data, onChange,
}: {
  data: { warehouseAddress: string; warehouseLat: string; warehouseLng: string; billingAddress: string };
  onChange: (k: string, v: string) => void;
}) {
  const { t } = useLocale();
  const handleLocationChange = useCallback((lat: string, lng: string, address: string) => {
    onChange('warehouseLat', lat);
    onChange('warehouseLng', lng);
    onChange('warehouseAddress', address);
  }, [onChange]);

  return (
    <div className="space-y-4">
      <div>
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>
          {t('supplier_portal.auth.register.step2.warehouse_location_label')}
        </label>
        <LocationPicker
          lat={data.warehouseLat}
          lng={data.warehouseLng}
          address={data.warehouseAddress}
          onLocationChange={handleLocationChange}
        />
      </div>
      <InputField label={t('supplier_portal.auth.register.step2.billing_address_label')} type="text" value={data.billingAddress} onChange={e => onChange('billingAddress', e.target.value)} placeholder={t('supplier_portal.auth.register.step2.billing_address_placeholder')} />
    </div>
  );
}

// ─── Step 3: Business Details ─────────────────────────────────────────────────
function Step3({
  data, onChange, coldChain, onColdChainToggle, palletization, onPalletization,
}: {
  data: { taxId: string; companyRegNumber: string };
  onChange: (k: string, v: string) => void;
  coldChain: boolean;
  onColdChainToggle: () => void;
  palletization: string;
  onPalletization: (v: string) => void;
}) {
  const { t } = useLocale();
  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <InputField label={t('supplier_portal.auth.register.step3.tax_id_label')} type="text" value={data.taxId} onChange={e => onChange('taxId', e.target.value)} placeholder={t('supplier_portal.auth.register.step3.tax_id_placeholder')} required />
        <InputField label={t('supplier_portal.auth.register.step3.company_reg_number_label')} type="text" value={data.companyRegNumber} onChange={e => onChange('companyRegNumber', e.target.value)} placeholder={t('supplier_portal.auth.register.step3.company_reg_number_placeholder')} />
      </div>

      {/* Fleet Operations */}
      <div className="p-4 md-shape-md" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
        <p className="md-typescale-label-medium mb-3" style={{ color: 'var(--foreground)' }}>
          {t('supplier_portal.auth.register.step3.fleet_profile_title')}
        </p>
        <div className="flex items-center justify-between mb-3">
          <div>
            <p className="md-typescale-body-small" style={{ color: 'var(--foreground)' }}>
              {t('supplier_portal.auth.register.step3.cold_chain_title')}
            </p>
            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
              {t('supplier_portal.auth.register.step3.cold_chain_description')}
            </p>
          </div>
          <button
            type="button"
            onClick={onColdChainToggle}
            className="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
            style={{ background: coldChain ? 'var(--accent)' : 'var(--border)' }}
            aria-pressed={coldChain}
          >
            <span
              className="inline-block h-4 w-4 rounded-full bg-white transition-transform"
              style={{ transform: coldChain ? 'translateX(22px)' : 'translateX(4px)' }}
            />
          </button>
        </div>
        <div>
          <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>
            {t('supplier_portal.auth.register.step3.palletization_label')}
          </label>
          <select
            className="md-input-outlined w-full"
            value={palletization}
            onChange={e => onPalletization(e.target.value)}
          >
            <option value="">{t('supplier_portal.auth.register.step3.palletization.not_specified')}</option>
            <option value="LOOSE_CARTONS">{t('supplier_portal.auth.register.step3.palletization.loose_cartons')}</option>
            <option value="EURO_PALLETS">{t('supplier_portal.auth.register.step3.palletization.euro_pallets')}</option>
            <option value="MIXED">{t('supplier_portal.auth.register.step3.palletization.mixed')}</option>
          </select>
        </div>
      </div>
    </div>
  );
}

// ─── Step 4: Product Categories ───────────────────────────────────────────────
function Step4Categories({
  selectedCats, toggleCat,
}: {
  selectedCats: string[];
  toggleCat: (id: string) => void;
}) {
  const { locale, t } = useLocale();
  const [catSearch, setCatSearch] = useState('');
  const localizedCategories = useMemo(
    () => CATEGORY_IDS.map((id) => ({
      id,
      label: t(`supplier_portal.auth.register.step4.categories.${id}`),
    })),
    [t],
  );
  const normalizedSearch = catSearch.trim().toLocaleLowerCase(locale);
  const filtered = useMemo(() => {
    if (!normalizedSearch) return localizedCategories;
    return localizedCategories.filter((category) =>
      category.label.toLocaleLowerCase(locale).includes(normalizedSearch),
    );
  }, [localizedCategories, locale, normalizedSearch]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <label className="md-typescale-label-medium" style={{ color: 'var(--foreground)' }}>
          {t('supplier_portal.auth.register.step4.label')} <span style={{ color: 'var(--accent)' }}>({selectedCats.length} {t('supplier_portal.auth.register.step4.selected_suffix')})</span>
        </label>
      </div>
      <input
        type="text"
        value={catSearch}
        onChange={e => setCatSearch(e.target.value)}
        placeholder={t('supplier_portal.auth.register.step4.search_placeholder')}
        className="md-input-outlined w-full"
        autoFocus
      />
      <div className="grid grid-cols-2 gap-2">
        {filtered.map(cat => {
          const active = selectedCats.includes(cat.id);
          return (
            <button
              key={cat.id}
              type="button"
              onClick={() => toggleCat(cat.id)}
              className={active ? 'md-chip md-chip-selected text-left' : 'md-chip text-left'}
              style={{ justifyContent: 'flex-start', padding: '10px 14px' }}
            >
              {active && (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" className="shrink-0">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/>
                </svg>
              )}
              <span className="md-typescale-label-medium">{cat.label}</span>
            </button>
          );
        })}
      </div>
      {selectedCats.length === 0 && (
        <p className="md-typescale-label-small mt-1" style={{ color: 'var(--danger)' }}>
          {t('supplier_portal.auth.register.step4.required_error')}
        </p>
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function SupplierRegisterPage() {
  const router = useRouter();
  const { locale, t } = useLocale();
  const [step, setStep] = useState(0);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const [country, setCountry] = useState(DEFAULT_COUNTRY);
  const [account, setAccount] = useState({
    companyName: '', contactPerson: '', email: '', phone: '', password: '', confirmPassword: '',
  });
  const [location, setLocation] = useState({
    warehouseAddress: '', warehouseLat: '', warehouseLng: '', billingAddress: '',
  });
  const [business, setBusiness] = useState({
    taxId: '', companyRegNumber: '',
  });
  const [fleetColdChain, setFleetColdChain] = useState(false);
  const [palletizationStandard, setPalletizationStandard] = useState('');
  const [selectedCats, setSelectedCats] = useState<string[]>([]);

  const patchAccount = (k: string, v: string) => setAccount(p => ({ ...p, [k]: v }));
  const patchLocation = (k: string, v: string) => setLocation(p => ({ ...p, [k]: v }));
  const patchBusiness = (k: string, v: string) => setBusiness(p => ({ ...p, [k]: v }));
  const toggleCat = (id: string) => setSelectedCats(p => p.includes(id) ? p.filter(x => x !== id) : [...p, id]);

  const canProceed = useCallback(() => {
    if (step === 0) {
      return account.companyName && account.contactPerson && account.email && account.phone &&
        account.password.length >= 8 && account.password === account.confirmPassword && country;
    }
    if (step === 1) return !!account.companyName;
    if (step === 2) return !!business.taxId;
    if (step === 3) return selectedCats.length > 0;
    return false;
  }, [step, account, business, selectedCats, country]);

  const next = () => {
    setError('');
    if (step === 0 && account.password !== account.confirmPassword) {
      setError(t('supplier_portal.auth.register.error.password_mismatch'));
      return;
    }
    setStep(s => Math.min(s + 1, STEPS.length - 1));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const back = () => { setStep(s => Math.max(s - 1, 0)); setError(''); };

  const submit = useCallback(async () => {
    setError('');
    setSubmitting(true);
    try {
      const phoneWithPrefix = `${dialingPrefix(country)}${account.phone.replace(/^0+/, '')}`;
      const body = {
        company_name: account.companyName,
        contact_person: account.contactPerson,
        email: account.email,
        phone: phoneWithPrefix,
        password: account.password,
        country_code: country,
        warehouse_address: location.warehouseAddress,
        warehouse_lat: location.warehouseLat ? parseFloat(location.warehouseLat) : 0,
        warehouse_lng: location.warehouseLng ? parseFloat(location.warehouseLng) : 0,
        billing_address: location.billingAddress || location.warehouseAddress,
        tax_id: business.taxId,
        company_reg_number: business.companyRegNumber,
        categories: selectedCats,
        fleet_cold_chain_compliant: fleetColdChain,
        ...(palletizationStandard && { palletization_standard: palletizationStandard }),
      };

      const res = await fetch(`${API}/v1/auth/supplier/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({ title: t('supplier_portal.auth.register.error.registration_failed') }));
        const errorMessage = body?.error === 'rate_limit_exceeded'
          ? t('error.rate_limited')
          : body?.message_key || body?.detail || body?.title
            ? translateProblemDetail(body, locale)
            : body?.error || t('supplier_portal.auth.register.error.http_error', { status: res.status });
        setError(errorMessage);
        setSubmitting(false);
        return;
      }

      const data = await res.json();
      document.cookie = `pegasus_admin_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      document.cookie = `admin_name=${encodeURIComponent(data.name || account.companyName)}; path=/; max-age=86400; SameSite=Lax`;
      if (isTauri()) {
        await storeToken(data.token, data.refresh_token || '');
      }
      // Exchange Firebase custom token for ID token session (graceful — legacy cookie still works)
      if (data.firebase_token) {
        await exchangeCustomToken(data.firebase_token);
      }
      // Redirect to billing setup (bank & payment gateway) before dashboard
      router.push('/setup/billing');
    } catch {
      setError(t('supplier_portal.auth.register.error.network'));
      setSubmitting(false);
    }
  }, [account, business, country, fleetColdChain, locale, location, palletizationStandard, router, selectedCats, t]);

  const stepValid = canProceed();
  const isLast = step === STEPS.length - 1;

  return (
    <div className="w-full py-6">
      {/* Brand — visible on mobile where the left panel is hidden */}
      <div className="flex items-center gap-3 mb-6 lg:hidden">
        <div
          className="w-10 h-10 rounded-xl flex items-center justify-center"
          style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
        >
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
            <path d="M20 4H4v2h16V4zm1 10v-2l-1-5H4l-1 5v2h1v6h10v-6h4v6h2v-6h1zm-9 4H6v-4h6v4z"/>
          </svg>
        </div>
        <div>
          <h1 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>{t('supplier_portal.auth.register.brand_title')}</h1>
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>{t('supplier_portal.auth.register.brand_subtitle')}</p>
        </div>
      </div>

        {/* Step indicator — circles with connecting lines */}
        <div className="auth-step-indicator">
          {STEPS.map((label, i) => {
            const isDone = i < step;
            const isActive = i === step;
            return (
              <div key={label} className={`auth-step-item${isDone ? ' step-done' : ''}`}>
                <div className={`auth-step-dot${isActive ? ' step-active' : ''}${isDone ? ' step-complete' : ''}`}>
                  {isDone ? (
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/></svg>
                  ) : (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d={STEP_ICONS[i]} /></svg>
                  )}
                </div>
                <span className={`auth-step-label${isActive ? ' step-active-label' : ''}`}>{t(`supplier_portal.auth.register.steps.${label}`)}</span>
              </div>
            );
          })}
        </div>

        {/* Form Content */}
        <div className="py-2">
          {/* Step Header */}
          <div className="mb-6">
            <h2 className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
              {step === 0 && t('supplier_portal.auth.register.header.account_title')}
              {step === 1 && t('supplier_portal.auth.register.header.location_title')}
              {step === 2 && t('supplier_portal.auth.register.header.business_title')}
              {step === 3 && t('supplier_portal.auth.register.header.categories_title')}
            </h2>
            <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
              {step === 0 && t('supplier_portal.auth.register.header.account_description')}
              {step === 1 && t('supplier_portal.auth.register.header.location_description')}
              {step === 2 && t('supplier_portal.auth.register.header.business_description')}
              {step === 3 && t('supplier_portal.auth.register.header.categories_description')}
            </p>
          </div>

          {/* Error */}
          {error && (
            <div className="mb-5 px-4 py-3 md-shape-lg md-typescale-body-small flex items-start gap-3" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }} role="alert">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" className="shrink-0 mt-0.5"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
              {error}
            </div>
          )}

          {/* Step Content — animated on step change */}
          <div key={step} className="auth-step-enter">
            {step === 0 && <Step1 data={account} onChange={patchAccount} country={country} onCountryChange={setCountry} />}
            {step === 1 && <Step2 data={location} onChange={patchLocation} />}
            {step === 2 && <Step3 data={business} onChange={patchBusiness} coldChain={fleetColdChain} onColdChainToggle={() => setFleetColdChain(v => !v)} palletization={palletizationStandard} onPalletization={setPalletizationStandard} />}
            {step === 3 && <Step4Categories selectedCats={selectedCats} toggleCat={toggleCat} />}
          </div>

          {/* Navigation */}
          <div className="flex gap-3 mt-8">
            {step > 0 && (
              <Button variant="outline" className="px-6" onPress={back}>
                {t('common.action.back')}
              </Button>
            )}
            <Button
              variant="primary"
              className="flex-1 py-3"
              onPress={isLast ? submit : next}
              isDisabled={!stepValid || submitting}
            >
              {submitting ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  {t('supplier_portal.auth.register.creating_account')}
                </span>
              ) : isLast ? t('supplier_portal.auth.register.submit') : t('supplier_portal.auth.register.continue_to', { step: t(`supplier_portal.auth.register.steps.${STEPS[step + 1]}`) })}
            </Button>
          </div>

          {step === 0 && (
            <p className="text-center md-typescale-body-small mt-5" style={{ color: 'var(--muted)' }}>
              {t('supplier_portal.auth.register.already_registered')}{' '}
              <a href="/auth/login" className="font-semibold hover:underline" style={{ color: 'var(--accent)' }}>{t('common.action.sign_in')}</a>
            </p>
          )}
        </div>

        <p className="text-center mt-6 md-typescale-label-small lg:hidden" style={{ color: 'var(--muted)', opacity: 0.6 }}>
          Pegasus &copy; 2026
        </p>
      </div>
  );
}
