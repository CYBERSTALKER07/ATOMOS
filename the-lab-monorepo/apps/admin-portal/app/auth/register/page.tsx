'use client';

import { useState, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import dynamic from 'next/dynamic';
import { Button } from '@heroui/react';
import { exchangeCustomToken } from '../../../lib/firebase';
import { COUNTRIES, DEFAULT_COUNTRY, findCountry, dialingPrefix } from '../../../lib/constants/countries';

const LocationPicker = dynamic(
  () => import('../../../components/location-picker/location-picker'),
  { ssr: false, loading: () => <div className="w-full h-72 flex items-center justify-center" style={{ background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 12 }}><span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Loading map...</span></div> }
);

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// ─── All canonical categories — Walmart-scale shelf coverage ─────────────────
const CATEGORIES = [
  // ── Beverages ──
  { id: 'cat-water', label: 'Water' },
  { id: 'cat-sparkling-water', label: 'Sparkling & Flavored Water' },
  { id: 'cat-soft-drinks', label: 'Soft Drinks & Soda' },
  { id: 'cat-juice', label: 'Juice & Nectars' },
  { id: 'cat-tea-coffee', label: 'Tea & Coffee' },
  { id: 'cat-energy-sports', label: 'Energy & Sports Drinks' },
  { id: 'cat-powdered-drinks', label: 'Powdered Drink Mixes' },
  { id: 'cat-kombucha', label: 'Kombucha & Probiotic Drinks' },
  { id: 'cat-plant-milk', label: 'Plant-Based Milk' },
  // ── Dairy & Refrigerated ──
  { id: 'cat-milk', label: 'Milk & Cream' },
  { id: 'cat-yogurt', label: 'Yogurt & Kefir' },
  { id: 'cat-cheese', label: 'Cheese' },
  { id: 'cat-butter-margarine', label: 'Butter & Margarine' },
  { id: 'cat-eggs', label: 'Eggs' },
  { id: 'cat-deli-meats', label: 'Deli Meats & Charcuterie' },
  { id: 'cat-hummus-dips', label: 'Hummus, Dips & Spreads' },
  { id: 'cat-tofu-tempeh', label: 'Tofu, Tempeh & Meat Alternatives' },
  { id: 'cat-fresh-pasta-sauces', label: 'Fresh Pasta & Sauces' },
  // ── Bakery ──
  { id: 'cat-bread', label: 'Bread & Rolls' },
  { id: 'cat-tortillas-wraps', label: 'Tortillas & Wraps' },
  { id: 'cat-bagels-english-muffins', label: 'Bagels & English Muffins' },
  { id: 'cat-cakes-pastries', label: 'Cakes & Pastries' },
  { id: 'cat-cookies-brownies', label: 'Cookies & Brownies' },
  { id: 'cat-pita-naan', label: 'Pita, Naan & Flatbread' },
  { id: 'cat-baking-mixes', label: 'Baking Mixes' },
  // ── Breakfast ──
  { id: 'cat-cereal', label: 'Cereal' },
  { id: 'cat-oatmeal-porridge', label: 'Oatmeal & Porridge' },
  { id: 'cat-granola-bars', label: 'Granola & Cereal Bars' },
  { id: 'cat-pancake-waffle', label: 'Pancake & Waffle Mix' },
  { id: 'cat-syrup-honey', label: 'Syrup, Honey & Jam' },
  // ── Pantry Staples ──
  { id: 'cat-rice', label: 'Rice' },
  { id: 'cat-pasta-noodles', label: 'Pasta & Noodles' },
  { id: 'cat-flour-baking', label: 'Flour & Baking Ingredients' },
  { id: 'cat-sugar-sweeteners', label: 'Sugar & Sweeteners' },
  { id: 'cat-cooking-oils', label: 'Cooking Oils' },
  { id: 'cat-vinegar', label: 'Vinegar' },
  { id: 'cat-canned-vegetables', label: 'Canned Vegetables' },
  { id: 'cat-canned-fruit', label: 'Canned Fruit' },
  { id: 'cat-canned-beans-legumes', label: 'Canned Beans & Legumes' },
  { id: 'cat-canned-meat-fish', label: 'Canned Meat & Fish' },
  { id: 'cat-canned-soup', label: 'Canned Soup & Broth' },
  { id: 'cat-tomato-products', label: 'Tomato Sauce & Paste' },
  { id: 'cat-condiments', label: 'Condiments & Ketchup' },
  { id: 'cat-mayonnaise-dressings', label: 'Mayonnaise & Dressings' },
  { id: 'cat-mustard-hot-sauce', label: 'Mustard & Hot Sauce' },
  { id: 'cat-soy-asian-sauces', label: 'Soy & Asian Sauces' },
  { id: 'cat-bbq-marinades', label: 'BBQ Sauce & Marinades' },
  { id: 'cat-spices-herbs', label: 'Spices & Dried Herbs' },
  { id: 'cat-salt-pepper', label: 'Salt & Pepper' },
  { id: 'cat-dried-beans-lentils', label: 'Dried Beans & Lentils' },
  { id: 'cat-grains-couscous', label: 'Grains, Quinoa & Couscous' },
  // ── Snacks ──
  { id: 'cat-chips-crisps', label: 'Chips & Crisps' },
  { id: 'cat-popcorn', label: 'Popcorn' },
  { id: 'cat-pretzels-crackers', label: 'Pretzels & Crackers' },
  { id: 'cat-nuts-seeds', label: 'Nuts & Seeds' },
  { id: 'cat-dried-fruit', label: 'Dried Fruit & Trail Mix' },
  { id: 'cat-jerky-meat-snacks', label: 'Jerky & Meat Snacks' },
  { id: 'cat-protein-bars', label: 'Protein & Nutrition Bars' },
  { id: 'cat-rice-cakes', label: 'Rice Cakes & Puffed Snacks' },
  // ── Candy & Sweets ──
  { id: 'cat-chocolate', label: 'Chocolate' },
  { id: 'cat-gummy-candy', label: 'Gummy & Chewy Candy' },
  { id: 'cat-hard-candy-mints', label: 'Hard Candy & Mints' },
  { id: 'cat-chewing-gum', label: 'Chewing Gum' },
  { id: 'cat-biscuits-cookies', label: 'Biscuits & Cookies' },
  { id: 'cat-ice-cream-toppings', label: 'Ice Cream Toppings & Cones' },
  // ── Fresh Produce ──
  { id: 'cat-fresh-fruit', label: 'Fresh Fruit' },
  { id: 'cat-fresh-vegetables', label: 'Fresh Vegetables' },
  { id: 'cat-herbs-salads', label: 'Fresh Herbs & Salads' },
  { id: 'cat-organic-produce', label: 'Organic Produce' },
  { id: 'cat-mushrooms', label: 'Mushrooms' },
  { id: 'cat-potatoes-onions', label: 'Potatoes, Onions & Root Veg' },
  // ── Meat & Seafood ──
  { id: 'cat-beef', label: 'Beef' },
  { id: 'cat-chicken-turkey', label: 'Chicken & Turkey' },
  { id: 'cat-pork', label: 'Pork' },
  { id: 'cat-lamb', label: 'Lamb & Mutton' },
  { id: 'cat-ground-meat', label: 'Ground Meat & Mince' },
  { id: 'cat-sausages-hotdogs', label: 'Sausages & Hot Dogs' },
  { id: 'cat-fresh-fish', label: 'Fresh Fish' },
  { id: 'cat-shrimp-shellfish', label: 'Shrimp & Shellfish' },
  { id: 'cat-smoked-fish', label: 'Smoked & Cured Fish' },
  // ── Frozen ──
  { id: 'cat-frozen-meals', label: 'Frozen Meals & Entrées' },
  { id: 'cat-frozen-pizza', label: 'Frozen Pizza' },
  { id: 'cat-frozen-vegetables', label: 'Frozen Vegetables' },
  { id: 'cat-frozen-fruit', label: 'Frozen Fruit' },
  { id: 'cat-frozen-meat-seafood', label: 'Frozen Meat & Seafood' },
  { id: 'cat-frozen-snacks', label: 'Frozen Snacks & Appetizers' },
  { id: 'cat-frozen-breakfast', label: 'Frozen Breakfast' },
  { id: 'cat-ice-cream', label: 'Ice Cream & Frozen Desserts' },
  { id: 'cat-frozen-fries-potatoes', label: 'Frozen Fries & Potatoes' },
  // ── International & Specialty ──
  { id: 'cat-mexican-food', label: 'Mexican & Latin Food' },
  { id: 'cat-asian-food', label: 'Asian Food' },
  { id: 'cat-indian-food', label: 'Indian Food' },
  { id: 'cat-middle-eastern-food', label: 'Middle Eastern Food' },
  { id: 'cat-italian-food', label: 'Italian Specialty' },
  { id: 'cat-kosher', label: 'Kosher' },
  { id: 'cat-halal', label: 'Halal' },
  { id: 'cat-gluten-free', label: 'Gluten-Free' },
  { id: 'cat-vegan-plant-based', label: 'Vegan & Plant-Based' },
  { id: 'cat-organic-natural', label: 'Organic & Natural' },
  // ── Baby & Kids ──
  { id: 'cat-baby-formula', label: 'Baby Formula' },
  { id: 'cat-baby-food', label: 'Baby Food & Snacks' },
  { id: 'cat-diapers', label: 'Diapers & Wipes' },
  { id: 'cat-baby-bath-skin', label: 'Baby Bath & Skin Care' },
  { id: 'cat-baby-feeding', label: 'Baby Bottles & Feeding' },
  // ── Health & Wellness ──
  { id: 'cat-medicine-otc', label: 'OTC Medicine & Pain Relief' },
  { id: 'cat-cold-flu', label: 'Cold, Flu & Allergy' },
  { id: 'cat-digestive-health', label: 'Digestive Health' },
  { id: 'cat-first-aid', label: 'First Aid & Bandages' },
  { id: 'cat-vitamins', label: 'Vitamins & Supplements' },
  { id: 'cat-protein-powder', label: 'Protein Powder & Shakes' },
  { id: 'cat-eye-ear-care', label: 'Eye & Ear Care' },
  { id: 'cat-diabetes-care', label: 'Diabetes Care' },
  { id: 'cat-mobility-aids', label: 'Mobility & Daily Living Aids' },
  // ── Personal Care & Beauty ──
  { id: 'cat-shampoo-conditioner', label: 'Shampoo & Conditioner' },
  { id: 'cat-hair-styling', label: 'Hair Styling & Treatment' },
  { id: 'cat-hair-color', label: 'Hair Color & Dye' },
  { id: 'cat-body-wash-soap', label: 'Body Wash & Bar Soap' },
  { id: 'cat-deodorant', label: 'Deodorant & Antiperspirant' },
  { id: 'cat-lotion-moisturizer', label: 'Lotion & Moisturizer' },
  { id: 'cat-sunscreen', label: 'Sunscreen & Sun Care' },
  { id: 'cat-face-care', label: 'Face Care & Cleansers' },
  { id: 'cat-lip-care', label: 'Lip Care & Balm' },
  { id: 'cat-oral-care', label: 'Toothpaste & Oral Care' },
  { id: 'cat-mouthwash', label: 'Mouthwash & Floss' },
  { id: 'cat-shaving', label: 'Shaving & Razors' },
  { id: 'cat-mens-grooming', label: "Men's Grooming" },
  { id: 'cat-feminine-care', label: 'Feminine Care' },
  { id: 'cat-cotton-pads', label: 'Cotton, Swabs & Pads' },
  { id: 'cat-cosmetics-face', label: 'Face Makeup & Foundation' },
  { id: 'cat-cosmetics-eyes', label: 'Eye Makeup & Mascara' },
  { id: 'cat-cosmetics-lips', label: 'Lipstick & Lip Gloss' },
  { id: 'cat-nail-care', label: 'Nail Polish & Nail Care' },
  { id: 'cat-fragrance', label: 'Perfume & Fragrance' },
  { id: 'cat-hair-tools', label: 'Hair Dryers, Irons & Tools' },
  // ── Household Cleaning ──
  { id: 'cat-all-purpose-cleaners', label: 'All-Purpose Cleaners' },
  { id: 'cat-bathroom-cleaners', label: 'Bathroom Cleaners' },
  { id: 'cat-kitchen-cleaners', label: 'Kitchen & Oven Cleaners' },
  { id: 'cat-glass-cleaners', label: 'Glass & Window Cleaners' },
  { id: 'cat-floor-care', label: 'Floor Care & Mopping' },
  { id: 'cat-disinfectants', label: 'Disinfectants & Sanitizers' },
  { id: 'cat-laundry-detergent', label: 'Laundry Detergent' },
  { id: 'cat-fabric-softener', label: 'Fabric Softener & Dryer Sheets' },
  { id: 'cat-stain-removers', label: 'Stain Removers' },
  { id: 'cat-dishwashing', label: 'Dish Soap & Dishwasher Pods' },
  { id: 'cat-trash-bags', label: 'Trash Bags' },
  { id: 'cat-sponges-cloths', label: 'Sponges, Cloths & Brushes' },
  { id: 'cat-air-fresheners', label: 'Air Fresheners & Candles' },
  { id: 'cat-pest-control', label: 'Pest Control & Insect Repellent' },
  // ── Paper & Disposable ──
  { id: 'cat-toilet-paper', label: 'Toilet Paper' },
  { id: 'cat-paper-towels', label: 'Paper Towels' },
  { id: 'cat-facial-tissue', label: 'Facial Tissue' },
  { id: 'cat-napkins', label: 'Napkins' },
  { id: 'cat-plates-cups-disposable', label: 'Disposable Plates, Cups & Cutlery' },
  { id: 'cat-food-wrap-bags', label: 'Food Wrap, Foil & Zip Bags' },
  // ── Kitchen & Dining ──
  { id: 'cat-cookware', label: 'Cookware & Pots' },
  { id: 'cat-bakeware', label: 'Bakeware' },
  { id: 'cat-kitchen-utensils', label: 'Kitchen Utensils & Gadgets' },
  { id: 'cat-knives-cutting', label: 'Knives & Cutting Boards' },
  { id: 'cat-food-storage', label: 'Food Storage & Containers' },
  { id: 'cat-water-bottles', label: 'Water Bottles & Tumblers' },
  { id: 'cat-dinnerware', label: 'Dinnerware & Glassware' },
  { id: 'cat-small-appliances', label: 'Small Kitchen Appliances' },
  // ── Home & Living ──
  { id: 'cat-bedding', label: 'Bedding & Sheets' },
  { id: 'cat-pillows-blankets', label: 'Pillows & Blankets' },
  { id: 'cat-towels', label: 'Bath Towels & Mats' },
  { id: 'cat-curtains-blinds', label: 'Curtains & Blinds' },
  { id: 'cat-home-decor', label: 'Home Décor & Frames' },
  { id: 'cat-candles-home-fragrance', label: 'Candles & Home Fragrance' },
  { id: 'cat-storage-organization', label: 'Storage & Organization' },
  { id: 'cat-hangers-laundry-supplies', label: 'Hangers & Laundry Supplies' },
  { id: 'cat-light-bulbs', label: 'Light Bulbs' },
  { id: 'cat-batteries', label: 'Batteries' },
  { id: 'cat-extension-cords', label: 'Extension Cords & Power Strips' },
  // ── Garden & Outdoor ──
  { id: 'cat-plants-seeds', label: 'Plants & Seeds' },
  { id: 'cat-soil-fertilizer', label: 'Soil & Fertilizer' },
  { id: 'cat-garden-tools', label: 'Garden Tools' },
  { id: 'cat-outdoor-furniture', label: 'Outdoor Furniture' },
  { id: 'cat-grills-charcoal', label: 'Grills & Charcoal' },
  // ── Pets ──
  { id: 'cat-dog-food', label: 'Dog Food' },
  { id: 'cat-cat-food', label: 'Cat Food' },
  { id: 'cat-pet-treats', label: 'Pet Treats' },
  { id: 'cat-pet-toys', label: 'Pet Toys' },
  { id: 'cat-pet-grooming', label: 'Pet Grooming & Health' },
  { id: 'cat-litter-waste', label: 'Cat Litter & Waste Bags' },
  { id: 'cat-pet-beds-carriers', label: 'Pet Beds & Carriers' },
  // ── Office & School ──
  { id: 'cat-pens-pencils', label: 'Pens, Pencils & Markers' },
  { id: 'cat-notebooks-paper', label: 'Notebooks & Paper' },
  { id: 'cat-binders-folders', label: 'Binders, Folders & Filing' },
  { id: 'cat-tape-glue-scissors', label: 'Tape, Glue & Scissors' },
  { id: 'cat-printer-ink', label: 'Printer Ink & Toner' },
  { id: 'cat-backpacks-bags', label: 'Backpacks & School Bags' },
  // ── Toys & Games ──
  { id: 'cat-action-figures', label: 'Action Figures & Dolls' },
  { id: 'cat-building-sets', label: 'Building Sets & Blocks' },
  { id: 'cat-board-games-puzzles', label: 'Board Games & Puzzles' },
  { id: 'cat-outdoor-play', label: 'Outdoor Play & Sports Toys' },
  { id: 'cat-arts-crafts', label: 'Arts & Crafts' },
  { id: 'cat-stuffed-animals', label: 'Stuffed Animals & Plush' },
  // ── Electronics & Accessories ──
  { id: 'cat-phone-accessories', label: 'Phone Cases & Accessories' },
  { id: 'cat-chargers-cables', label: 'Chargers & Cables' },
  { id: 'cat-headphones-earbuds', label: 'Headphones & Earbuds' },
  { id: 'cat-memory-cards-usb', label: 'Memory Cards & USB Drives' },
  { id: 'cat-smart-home', label: 'Smart Home Devices' },
  // ── Automotive ──
  { id: 'cat-motor-oil', label: 'Motor Oil & Fluids' },
  { id: 'cat-car-cleaning', label: 'Car Cleaning & Detailing' },
  { id: 'cat-car-fresheners', label: 'Car Air Fresheners' },
  { id: 'cat-car-accessories', label: 'Car Accessories' },
  // ── Party & Seasonal ──
  { id: 'cat-party-supplies', label: 'Party Supplies & Balloons' },
  { id: 'cat-gift-wrap', label: 'Gift Wrap & Bags' },
  { id: 'cat-greeting-cards', label: 'Greeting Cards' },
  { id: 'cat-seasonal', label: 'Seasonal & Holiday Items' },
  // ── Tobacco & Adjacent ──
  { id: 'cat-tobacco', label: 'Tobacco Products' },
  { id: 'cat-lighters-matches', label: 'Lighters & Matches' },
  // ── Other ──
  { id: 'cat-other', label: 'Other' },
];

const STEPS = ['Account', 'Location', 'Business', 'Categories'];

const STEP_ICONS: Record<number, string> = {
  0: 'M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z',
  1: 'M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z',
  2: 'M12 7V3H2v18h20V7H12zM6 19H4v-2h2v2zm0-4H4v-2h2v2zm0-4H4V9h2v2zm0-4H4V5h2v2zm4 12H8v-2h2v2zm0-4H8v-2h2v2zm0-4H8V9h2v2zm0-4H8V5h2v2zm10 12h-8v-2h2v-2h-2v-2h2v-2h-2V9h8v10zm-2-8h-2v2h2v-2zm0 4h-2v2h2v-2z',
  3: 'M4 10v7h3v-7H4zm6 0v7h3v-7h-3zM2 22h19v-3H2v3zm14-12v7h3v-7h-3zm-4.5-9L2 6v2h19V6l-9.5-5z',
};

// ─── Shared Input Component ───────────────────────────────────────────────────
function InputField({ label, ...props }: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
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
            aria-label={showPw ? 'Hide password' : 'Show password'}
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
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>Country *</label>
        <div className="relative">
          <button
            type="button"
            onClick={() => setCountryOpen(v => !v)}
            className="md-input-outlined w-full text-left flex items-center justify-between"
          >
            <span>{selected ? `${selected.label} (${selected.code})` : 'Select country'}</span>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--muted)' }}>
              <path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6z"/>
            </svg>
          </button>
          {countryOpen && (
            <div className="absolute z-50 mt-1 w-full max-h-60 overflow-auto md-shape-md" style={{ background: 'var(--surface-container)', border: '1px solid var(--border)' }}>
              <div className="sticky top-0 p-2" style={{ background: 'var(--surface-container)' }}>
                <input
                  type="text"
                  value={countrySearch}
                  onChange={e => setCountrySearch(e.target.value)}
                  placeholder="Search countries..."
                  className="md-input-outlined w-full"
                  autoFocus
                />
              </div>
              {filteredCountries.map(c => (
                <button
                  key={c.value}
                  type="button"
                  className="w-full text-left px-3 py-2 md-typescale-body-small hover:opacity-80 flex items-center justify-between"
                  style={{ background: c.value === country ? 'var(--accent-soft)' : 'transparent', color: 'var(--foreground)' }}
                  onClick={() => { onCountryChange(c.value); setCountryOpen(false); setCountrySearch(''); }}
                >
                  <span>{c.label}</span>
                  <span style={{ color: 'var(--muted)' }}>{c.code}</span>
                </button>
              ))}
              {filteredCountries.length === 0 && (
                <p className="px-3 py-4 md-typescale-label-small text-center" style={{ color: 'var(--muted)' }}>No countries match your search</p>
              )}
            </div>
          )}
        </div>
      </div>

      <InputField label="Company Name *" type="text" value={data.companyName} onChange={e => onChange('companyName', e.target.value)} placeholder="Lab Beverages Ltd." required autoFocus />
      <InputField label="Contact Person *" type="text" value={data.contactPerson} onChange={e => onChange('contactPerson', e.target.value)} placeholder="Aziz Karimov" required />
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <InputField label="Email Address *" type="email" value={data.email} onChange={e => onChange('email', e.target.value)} placeholder="info@company.uz" required />
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>Phone Number *</label>
          <div className="flex gap-2">
            <span className="md-input-outlined flex items-center px-3 shrink-0 md-typescale-body-small" style={{ color: 'var(--muted)', minWidth: 64 }}>{prefix}</span>
            <input type="tel" value={data.phone} onChange={e => onChange('phone', e.target.value)} placeholder="901234567" required className="md-input-outlined w-full" />
          </div>
        </div>
      </div>
      <InputField label="Password *" type="password" value={data.password} onChange={e => onChange('password', e.target.value)} placeholder="Minimum 8 characters" required />
      <InputField label="Confirm Password *" type="password" value={data.confirmPassword} onChange={e => onChange('confirmPassword', e.target.value)} placeholder="Repeat your password" required />
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
  const handleLocationChange = useCallback((lat: string, lng: string, address: string) => {
    onChange('warehouseLat', lat);
    onChange('warehouseLng', lng);
    onChange('warehouseAddress', address);
  }, [onChange]);

  return (
    <div className="space-y-4">
      <div>
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>
          Warehouse / Storage Location *
        </label>
        <LocationPicker
          lat={data.warehouseLat}
          lng={data.warehouseLng}
          address={data.warehouseAddress}
          onLocationChange={handleLocationChange}
        />
      </div>
      <InputField label="Billing Address" type="text" value={data.billingAddress} onChange={e => onChange('billingAddress', e.target.value)} placeholder="Same as warehouse or head office address" />
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
  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <InputField label="Tax ID / STIR *" type="text" value={data.taxId} onChange={e => onChange('taxId', e.target.value)} placeholder="STIR-12345678" required />
        <InputField label="Company Reg. Number" type="text" value={data.companyRegNumber} onChange={e => onChange('companyRegNumber', e.target.value)} placeholder="MYF-001-UZ" />
      </div>

      {/* Fleet Operations */}
      <div className="p-4 md-shape-md" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
        <p className="md-typescale-label-medium mb-3" style={{ color: 'var(--foreground)' }}>Fleet &amp; Logistics Profile</p>
        <div className="flex items-center justify-between mb-3">
          <div>
            <p className="md-typescale-body-small" style={{ color: 'var(--foreground)' }}>Cold Chain Compliant Fleet</p>
            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Refrigerated vehicles for temperature-sensitive goods</p>
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
          <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--muted)' }}>Palletization Standard</label>
          <select
            className="md-input-outlined w-full"
            value={palletization}
            onChange={e => onPalletization(e.target.value)}
          >
            <option value="">Not specified</option>
            <option value="LOOSE_CARTONS">Loose Cartons</option>
            <option value="EURO_PALLETS">Euro Pallets (120×80 cm)</option>
            <option value="MIXED">Mixed</option>
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
  const [catSearch, setCatSearch] = useState('');
  const filtered = catSearch
    ? CATEGORIES.filter(c => c.label.toLowerCase().includes(catSearch.toLowerCase()))
    : CATEGORIES;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <label className="md-typescale-label-medium" style={{ color: 'var(--foreground)' }}>
          Product Categories * <span style={{ color: 'var(--accent)' }}>({selectedCats.length} selected)</span>
        </label>
      </div>
      <input
        type="text"
        value={catSearch}
        onChange={e => setCatSearch(e.target.value)}
        placeholder="Search categories..."
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
        <p className="md-typescale-label-small mt-1" style={{ color: 'var(--danger)' }}>Select at least one category to continue.</p>
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function SupplierRegisterPage() {
  const router = useRouter();
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
      setError('Passwords do not match.');
      return;
    }
    setStep(s => Math.min(s + 1, STEPS.length - 1));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const back = () => { setStep(s => Math.max(s - 1, 0)); setError(''); };

  const submit = async () => {
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
        const j = await res.json().catch(() => ({ error: 'Registration failed' }));
        setError(j.error || `Error ${res.status}`);
        setSubmitting(false);
        return;
      }

      const data = await res.json();
      document.cookie = `admin_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      document.cookie = `admin_name=${encodeURIComponent(data.name || account.companyName)}; path=/; max-age=86400; SameSite=Lax`;
      // Exchange Firebase custom token for ID token session (graceful — legacy cookie still works)
      if (data.firebase_token) {
        await exchangeCustomToken(data.firebase_token);
      }
      // Redirect to billing setup (bank & payment gateway) before dashboard
      router.push('/setup/billing');
    } catch {
      setError('Network error \u2014 is the backend running?');
      setSubmitting(false);
    }
  };

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
          <h1 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>Lab Hub</h1>
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Supplier Registration</p>
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
                <span className={`auth-step-label${isActive ? ' step-active-label' : ''}`}>{label}</span>
              </div>
            );
          })}
        </div>

        {/* Form Content */}
        <div className="py-2">
          {/* Step Header */}
          <div className="mb-6">
            <h2 className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
              {step === 0 && 'Create your supplier account'}
              {step === 1 && 'Where are you based?'}
              {step === 2 && 'Business details'}
              {step === 3 && 'Product categories'}
            </h2>
            <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
              {step === 0 && 'Select your country and enter your primary company information'}
              {step === 1 && 'Warehouse location will be used for route planning and delivery logistics'}
              {step === 2 && 'Tax and fleet logistics configuration'}
              {step === 3 && 'Select all product categories you supply \u2014 retailers will filter by these'}
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
                Back
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
                  Creating Account...
                </span>
              ) : isLast ? 'Create Supplier Account' : `Continue to ${STEPS[step + 1]}`}
            </Button>
          </div>

          {step === 0 && (
            <p className="text-center md-typescale-body-small mt-5" style={{ color: 'var(--muted)' }}>
              Already registered?{' '}
              <a href="/auth/login" className="font-semibold hover:underline" style={{ color: 'var(--accent)' }}>Sign in</a>
            </p>
          )}
        </div>

        <p className="text-center mt-6 md-typescale-label-small lg:hidden" style={{ color: 'var(--muted)', opacity: 0.6 }}>
          The Lab Industries &copy; 2026
        </p>
      </div>
  );
}
