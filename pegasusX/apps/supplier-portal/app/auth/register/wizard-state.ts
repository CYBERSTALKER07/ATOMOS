// Shared step-form types for the supplier onboarding wizard.
//
// HARD PRODUCT INVARIANT: this wizard has exactly four steps —
//   1. Account     (country selector + phone with auto-prefix)
//   2. Location    (warehouse + billing address)
//   3. Business    (tax id, company reg, fleet config)
//   4. Categories  (product / service categories the supplier serves)
// Bank + payment-gateway setup is intentionally NOT in this wizard —
// it lives at /setup/billing post-registration to reduce registration
// friction. Do not collapse below 4 steps. Do not move banking back in.

export type StepId = "account" | "location" | "business" | "categories";

export interface Country {
  code: string;       // ISO-3166 alpha-2
  name: string;
  dialCode: string;   // e.g. "+998"
  currency: string;   // ISO-4217
}

export const COUNTRIES: Country[] = [
  { code: "UZ", name: "Uzbekistan",    dialCode: "+998", currency: "UZS" },
  { code: "KZ", name: "Kazakhstan",    dialCode: "+7",   currency: "KZT" },
  { code: "KG", name: "Kyrgyzstan",    dialCode: "+996", currency: "KGS" },
  { code: "TJ", name: "Tajikistan",    dialCode: "+992", currency: "TJS" },
  { code: "TM", name: "Turkmenistan",  dialCode: "+993", currency: "TMT" },
  { code: "AE", name: "United Arab Emirates", dialCode: "+971", currency: "AED" },
  { code: "TR", name: "Türkiye",       dialCode: "+90",  currency: "TRY" },
  { code: "RU", name: "Russia",        dialCode: "+7",   currency: "RUB" },
  { code: "US", name: "United States", dialCode: "+1",   currency: "USD" },
  { code: "GB", name: "United Kingdom",dialCode: "+44",  currency: "GBP" },
];

export const CATEGORY_OPTIONS: ReadonlyArray<{ id: string; label: string }> = [
  { id: "GROCERY",      label: "Grocery" },
  { id: "BEVERAGES",    label: "Beverages" },
  { id: "HOUSEHOLD",    label: "Household goods" },
  { id: "PERSONAL_CARE",label: "Personal care" },
  { id: "PHARMACY",     label: "Pharmacy" },
  { id: "ELECTRONICS",  label: "Electronics" },
  { id: "STATIONERY",   label: "Stationery" },
  { id: "TOBACCO",      label: "Tobacco" },
  { id: "FROZEN",       label: "Frozen" },
  { id: "BAKERY",       label: "Bakery" },
];

export interface AccountStep {
  legalName: string;
  contactName: string;
  email: string;
  countryCode: string;     // ISO-3166 alpha-2
  phoneLocal: string;      // local portion only — UI shows the dial code separately
  password: string;
}

export interface LocationStep {
  warehouseName: string;
  warehouseLine1: string;
  warehouseCity: string;
  warehouseRegion: string;
  warehousePostalCode: string;
  warehouseLat: string;    // string in UI; coerced to number on submit
  warehouseLng: string;

  billingSameAsWarehouse: boolean;
  billingLine1: string;
  billingCity: string;
  billingRegion: string;
  billingPostalCode: string;
}

export interface BusinessStep {
  taxId: string;
  companyRegNumber: string;
  fleetVehicleCount: number;
  fleetMaxVU: number;       // total Volumetric Units across fleet
  factoryCount: number;
}

export interface CategoriesStep {
  selectedCategoryIds: string[];
}

export interface WizardState {
  step: StepId;
  account: AccountStep;
  location: LocationStep;
  business: BusinessStep;
  categories: CategoriesStep;
}

export const INITIAL_STATE: WizardState = {
  step: "account",
  account: {
    legalName: "",
    contactName: "",
    email: "",
    countryCode: "UZ",
    phoneLocal: "",
    password: "",
  },
  location: {
    warehouseName: "",
    warehouseLine1: "",
    warehouseCity: "",
    warehouseRegion: "",
    warehousePostalCode: "",
    warehouseLat: "",
    warehouseLng: "",
    billingSameAsWarehouse: true,
    billingLine1: "",
    billingCity: "",
    billingRegion: "",
    billingPostalCode: "",
  },
  business: {
    taxId: "",
    companyRegNumber: "",
    fleetVehicleCount: 0,
    fleetMaxVU: 0,
    factoryCount: 0,
  },
  categories: {
    selectedCategoryIds: [],
  },
};

export const STEP_ORDER: StepId[] = ["account", "location", "business", "categories"];

export const STEP_LABELS: Record<StepId, string> = {
  account: "Account",
  location: "Location",
  business: "Business",
  categories: "Categories",
};

// Validation returns a map of field-name -> error string. Empty map = valid.
export function validateAccount(s: AccountStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (s.legalName.trim().length < 2) e.legalName = "Legal name is required";
  if (s.contactName.trim().length < 2) e.contactName = "Contact name is required";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s.email)) e.email = "Valid email is required";
  if (!s.countryCode) e.countryCode = "Country is required";
  if (!/^\d{6,14}$/.test(s.phoneLocal)) e.phoneLocal = "Phone digits only (6–14)";
  if (s.password.length < 10) e.password = "Password must be at least 10 characters";
  return e;
}

export function validateLocation(s: LocationStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (s.warehouseName.trim().length < 2) e.warehouseName = "Warehouse name required";
  if (s.warehouseLine1.trim().length < 3) e.warehouseLine1 = "Street address required";
  if (s.warehouseCity.trim().length < 2) e.warehouseCity = "City required";
  const lat = Number(s.warehouseLat);
  const lng = Number(s.warehouseLng);
  if (!isFinite(lat) || lat < -90 || lat > 90) e.warehouseLat = "Latitude must be in [-90, 90]";
  if (!isFinite(lng) || lng < -180 || lng > 180) e.warehouseLng = "Longitude must be in [-180, 180]";
  if (!s.billingSameAsWarehouse) {
    if (s.billingLine1.trim().length < 3) e.billingLine1 = "Billing street required";
    if (s.billingCity.trim().length < 2) e.billingCity = "Billing city required";
  }
  return e;
}

export function validateBusiness(s: BusinessStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (s.taxId.trim().length < 4) e.taxId = "Tax ID required";
  if (s.companyRegNumber.trim().length < 4) e.companyRegNumber = "Company registration number required";
  if (!Number.isInteger(s.fleetVehicleCount) || s.fleetVehicleCount < 0) e.fleetVehicleCount = "Vehicle count must be a non-negative integer";
  if (!Number.isInteger(s.fleetMaxVU) || s.fleetMaxVU < 0) e.fleetMaxVU = "Fleet VU must be a non-negative integer";
  if (!Number.isInteger(s.factoryCount) || s.factoryCount < 0) e.factoryCount = "Factory count must be a non-negative integer";
  return e;
}

export function validateCategories(s: CategoriesStep): Record<string, string> {
  const e: Record<string, string> = {};
  if (s.selectedCategoryIds.length === 0) e.selectedCategoryIds = "Select at least one category";
  return e;
}
