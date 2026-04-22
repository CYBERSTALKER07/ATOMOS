package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"config"
)

// ═══════════════════════════════════════════════════════════════════════════════
// THE LAB INDUSTRIES — E2E SEED SCRIPT
// Deterministic test matrix for local live-fire testing.
// Connects to Spanner emulator + Redis. Idempotent (truncates before insert).
// ═══════════════════════════════════════════════════════════════════════════════

const (
	supplierID  = "SUP-001"
	retailerID  = "RET-001"
	driverID    = "DRV-001"
	payloaderID = "WRK-001"
	vehicleID   = "VEH-001"

	sku1 = "COKE-500-50"
	sku2 = "SPRITE-1L-20"
	sku3 = "FANTA-CAN-24"

	primaryCategoryID = "cat-soft-drinks"

	geoKey = "geo:proximity"
)

// ── ADDITIONAL RETAILERS ───────────────────────────────────────────────────
type demoRetailer struct {
	id       string
	name     string
	phone    string
	shopName string
	lat      float64
	lng      float64
	taxID    string
}

var demoRetailers = []demoRetailer{
	{id: "RET-001", name: "Tashkent Central Market", phone: retailerPhone, shopName: "Tashkent Central Market", lat: 41.2995, lng: 69.2401, taxID: "UZ9999001"},
	{id: "RET-002", name: "Sergeli Bazaar", phone: "+998912345678", shopName: "Sergeli Bazaar", lat: 41.2271, lng: 69.2186, taxID: "UZ9999002"},
	{id: "RET-003", name: "Yunusabad Mini-Mart", phone: "+998913456789", shopName: "Yunusabad Mini-Mart", lat: 41.3547, lng: 69.2847, taxID: "UZ9999003"},
	{id: "RET-004", name: "Chorsu Corner Shop", phone: "+998914567890", shopName: "Chorsu Corner Shop", lat: 41.3262, lng: 69.2340, taxID: "UZ9999004"},
}

// ── ADDITIONAL DRIVERS ─────────────────────────────────────────────────────
type demoDriver struct {
	id         string
	name       string
	phone      string
	supplierID string
	driverType string
	vehicle    string
	capacity   int64
}

var demoDrivers = []demoDriver{
	{id: "DRV-001", name: "Rustam Fleet", phone: driverPhone, supplierID: "SUP-001", driverType: "IN_HOUSE", vehicle: "Box Truck", capacity: 12},
	{id: "DRV-002", name: "Davron Express", phone: "+998919876543", supplierID: "SUP-002", driverType: "IN_HOUSE", vehicle: "Transit Van", capacity: 8},
	{id: "DRV-003", name: "Sherzod Logistics", phone: "+998918765432", supplierID: "SUP-003", driverType: "CONTRACTOR", vehicle: "Minivan", capacity: 5},
}

// ── PAYLOADERS (warehouse staff) ───────────────────────────────────────────
type demoPayloader struct {
	id         string
	name       string
	phone      string
	supplierID string
}

var demoPayloaders = []demoPayloader{
	{id: "WRK-001", name: "Anvar Warehouse", phone: payloaderPhone, supplierID: "SUP-001"},
	{id: "WRK-002", name: "Jasur Loading", phone: "+998905552345", supplierID: "SUP-002"},
	{id: "WRK-003", name: "Otabek Dock", phone: "+998905553456", supplierID: "SUP-003"},
	{id: "WRK-004", name: "Mirzo Staging", phone: "+998905554567", supplierID: "SUP-004"},
}

// nilStr returns nil for empty strings (Spanner NULL), otherwise the string value.
func nilStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// vaultEncrypt encrypts plaintext using AES-256-GCM with the VAULT_MASTER_KEY.
// Returns nonce||ciphertext bytes suitable for Spanner BYTES(MAX) columns.
func vaultEncrypt(plaintext []byte) ([]byte, error) {
	keyHex := os.Getenv("VAULT_MASTER_KEY")
	if keyHex == "" {
		return nil, fmt.Errorf("VAULT_MASTER_KEY not set")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("VAULT_MASTER_KEY must be 64 hex chars (32 bytes)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

const (
	supplierPhone  = "+998900001111"
	retailerPhone  = "+998901234567"
	driverPhone    = "+998909876543"
	payloaderPhone = "+998905551234"

	passwordPlain = "password123"
	pinPlain      = "123456"
	payloaderPin  = "654321"
)

const (
	retailerLat = 41.2995
	retailerLng = 69.2401
)

type demoCategory struct {
	id        string
	name      string
	icon      string
	sortOrder int64
}

type demoProduct struct {
	sku             string
	name            string
	desc            string
	categoryID      string
	unitsPerBlock   int64
	price        int64
	stockQty        int64
	palletFootprint float64
}

type demoSupplier struct {
	id                   string
	name                 string
	phone                string
	email                string
	contactPerson        string
	companyRegNumber     string
	billingAddress       string
	primaryCategoryID    string
	operatingCategoryIDs []string
	bankName             string
	accountNumber        string
	cardNumber           string
	paymentGateway       string
	products             []demoProduct
}

var demoCategories = []demoCategory{
	{id: "cat-water", name: "Water", icon: "drop.fill", sortOrder: 10},
	{id: "cat-soft-drinks", name: "Soft Drinks", icon: "sparkles", sortOrder: 20},
	{id: "cat-energy-sports", name: "Energy & Sports Drinks", icon: "bolt.fill", sortOrder: 50},
	{id: "cat-dairy-eggs", name: "Dairy & Eggs", icon: "carton.fill", sortOrder: 60},
	{id: "cat-cheese-deli", name: "Cheese & Deli", icon: "fork.knife", sortOrder: 70},
	{id: "cat-bread-bakery", name: "Bread & Bakery", icon: "birthday.cake.fill", sortOrder: 80},
	{id: "cat-breakfast-cereal", name: "Breakfast & Cereal", icon: "sun.max.fill", sortOrder: 90},
	{id: "cat-rice-pasta-grains", name: "Rice, Pasta & Grains", icon: "takeoutbag.and.cup.and.straw.fill", sortOrder: 100},
	{id: "cat-canned-jarred", name: "Canned & Jarred Goods", icon: "cylinder.fill", sortOrder: 110},
	{id: "cat-condiments-sauces", name: "Condiments & Sauces", icon: "takeoutbag.and.cup.and.straw", sortOrder: 120},
	{id: "cat-oils-spices", name: "Oils, Spices & Seasoning", icon: "flame.fill", sortOrder: 130},
	{id: "cat-snacks-chips", name: "Snacks & Chips", icon: "shippingbox.fill", sortOrder: 140},
	{id: "cat-nuts-dried-fruit", name: "Nuts & Dried Fruit", icon: "leaf.fill", sortOrder: 150},
	{id: "cat-candy-chocolate", name: "Candy & Chocolate", icon: "heart.fill", sortOrder: 160},
	{id: "cat-fresh-fruit", name: "Fresh Fruit", icon: "apple.logo", sortOrder: 180},
	{id: "cat-fresh-vegetables", name: "Fresh Vegetables", icon: "carrot.fill", sortOrder: 190},
	{id: "cat-herbs-salads", name: "Herbs & Salads", icon: "leaf.circle.fill", sortOrder: 200},
	{id: "cat-frozen-food", name: "Frozen Food", icon: "snowflake", sortOrder: 230},
	{id: "cat-personal-care", name: "Personal Care", icon: "person.crop.circle.fill", sortOrder: 270},
	{id: "cat-oral-care", name: "Oral Care", icon: "mouth.fill", sortOrder: 280},
	{id: "cat-skin-care", name: "Skin Care", icon: "sparkles", sortOrder: 300},
	{id: "cat-household-cleaning", name: "Household Cleaning", icon: "sparkles", sortOrder: 360},
	{id: "cat-laundry", name: "Laundry", icon: "washer.fill", sortOrder: 370},
	{id: "cat-paper-hygiene", name: "Paper & Hygiene", icon: "newspaper.fill", sortOrder: 380},
}

var demoSuppliers = []demoSupplier{
	{
		id:                   supplierID,
		name:                 "Lab Beverages Ltd.",
		phone:                supplierPhone,
		email:                "info@labbeverages.uz",
		contactPerson:        "Aziz Karimov",
		companyRegNumber:     "STIR-001-UZ",
		billingAddress:       "12 Amir Temur St, Tashkent 100000",
		bankName:             "Kapitalbank",
		accountNumber:        "20208000900100001010",
		cardNumber:           "8600140012345678",
		paymentGateway:       "PAYME",
		primaryCategoryID:    primaryCategoryID,
		operatingCategoryIDs: []string{"cat-soft-drinks", "cat-water", "cat-energy-sports"},
		products: []demoProduct{
			{sku: sku1, name: "Cola 500ml (50-Pack)", desc: "50-pack PET cola bottles", categoryID: "cat-soft-drinks", unitsPerBlock: 50, price: 250_000, stockQty: 1000, palletFootprint: 0.05},
			{sku: sku2, name: "Still Water 1L (20-Pack)", desc: "20-pack bottled mineral water", categoryID: "cat-water", unitsPerBlock: 20, price: 180_000, stockQty: 900, palletFootprint: 0.08},
			{sku: sku3, name: "Energy Drink 250ml (24-Pack)", desc: "24-pack high-turn energy cans", categoryID: "cat-energy-sports", unitsPerBlock: 24, price: 120_000, stockQty: 800, palletFootprint: 0.04},
			{sku: "SUP1-WATER-02", name: "Sparkling Water 500ml (30-Pack)", desc: "Carbonated mineral water bottles", categoryID: "cat-water", unitsPerBlock: 30, price: 210_000, stockQty: 600, palletFootprint: 0.06},
			{sku: "SUP1-SODA-02", name: "Lemon Soda 330ml (24-Pack)", desc: "Citrus soft drink cans", categoryID: "cat-soft-drinks", unitsPerBlock: 24, price: 145_000, stockQty: 750, palletFootprint: 0.04},
		},
	},
	{
		id:                   "SUP-002",
		name:                 "Silk Road Pantry",
		phone:                "+998900001112",
		email:                "orders@silkroadpantry.uz",
		contactPerson:        "Dildora Rakhimova",
		companyRegNumber:     "STIR-002-UZ",
		billingAddress:       "45 Navoi Ave, Tashkent 100015",
		bankName:             "Uzpromstroybank",
		accountNumber:        "20208000900200002020",
		cardNumber:           "8600140023456789",
		paymentGateway:       "CLICK",
		primaryCategoryID:    "cat-bread-bakery",
		operatingCategoryIDs: []string{"cat-bread-bakery", "cat-breakfast-cereal", "cat-rice-pasta-grains"},
		products: []demoProduct{
			{sku: "SUP2-BREAD-01", name: "Whole Wheat Loaf (18-Pack)", desc: "Fresh-baked whole wheat loaves", categoryID: "cat-bread-bakery", unitsPerBlock: 18, price: 95_000, stockQty: 140, palletFootprint: 0.06},
			{sku: "SUP2-CEREAL-01", name: "Granola Crunch (12-Pack)", desc: "Retail breakfast cereal assortment", categoryID: "cat-breakfast-cereal", unitsPerBlock: 12, price: 150_000, stockQty: 210, palletFootprint: 0.05},
			{sku: "SUP2-RICE-01", name: "Premium Basmati Rice 5kg (6-Pack)", desc: "High-margin rice sacks for pantry aisles", categoryID: "cat-rice-pasta-grains", unitsPerBlock: 6, price: 240_000, stockQty: 120, palletFootprint: 0.11},
			{sku: "SUP2-PASTA-01", name: "Spaghetti 500g (20-Pack)", desc: "Italian-style pasta for pantry shelves", categoryID: "cat-rice-pasta-grains", unitsPerBlock: 20, price: 78_000, stockQty: 300, palletFootprint: 0.04},
			{sku: "SUP2-OATS-01", name: "Rolled Oats 1kg (10-Pack)", desc: "Breakfast cereal staple", categoryID: "cat-breakfast-cereal", unitsPerBlock: 10, price: 125_000, stockQty: 180, palletFootprint: 0.06},
		},
	},
	{
		id:                   "SUP-003",
		name:                 "Fresh Harvest Market",
		phone:                "+998900001113",
		email:                "supply@freshmarket.uz",
		contactPerson:        "Botir Yusupov",
		companyRegNumber:     "STIR-003-UZ",
		billingAddress:       "78 Bunyodkor St, Tashkent 100047",
		bankName:             "Asaka Bank",
		accountNumber:        "20208000900300003030",
		cardNumber:           "8600140034567890",
		paymentGateway:       "PAYME",
		primaryCategoryID:    "cat-fresh-fruit",
		operatingCategoryIDs: []string{"cat-fresh-fruit", "cat-fresh-vegetables", "cat-herbs-salads"},
		products: []demoProduct{
			{sku: "SUP3-FRUIT-01", name: "Royal Gala Apples Crate", desc: "High-turn fruit crate for produce displays", categoryID: "cat-fresh-fruit", unitsPerBlock: 1, price: 175_000, stockQty: 80, palletFootprint: 0.14},
			{sku: "SUP3-VEG-01", name: "Tomato Box 10kg", desc: "Fresh tomatoes boxed for retailer shelves", categoryID: "cat-fresh-vegetables", unitsPerBlock: 1, price: 110_000, stockQty: 95, palletFootprint: 0.12},
			{sku: "SUP3-HERB-01", name: "Romaine & Herb Mix Case", desc: "Leafy greens and herb case", categoryID: "cat-herbs-salads", unitsPerBlock: 1, price: 88_000, stockQty: 60, palletFootprint: 0.09},
			{sku: "SUP3-BANANA-01", name: "Banana Box 18kg", desc: "Premium banana box for high-turn displays", categoryID: "cat-fresh-fruit", unitsPerBlock: 1, price: 145_000, stockQty: 120, palletFootprint: 0.13},
			{sku: "SUP3-ONION-01", name: "Yellow Onion Sack 25kg", desc: "Bulk staple vegetable for retailers", categoryID: "cat-fresh-vegetables", unitsPerBlock: 1, price: 65_000, stockQty: 200, palletFootprint: 0.15},
		},
	},
	{
		id:                   "SUP-004",
		name:                 "Prime Dairy & Deli",
		phone:                "+998900001114",
		email:                "sales@primedairy.uz",
		contactPerson:        "Nodira Toshmatova",
		companyRegNumber:     "STIR-004-UZ",
		billingAddress:       "33 Shota Rustaveli St, Tashkent 100070",
		bankName:             "Ipoteka Bank",
		accountNumber:        "20208000900400004040",
		cardNumber:           "8600140045678901",
		paymentGateway:       "CLICK",
		primaryCategoryID:    "cat-dairy-eggs",
		operatingCategoryIDs: []string{"cat-dairy-eggs", "cat-cheese-deli", "cat-frozen-food"},
		products: []demoProduct{
			{sku: "SUP4-DAIRY-01", name: "Full Cream Milk 1L (12-Pack)", desc: "Chilled dairy line core item", categoryID: "cat-dairy-eggs", unitsPerBlock: 12, price: 132_000, stockQty: 240, palletFootprint: 0.07},
			{sku: "SUP4-DELI-01", name: "Cheddar Blocks 2kg (8-Pack)", desc: "Deli counter cheese blocks", categoryID: "cat-cheese-deli", unitsPerBlock: 8, price: 320_000, stockQty: 70, palletFootprint: 0.1},
			{sku: "SUP4-FROZEN-01", name: "Frozen Fries 2.5kg (6-Pack)", desc: "Frozen aisle staple", categoryID: "cat-frozen-food", unitsPerBlock: 6, price: 210_000, stockQty: 110, palletFootprint: 0.12},
			{sku: "SUP4-EGGS-01", name: "Free Range Eggs 30-Pack", desc: "Premium egg tray for dairy section", categoryID: "cat-dairy-eggs", unitsPerBlock: 1, price: 85_000, stockQty: 350, palletFootprint: 0.05},
			{sku: "SUP4-YOGURT-01", name: "Natural Yogurt 500g (16-Pack)", desc: "Plain yogurt cups for dairy shelves", categoryID: "cat-dairy-eggs", unitsPerBlock: 16, price: 176_000, stockQty: 190, palletFootprint: 0.06},
		},
	},
	{
		id:                   "SUP-005",
		name:                 "HomeCare Depot",
		phone:                "+998900001115",
		email:                "info@homecare.uz",
		contactPerson:        "Kamol Ismoilov",
		companyRegNumber:     "STIR-005-UZ",
		billingAddress:       "99 Chilanzar St, Tashkent 100115",
		bankName:             "Kapitalbank",
		accountNumber:        "20208000900500005050",
		cardNumber:           "8600140056789012",
		paymentGateway:       "PAYME",
		primaryCategoryID:    "cat-household-cleaning",
		operatingCategoryIDs: []string{"cat-household-cleaning", "cat-laundry", "cat-paper-hygiene"},
		products: []demoProduct{
			{sku: "SUP5-CLEAN-01", name: "Floor Cleaner 5L (4-Pack)", desc: "Bulk cleaning liquid for home care aisles", categoryID: "cat-household-cleaning", unitsPerBlock: 4, price: 168_000, stockQty: 150, palletFootprint: 0.09},
			{sku: "SUP5-LAUNDRY-01", name: "Laundry Pods 60ct (8-Pack)", desc: "Concentrated laundry capsule blocks", categoryID: "cat-laundry", unitsPerBlock: 8, price: 280_000, stockQty: 125, palletFootprint: 0.07},
			{sku: "SUP5-PAPER-01", name: "Paper Towels 12-Roll Bale", desc: "High-volume hygiene and paper item", categoryID: "cat-paper-hygiene", unitsPerBlock: 1, price: 145_000, stockQty: 100, palletFootprint: 0.13},
			{sku: "SUP5-SPRAY-01", name: "All-Purpose Spray 750ml (12-Pack)", desc: "Kitchen and bathroom cleaning spray", categoryID: "cat-household-cleaning", unitsPerBlock: 12, price: 198_000, stockQty: 200, palletFootprint: 0.06},
			{sku: "SUP5-TISSUE-01", name: "Toilet Paper 24-Roll Mega Pack", desc: "Bulk tissue paper for hygiene aisle", categoryID: "cat-paper-hygiene", unitsPerBlock: 1, price: 115_000, stockQty: 180, palletFootprint: 0.14},
		},
	},
	{
		id:                   "SUP-006",
		name:                 "BeautyLab Essentials",
		phone:                "+998900001116",
		email:                "orders@beautylab.uz",
		contactPerson:        "Malika Saidova",
		companyRegNumber:     "STIR-006-UZ",
		billingAddress:       "15 Mirabad St, Tashkent 100015",
		bankName:             "Hamkorbank",
		accountNumber:        "20208000900600006060",
		cardNumber:           "8600140067890123",
		paymentGateway:       "CLICK",
		primaryCategoryID:    "cat-personal-care",
		operatingCategoryIDs: []string{"cat-personal-care", "cat-oral-care", "cat-skin-care"},
		products: []demoProduct{
			{sku: "SUP6-PCARE-01", name: "Body Wash 750ml (12-Pack)", desc: "Personal care body wash assortment", categoryID: "cat-personal-care", unitsPerBlock: 12, price: 216_000, stockQty: 160, palletFootprint: 0.06},
			{sku: "SUP6-ORAL-01", name: "Whitening Toothpaste 24-Pack", desc: "Oral care shelf core", categoryID: "cat-oral-care", unitsPerBlock: 24, price: 190_000, stockQty: 220, palletFootprint: 0.05},
			{sku: "SUP6-SKIN-01", name: "Vitamin C Serum 30ml (10-Pack)", desc: "Premium skin care block", categoryID: "cat-skin-care", unitsPerBlock: 10, price: 340_000, stockQty: 90, palletFootprint: 0.04},
			{sku: "SUP6-SHAMPOO-01", name: "Anti-Dandruff Shampoo 400ml (10-Pack)", desc: "Hair care bestseller for personal care aisle", categoryID: "cat-personal-care", unitsPerBlock: 10, price: 195_000, stockQty: 175, palletFootprint: 0.05},
			{sku: "SUP6-BRUSH-01", name: "Bamboo Toothbrush (48-Pack)", desc: "Eco-friendly oral care display unit", categoryID: "cat-oral-care", unitsPerBlock: 48, price: 96_000, stockQty: 300, palletFootprint: 0.03},
		},
	},
	{
		id:                   "SUP-007",
		name:                 "Pantry Reserve",
		phone:                "+998900001117",
		email:                "info@pantryreserve.uz",
		contactPerson:        "Sardor Nazarov",
		companyRegNumber:     "STIR-007-UZ",
		billingAddress:       "22 Mustaqillik St, Tashkent 100000",
		bankName:             "Asaka Bank",
		accountNumber:        "20208000900700007070",
		cardNumber:           "8600140078901234",
		paymentGateway:       "BANK_TRANSFER",
		primaryCategoryID:    "cat-canned-jarred",
		operatingCategoryIDs: []string{"cat-canned-jarred", "cat-condiments-sauces", "cat-oils-spices"},
		products: []demoProduct{
			{sku: "SUP7-CAN-01", name: "Canned Tomatoes 24-Pack", desc: "Pantry canned goods block", categoryID: "cat-canned-jarred", unitsPerBlock: 24, price: 205_000, stockQty: 180, palletFootprint: 0.09},
			{sku: "SUP7-SAUCE-01", name: "Tomato Ketchup 12-Pack", desc: "Condiment bestseller for checkout zones", categoryID: "cat-condiments-sauces", unitsPerBlock: 12, price: 132_000, stockQty: 190, palletFootprint: 0.05},
			{sku: "SUP7-OIL-01", name: "Sunflower Oil 5L (4-Pack)", desc: "Cooking oil pack for pantry aisles", categoryID: "cat-oils-spices", unitsPerBlock: 4, price: 260_000, stockQty: 130, palletFootprint: 0.1},
			{sku: "SUP7-BEAN-01", name: "Canned Chickpeas 24-Pack", desc: "Shelf-stable protein staple", categoryID: "cat-canned-jarred", unitsPerBlock: 24, price: 192_000, stockQty: 200, palletFootprint: 0.09},
			{sku: "SUP7-SPICE-01", name: "Ground Black Pepper 100g (20-Pack)", desc: "Premium spice display unit", categoryID: "cat-oils-spices", unitsPerBlock: 20, price: 160_000, stockQty: 250, palletFootprint: 0.03},
		},
	},
	{
		id:                   "SUP-008",
		name:                 "SnackWorks Distribution",
		phone:                "+998900001118",
		email:                "dispatch@snackworks.uz",
		contactPerson:        "Ulugbek Turaev",
		companyRegNumber:     "STIR-008-UZ",
		billingAddress:       "56 Beruniy St, Tashkent 100100",
		bankName:             "Uzpromstroybank",
		accountNumber:        "20208000900800008080",
		cardNumber:           "8600140089012345",
		paymentGateway:       "PAYME",
		primaryCategoryID:    "cat-snacks-chips",
		operatingCategoryIDs: []string{"cat-snacks-chips", "cat-nuts-dried-fruit", "cat-candy-chocolate"},
		products: []demoProduct{
			{sku: "SUP8-SNACK-01", name: "Potato Chips Mixed Case", desc: "Impulse snack assortment", categoryID: "cat-snacks-chips", unitsPerBlock: 18, price: 198_000, stockQty: 260, palletFootprint: 0.06},
			{sku: "SUP8-NUTS-01", name: "Roasted Almonds 12-Pack", desc: "Healthy snack and dried fruit line", categoryID: "cat-nuts-dried-fruit", unitsPerBlock: 12, price: 225_000, stockQty: 145, palletFootprint: 0.05},
			{sku: "SUP8-CANDY-01", name: "Chocolate Bars Carton", desc: "Checkout-ready candy carton", categoryID: "cat-candy-chocolate", unitsPerBlock: 30, price: 175_000, stockQty: 300, palletFootprint: 0.04},
			{sku: "SUP8-POPCORN-01", name: "Microwave Popcorn 12-Pack", desc: "Ready-to-pop snack display", categoryID: "cat-snacks-chips", unitsPerBlock: 12, price: 108_000, stockQty: 220, palletFootprint: 0.05},
			{sku: "SUP8-TRAIL-01", name: "Trail Mix 500g (10-Pack)", desc: "Healthy nuts and dried fruit blend", categoryID: "cat-nuts-dried-fruit", unitsPerBlock: 10, price: 185_000, stockQty: 170, palletFootprint: 0.04},
		},
	},
}

var demoCategoryByID = func() map[string]demoCategory {
	index := make(map[string]demoCategory, len(demoCategories))
	for _, category := range demoCategories {
		index[category.id] = category
	}
	return index
}()

func main() {
	log.Println("══════════════════════════════════════════════════════════════")
	log.Println("  THE LAB INDUSTRIES — E2E SEED SCRIPT")
	log.Println("══════════════════════════════════════════════════════════════")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[SEED] Config load failed: %v", err)
	}

	ctx := context.Background()

	// ── Spanner Client ─────────────────────────────────────────────────────
	emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST")
	if emulatorAddr == "" {
		emulatorAddr = "localhost:9010"
		os.Setenv("SPANNER_EMULATOR_HOST", emulatorAddr)
	}

	opts := []option.ClientOption{
		option.WithEndpoint(emulatorAddr),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}

	dbName := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)

	spannerClient, err := spanner.NewClient(ctx, dbName, opts...)
	if err != nil {
		log.Fatalf("[SEED] Spanner client failed: %v", err)
	}
	defer spannerClient.Close()
	log.Printf("[SEED] Spanner connected → %s", dbName)

	// ── Redis Client ───────────────────────────────────────────────────────
	redisAddr := cfg.RedisAddress
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[SEED] Redis ping failed at %s: %v", redisAddr, err)
	}
	defer rdb.Close()
	log.Printf("[SEED] Redis connected → %s", redisAddr)

	// ── Phase 0: Apply schema migrations (columns added by main.go) ────────
	applyMigrations(ctx, dbName, opts)

	// ── Phase 1: Truncate existing data (idempotency) ──────────────────────
	truncateTables(ctx, spannerClient)

	// ── Phase 2: Bcrypt hashes ─────────────────────────────────────────────
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(passwordPlain), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[SEED] bcrypt password hash failed: %v", err)
	}
	pinHash, err := bcrypt.GenerateFromPassword([]byte(pinPlain), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[SEED] bcrypt PIN hash failed: %v", err)
	}
	payloaderPinHash, err := bcrypt.GenerateFromPassword([]byte(payloaderPin), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[SEED] bcrypt payloader PIN hash failed: %v", err)
	}
	log.Println("[SEED] Bcrypt hashes generated.")

	// ── Phase 3: Insert seed data ──────────────────────────────────────────
	seedSpanner(ctx, spannerClient, string(passwordHash), string(pinHash), string(payloaderPinHash))

	// ── Phase 4: Redis GEO index ───────────────────────────────────────────
	seedRedisGeo(ctx, rdb)

	// ── Phase 5: Kafka topics ──────────────────────────────────────────────
	seedKafkaTopics(cfg.KafkaBrokerAddress)

	log.Println("══════════════════════════════════════════════════════════════")
	log.Println("  E2E SEED COMPLETE — CREDENTIALS MATRIX")
	log.Println("══════════════════════════════════════════════════════════════")
	log.Println("  ── SUPPLIERS (Admin Portal) ─── password: password123")
	for _, s := range demoSuppliers {
		log.Printf("    %s  %-30s  %s", s.id, s.name, s.phone)
	}
	log.Println("  ── RETAILERS (Mobile App) ───── password: password123")
	for _, r := range demoRetailers {
		log.Printf("    %s  %-30s  %s", r.id, r.name, r.phone)
	}
	log.Println("  ── DRIVERS (Mobile App) ─────── PIN: 123456")
	for _, d := range demoDrivers {
		log.Printf("    %s  %-30s  %s  → %s", d.id, d.name, d.phone, d.supplierID)
	}
	log.Println("  ── PAYLOADERS (Tablet) ──────── PIN: 654321")
	for _, p := range demoPayloaders {
		log.Printf("    %s  %-30s  %s  → %s", p.id, p.name, p.phone, p.supplierID)
	}
	log.Println("══════════════════════════════════════════════════════════════")
}

func applyMigrations(ctx context.Context, dbName string, opts []option.ClientOption) {
	adminClient, err := database.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		log.Printf("[MIGRATE] Admin client failed: %v (skipping migrations)", err)
		return
	}
	defer adminClient.Close()

	// These columns are normally added by main.go on first boot.
	// The seed script needs them to exist before inserting.
	alterStmts := []string{
		"ALTER TABLE Drivers ADD COLUMN Phone STRING(20)",
		"ALTER TABLE Drivers ADD COLUMN PinHash STRING(MAX)",
		"ALTER TABLE Drivers ADD COLUMN SupplierId STRING(36)",
		"ALTER TABLE Drivers ADD COLUMN DriverType STRING(20)",
		"ALTER TABLE Drivers ADD COLUMN VehicleType STRING(50)",
		"ALTER TABLE Drivers ADD COLUMN LicensePlate STRING(30)",
		"ALTER TABLE Drivers ADD COLUMN IsActive BOOL",
		"ALTER TABLE Retailers ADD COLUMN Phone STRING(20)",
		"ALTER TABLE Retailers ADD COLUMN PasswordHash STRING(MAX)",
		"ALTER TABLE Retailers ADD COLUMN ShopName STRING(MAX)",
		"ALTER TABLE Retailers ADD COLUMN FcmToken STRING(MAX)",
		"ALTER TABLE Retailers ADD COLUMN TelegramChatId STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN Phone STRING(20)",
		"ALTER TABLE Suppliers ADD COLUMN PasswordHash STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN TaxId STRING(64)",
		"ALTER TABLE Suppliers ADD COLUMN IsConfigured BOOL",
		"ALTER TABLE Suppliers ADD COLUMN OperatingCategories ARRAY<STRING(36)>",
		"ALTER TABLE Suppliers ADD COLUMN Email STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN ContactPerson STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN CompanyRegNumber STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN BillingAddress STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN BankName STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN AccountNumber STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN CardNumber STRING(MAX)",
		"ALTER TABLE Suppliers ADD COLUMN PaymentGateway STRING(20)",
		"CREATE TABLE PlatformCategories (CategoryId STRING(36) NOT NULL, DisplayName STRING(MAX) NOT NULL, IconUrl STRING(MAX), DisplayOrder INT64 NOT NULL) PRIMARY KEY (CategoryId)",
		// Auto-Dispatch Engine: dimensional columns for bin-packing
		"ALTER TABLE Drivers ADD COLUMN MaxPalletCapacity INT64",
		"ALTER TABLE SupplierProducts ADD COLUMN PalletFootprint FLOAT64",
		// Phase 11: Scheduled Orders
		"ALTER TABLE Orders ADD COLUMN RequestedDeliveryDate TIMESTAMP",
		// Phase 12: Optimistic Concurrency Control + Freeze Locks
		"ALTER TABLE Orders ADD COLUMN Version INT64 NOT NULL DEFAULT (1)",
		"ALTER TABLE Orders ADD COLUMN LockedUntil TIMESTAMP",
		"CREATE TABLE WarehouseStaff (WorkerId STRING(36) NOT NULL, SupplierId STRING(36) NOT NULL, Name STRING(MAX) NOT NULL, Phone STRING(20) NOT NULL, PinHash STRING(MAX) NOT NULL, IsActive BOOL NOT NULL, CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)) PRIMARY KEY (WorkerId)",
		"CREATE INDEX Idx_WarehouseStaff_BySupplierId ON WarehouseStaff(SupplierId)",
		"CREATE INDEX Idx_WarehouseStaff_ByPhone ON WarehouseStaff(Phone)",
		// Widen State column to fit PENDING_CASH_COLLECTION (23 chars)
		"ALTER TABLE Orders ALTER COLUMN State STRING(30) NOT NULL",
	}

	for _, stmt := range alterStmts {
		op, ddlErr := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   dbName,
			Statements: []string{stmt},
		})
		if ddlErr != nil {
			// Column already exists — expected on re-run
			continue
		}
		op.Wait(ctx)
	}
	log.Println("[MIGRATE] Schema migrations applied.")
}

func truncateTables(ctx context.Context, client *spanner.Client) {
	tables := []string{
		"SupplierPaymentConfigs",
		"WarehouseStaff",
		"SupplierInventory",
		"SupplierProducts",
		"PlatformCategories",
		"RetailerSuppliers",
		"RetailerProductSettings",
		"RetailerSupplierSettings",
		"RetailerGlobalSettings",
		"OrderLineItems",
		"LedgerEntries",
		"LedgerAnomalies",
		"Orders",
		"Drivers",
		"Retailers",
		"Suppliers",
		"Categories",
	}

	for _, t := range tables {
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			iter := txn.Query(ctx, spanner.Statement{
				SQL: fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", t),
			})
			_, err := iter.Next()
			iter.Stop()
			if err == iterator.Done {
				return nil
			}

			stmt := spanner.Statement{SQL: fmt.Sprintf("DELETE FROM %s WHERE TRUE", t)}
			count, err := txn.Update(ctx, stmt)
			if err != nil {
				return err
			}
			log.Printf("[TRUNCATE] %s → %d rows deleted", t, count)
			return nil
		})
		if err != nil {
			log.Printf("[TRUNCATE] %s skipped: %v", t, err)
		}
	}
}

func seedSpanner(ctx context.Context, client *spanner.Client, passwordHash, pinHash, payloaderPinHash string) {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}

		// ── CANONICAL DEMO CATEGORY MATRIX ─────────────────────────────────
		for _, category := range demoCategories {
			mutations = append(mutations,
				spanner.InsertOrUpdate("PlatformCategories",
					[]string{"CategoryId", "DisplayName", "IconUrl", "DisplayOrder"},
					[]interface{}{category.id, category.name, category.icon, category.sortOrder}),
				spanner.InsertOrUpdate("Categories",
					[]string{"CategoryId", "Name", "Icon", "SortOrder", "CreatedAt"},
					[]interface{}{category.id, category.name, category.icon, category.sortOrder, spanner.CommitTimestamp}),
			)
		}

		// ── SUPPLIERS + PRODUCTS + INVENTORY ───────────────────────────────
		for _, supplier := range demoSuppliers {
			primaryCategoryName := demoCategoryByID[supplier.primaryCategoryID].name
			mutations = append(mutations, spanner.Insert("Suppliers",
				[]string{"SupplierId", "Name", "Category", "Phone", "Email", "PasswordHash",
					"TaxId", "ContactPerson", "CompanyRegNumber", "BillingAddress",
					"IsConfigured", "OperatingCategories",
					"BankName", "AccountNumber", "CardNumber", "PaymentGateway", "CreatedAt"},
				[]interface{}{supplier.id, supplier.name, primaryCategoryName, supplier.phone, supplier.email, passwordHash,
					strings.ReplaceAll(supplier.id, "SUP-", "UZ-TAX-"), supplier.contactPerson, supplier.companyRegNumber, supplier.billingAddress,
					true, supplier.operatingCategoryIDs,
					supplier.bankName, supplier.accountNumber, supplier.cardNumber, supplier.paymentGateway, spanner.CommitTimestamp}))

			mutations = append(mutations, spanner.Insert("RetailerSuppliers",
				[]string{"RetailerId", "SupplierId", "AddedAt"},
				[]interface{}{retailerID, supplier.id, spanner.CommitTimestamp}))

			for _, product := range supplier.products {
				mutations = append(mutations, spanner.Insert("SupplierProducts",
					[]string{"SkuId", "SupplierId", "Name", "Description", "ImageUrl",
						"SellByBlock", "UnitsPerBlock", "BasePrice", "PalletFootprint", "IsActive", "CategoryId", "CreatedAt"},
					[]interface{}{product.sku, supplier.id, product.name, product.desc,
						"https://placehold.co/150",
						true, product.unitsPerBlock, product.price, product.palletFootprint, true, product.categoryID, spanner.CommitTimestamp}))

				mutations = append(mutations, spanner.Insert("SupplierInventory",
					[]string{"ProductId", "SupplierId", "QuantityAvailable", "UpdatedAt"},
					[]interface{}{product.sku, supplier.id, product.stockQty, spanner.CommitTimestamp}))
			}
		}

		// ── RETAILERS (bcrypt PasswordHash + GPS WKT) ──────────────────────
		for _, ret := range demoRetailers {
			shopWKT := fmt.Sprintf("POINT(%f %f)", ret.lng, ret.lat)
			mutations = append(mutations, spanner.Insert("Retailers",
				[]string{"RetailerId", "Name", "Phone", "ShopName", "ShopLocation",
					"TaxIdentificationNumber", "Status", "PasswordHash", "CreatedAt"},
				[]interface{}{ret.id, ret.name, ret.phone, ret.shopName,
					shopWKT, ret.taxID, "VERIFIED", passwordHash, spanner.CommitTimestamp}))

			mutations = append(mutations, spanner.Insert("RetailerGlobalSettings",
				[]string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"},
				[]interface{}{ret.id, false, spanner.CommitTimestamp}))

			// Link each retailer to all suppliers
			if ret.id != retailerID { // RET-001 already linked above in supplier loop
				for _, supplier := range demoSuppliers {
					mutations = append(mutations, spanner.Insert("RetailerSuppliers",
						[]string{"RetailerId", "SupplierId", "AddedAt"},
						[]interface{}{ret.id, supplier.id, spanner.CommitTimestamp}))
				}
			}
		}

		// ── DRIVERS (bcrypt PinHash, fleet-linked) ─────────────────────────
		for _, drv := range demoDrivers {
			mutations = append(mutations, spanner.Insert("Drivers",
				[]string{"DriverId", "Name", "Phone", "PinHash", "SupplierId",
					"DriverType", "VehicleType", "MaxPalletCapacity", "IsActive", "CreatedAt"},
				[]interface{}{drv.id, drv.name, drv.phone, pinHash, drv.supplierID,
					drv.driverType, drv.vehicle, drv.capacity, true, spanner.CommitTimestamp}))
		}

		// ── PAYLOADERS (warehouse workers, PIN-auth) ──────────────────────────
		for _, pl := range demoPayloaders {
			mutations = append(mutations, spanner.Insert("WarehouseStaff",
				[]string{"WorkerId", "SupplierId", "Name", "Phone", "PinHash", "IsActive", "CreatedAt"},
				[]interface{}{pl.id, pl.supplierID, pl.name, pl.phone, payloaderPinHash, true, spanner.CommitTimestamp}))
		}

		// ── SUPPLIER PAYMENT CONFIGS (AES-256-GCM encrypted vault) ─────────
		type paymentConfig struct {
			configID   string
			supplierID string
			gateway    string
			merchantID string
			serviceID  string
			secretKey  string
		}
		paymentConfigs := []paymentConfig{
			{"PCFG-001", supplierID, "PAYME", "payme_merchant_001", "", "payme_secret_key_SUP001_live"},
			{"PCFG-002", supplierID, "CLICK", "click_merchant_001", "click_svc_001", "click_secret_key_SUP001_live"},
		}
		for _, pc := range paymentConfigs {
			encrypted, err := vaultEncrypt([]byte(pc.secretKey))
			if err != nil {
				log.Printf("[SEED] VAULT ENCRYPT WARNING for %s: %v (skipping payment config)", pc.configID, err)
				continue
			}
			cols := []string{"ConfigId", "SupplierId", "GatewayName", "MerchantId", "SecretKey", "IsActive", "CreatedAt"}
			vals := []interface{}{pc.configID, pc.supplierID, pc.gateway, pc.merchantID, encrypted, true, spanner.CommitTimestamp}
			mutations = append(mutations, spanner.Insert("SupplierPaymentConfigs", cols, vals))
		}

		// ═══════════════════════════════════════════════════════════════════
		// COMPREHENSIVE ORDER MATRIX — Multi-retailer, multi-supplier, all states
		// ═══════════════════════════════════════════════════════════════════
		type seedOrder struct {
			id            string
			retailerID    string
			supplierID    string
			driverID      string
			routeID       string
			state         string
			paymentStatus string
			amount     int64
			orderSource   string
			deliveryToken string // empty = NULL (pre-dispatch); set for dispatched orders
		}
		seedOrders := []seedOrder{
			// ── RET-001 orders (Tashkent Central Market) ── various suppliers
			{"ORD-SEED-001", "RET-001", "SUP-001", "DRV-001", "TRUCK-TASH-01", "PENDING", "PENDING", 620_000, "RETAILER_APP", ""},
			{"ORD-SEED-002", "RET-001", "SUP-001", "DRV-001", "TRUCK-TASH-01", "LOADED", "PENDING", 300_000, "RETAILER_APP", "a1b2c3d4e5f60002"},
			{"ORD-SEED-003", "RET-001", "SUP-002", "DRV-002", "TRUCK-TASH-02", "IN_TRANSIT", "PENDING", 485_000, "RETAILER_APP", "a1b2c3d4e5f60003"},
			{"ORD-SEED-004", "RET-001", "SUP-003", "DRV-003", "TRUCK-TASH-03", "ARRIVED", "AWAITING_GATEWAY_WEBHOOK", 373_000, "RETAILER_APP", "a1b2c3d4e5f60004"},
			{"ORD-SEED-005", "RET-001", "SUP-004", "DRV-001", "TRUCK-TASH-01", "COMPLETED", "PAID", 662_000, "ADMIN_PORTAL", "a1b2c3d4e5f60005"},
			{"ORD-SEED-006", "RET-001", "SUP-008", "", "", "CANCELLED", "FAILED", 198_000, "RETAILER_APP", ""},

			// ── RET-002 orders (Sergeli Bazaar) ── various suppliers
			{"ORD-SEED-007", "RET-002", "SUP-001", "DRV-001", "TRUCK-TASH-01", "PENDING", "PENDING", 430_000, "RETAILER_APP", ""},
			{"ORD-SEED-008", "RET-002", "SUP-005", "DRV-002", "TRUCK-TASH-02", "IN_TRANSIT", "PENDING", 593_000, "ADMIN_PORTAL", "b2c3d4e5f6a10008"},
			{"ORD-SEED-009", "RET-002", "SUP-006", "DRV-003", "TRUCK-TASH-03", "ARRIVING", "PENDING", 746_000, "RETAILER_APP", "b2c3d4e5f6a10009"},
			{"ORD-SEED-010", "RET-002", "SUP-007", "DRV-001", "TRUCK-TASH-01", "COMPLETED", "PAID", 597_000, "RETAILER_APP", "b2c3d4e5f6a10010"},
			{"ORD-SEED-011", "RET-002", "SUP-002", "", "", "SCHEDULED", "PENDING", 240_000, "ADMIN_PORTAL", ""},
			{"ORD-SEED-012", "RET-002", "SUP-003", "DRV-002", "TRUCK-TASH-02", "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", 285_000, "RETAILER_APP", "b2c3d4e5f6a10012"},

			// ── RET-003 orders (Yunusabad Mini-Mart) ── various suppliers
			{"ORD-SEED-013", "RET-003", "SUP-004", "DRV-001", "TRUCK-TASH-01", "LOADED", "PENDING", 452_000, "RETAILER_APP", "c3d4e5f6a1b20013"},
			{"ORD-SEED-014", "RET-003", "SUP-001", "DRV-001", "TRUCK-TASH-01", "IN_TRANSIT", "PENDING", 250_000, "ADMIN_PORTAL", "c3d4e5f6a1b20014"},
			{"ORD-SEED-015", "RET-003", "SUP-008", "DRV-003", "TRUCK-TASH-03", "COMPLETED", "PAID", 398_000, "RETAILER_APP", "c3d4e5f6a1b20015"},
			{"ORD-SEED-016", "RET-003", "SUP-005", "DRV-002", "TRUCK-TASH-02", "PENDING_CASH_COLLECTION", "PENDING_CASH_COLLECTION", 425_000, "RETAILER_APP", "c3d4e5f6a1b20016"},
			{"ORD-SEED-017", "RET-003", "SUP-006", "", "", "PENDING_REVIEW", "PENDING", 530_000, "AI_FORECAST", ""},
			{"ORD-SEED-018", "RET-003", "SUP-007", "", "", "CANCELLED", "FAILED", 260_000, "RETAILER_APP", ""},

			// ── RET-004 orders (Chorsu Corner Shop) ── various suppliers
			{"ORD-SEED-019", "RET-004", "SUP-001", "DRV-001", "TRUCK-TASH-01", "PENDING", "PENDING", 370_000, "RETAILER_APP", ""},
			{"ORD-SEED-020", "RET-004", "SUP-002", "DRV-002", "TRUCK-TASH-02", "LOADED", "PENDING", 335_000, "ADMIN_PORTAL", "d4e5f6a1b2c30020"},
			{"ORD-SEED-021", "RET-004", "SUP-003", "DRV-003", "TRUCK-TASH-03", "ARRIVED", "AWAITING_GATEWAY_WEBHOOK", 263_000, "RETAILER_APP", "d4e5f6a1b2c30021"},
			{"ORD-SEED-022", "RET-004", "SUP-004", "DRV-001", "TRUCK-TASH-01", "COMPLETED", "PAID", 320_000, "RETAILER_APP", "d4e5f6a1b2c30022"},
			{"ORD-SEED-023", "RET-004", "SUP-005", "DRV-002", "TRUCK-TASH-02", "QUARANTINE", "PENDING", 168_000, "RETAILER_APP", "d4e5f6a1b2c30023"},
			{"ORD-SEED-024", "RET-004", "SUP-008", "DRV-003", "TRUCK-TASH-03", "IN_TRANSIT", "PENDING", 573_000, "RETAILER_APP", "d4e5f6a1b2c30024"},
		}
		orderCols := []string{"OrderId", "RetailerId", "SupplierId", "DriverId", "State",
			"Amount", "PaymentStatus", "RouteId", "OrderSource", "DeliveryToken", "CreatedAt"}
		for _, o := range seedOrders {
			vals := []interface{}{o.id, o.retailerID, o.supplierID, nilStr(o.driverID), o.state,
				o.amount, o.paymentStatus, nilStr(o.routeID), o.orderSource, nilStr(o.deliveryToken), spanner.CommitTimestamp}
			mutations = append(mutations, spanner.Insert("Orders", orderCols, vals))
		}

		// ── ORDER LINE ITEMS ───────────────────────────────────────────────
		type seedLineItem struct {
			id, orderId, skuId string
			qty, unitPrice     int64
			status             string
		}
		seedItems := []seedLineItem{
			// ORD-SEED-001: SUP-001 products → RET-001 (PENDING)
			{"LI-001", "ORD-SEED-001", "COKE-500-50", 2, 250_000, "PENDING"},
			{"LI-002", "ORD-SEED-001", "FANTA-CAN-24", 1, 120_000, "PENDING"},
			// ORD-SEED-002: SUP-001 → RET-001 (LOADED)
			{"LI-003", "ORD-SEED-002", "SPRITE-1L-20", 1, 180_000, "PENDING"},
			{"LI-004", "ORD-SEED-002", "FANTA-CAN-24", 1, 120_000, "PENDING"},
			// ORD-SEED-003: SUP-002 → RET-001 (IN_TRANSIT)
			{"LI-005", "ORD-SEED-003", "SUP2-BREAD-01", 2, 95_000, "PENDING"},
			{"LI-006", "ORD-SEED-003", "SUP2-CEREAL-01", 1, 150_000, "PENDING"},
			{"LI-007", "ORD-SEED-003", "SUP2-RICE-01", 1, 240_000, "PENDING"},
			// ORD-SEED-004: SUP-003 → RET-001 (ARRIVED)
			{"LI-008", "ORD-SEED-004", "SUP3-FRUIT-01", 2, 175_000, "PENDING"},
			{"LI-009", "ORD-SEED-004", "SUP3-HERB-01", 1, 88_000, "PENDING"},
			// ORD-SEED-005: SUP-004 → RET-001 (COMPLETED)
			{"LI-010", "ORD-SEED-005", "SUP4-DAIRY-01", 3, 132_000, "DELIVERED"},
			{"LI-011", "ORD-SEED-005", "SUP4-DELI-01", 1, 320_000, "DELIVERED"},
			// ORD-SEED-006: SUP-008 → RET-001 (CANCELLED)
			{"LI-012", "ORD-SEED-006", "SUP8-SNACK-01", 1, 198_000, "REJECTED_DAMAGED"},
			// ORD-SEED-007: SUP-001 → RET-002 (PENDING)
			{"LI-013", "ORD-SEED-007", "COKE-500-50", 1, 250_000, "PENDING"},
			{"LI-014", "ORD-SEED-007", "SPRITE-1L-20", 1, 180_000, "PENDING"},
			// ORD-SEED-008: SUP-005 → RET-002 (IN_TRANSIT)
			{"LI-015", "ORD-SEED-008", "SUP5-CLEAN-01", 2, 168_000, "PENDING"},
			{"LI-016", "ORD-SEED-008", "SUP5-LAUNDRY-01", 1, 280_000, "PENDING"},
			// ORD-SEED-009: SUP-006 → RET-002 (ARRIVING)
			{"LI-017", "ORD-SEED-009", "SUP6-PCARE-01", 2, 216_000, "PENDING"},
			{"LI-018", "ORD-SEED-009", "SUP6-ORAL-01", 1, 190_000, "PENDING"},
			{"LI-019", "ORD-SEED-009", "SUP6-SKIN-01", 1, 340_000, "PENDING"},
			// ORD-SEED-010: SUP-007 → RET-002 (COMPLETED)
			{"LI-020", "ORD-SEED-010", "SUP7-CAN-01", 1, 205_000, "DELIVERED"},
			{"LI-021", "ORD-SEED-010", "SUP7-SAUCE-01", 1, 132_000, "DELIVERED"},
			{"LI-022", "ORD-SEED-010", "SUP7-OIL-01", 1, 260_000, "DELIVERED"},
			// ORD-SEED-011: SUP-002 → RET-002 (SCHEDULED)
			{"LI-023", "ORD-SEED-011", "SUP2-RICE-01", 1, 240_000, "PENDING"},
			// ORD-SEED-012: SUP-003 → RET-002 (AWAITING_PAYMENT)
			{"LI-024", "ORD-SEED-012", "SUP3-VEG-01", 2, 110_000, "DELIVERED"},
			{"LI-025", "ORD-SEED-012", "SUP3-HERB-01", 1, 88_000, "DELIVERED"},
			// ORD-SEED-013: SUP-004 → RET-003 (LOADED)
			{"LI-026", "ORD-SEED-013", "SUP4-DAIRY-01", 2, 132_000, "PENDING"},
			{"LI-027", "ORD-SEED-013", "SUP4-FROZEN-01", 1, 210_000, "PENDING"},
			// ORD-SEED-014: SUP-001 → RET-003 (IN_TRANSIT)
			{"LI-028", "ORD-SEED-014", "COKE-500-50", 1, 250_000, "PENDING"},
			// ORD-SEED-015: SUP-008 → RET-003 (COMPLETED)
			{"LI-029", "ORD-SEED-015", "SUP8-SNACK-01", 1, 198_000, "DELIVERED"},
			{"LI-030", "ORD-SEED-015", "SUP8-NUTS-01", 1, 225_000, "DELIVERED"},
			// ORD-SEED-016: SUP-005 → RET-003 (PENDING_CASH_COLLECTION)
			{"LI-031", "ORD-SEED-016", "SUP5-PAPER-01", 2, 145_000, "DELIVERED"},
			{"LI-032", "ORD-SEED-016", "SUP5-CLEAN-01", 1, 168_000, "DELIVERED"},
			// ORD-SEED-017: SUP-006 → RET-003 (PENDING_REVIEW, from AI)
			{"LI-033", "ORD-SEED-017", "SUP6-SKIN-01", 1, 340_000, "PENDING"},
			{"LI-034", "ORD-SEED-017", "SUP6-ORAL-01", 1, 190_000, "PENDING"},
			// ORD-SEED-018: SUP-007 → RET-003 (CANCELLED)
			{"LI-035", "ORD-SEED-018", "SUP7-OIL-01", 1, 260_000, "REJECTED_DAMAGED"},
			// ORD-SEED-019: SUP-001 → RET-004 (PENDING)
			{"LI-036", "ORD-SEED-019", "COKE-500-50", 1, 250_000, "PENDING"},
			{"LI-037", "ORD-SEED-019", "FANTA-CAN-24", 1, 120_000, "PENDING"},
			// ORD-SEED-020: SUP-002 → RET-004 (LOADED)
			{"LI-038", "ORD-SEED-020", "SUP2-BREAD-01", 2, 95_000, "PENDING"},
			{"LI-039", "ORD-SEED-020", "SUP2-CEREAL-01", 1, 150_000, "PENDING"},
			// ORD-SEED-021: SUP-003 → RET-004 (ARRIVED)
			{"LI-040", "ORD-SEED-021", "SUP3-FRUIT-01", 1, 175_000, "PENDING"},
			{"LI-041", "ORD-SEED-021", "SUP3-HERB-01", 1, 88_000, "PENDING"},
			// ORD-SEED-022: SUP-004 → RET-004 (COMPLETED)
			{"LI-042", "ORD-SEED-022", "SUP4-DELI-01", 1, 320_000, "DELIVERED"},
			// ORD-SEED-023: SUP-005 → RET-004 (QUARANTINE)
			{"LI-043", "ORD-SEED-023", "SUP5-CLEAN-01", 1, 168_000, "PENDING"},
			// ORD-SEED-024: SUP-008 → RET-004 (IN_TRANSIT)
			{"LI-044", "ORD-SEED-024", "SUP8-SNACK-01", 1, 198_000, "PENDING"},
			{"LI-045", "ORD-SEED-024", "SUP8-CANDY-01", 2, 175_000, "PENDING"},
			{"LI-046", "ORD-SEED-024", "SUP8-NUTS-01", 1, 225_000, "PENDING"},
		}
		for _, li := range seedItems {
			mutations = append(mutations, spanner.Insert("OrderLineItems",
				[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Status"},
				[]interface{}{li.id, li.orderId, li.skuId, li.qty, li.unitPrice, li.status}))
		}

		// ── MASTER INVOICES (for completed orders) ─────────────────────────
		type seedInvoice struct {
			invoiceID   string
			retailerID  string
			total    int64
			state       string
			orderID     string
			paymentMode string
		}
		seedInvoices := []seedInvoice{
			{"INV-001", "RET-001", 662_000, "PAID", "ORD-SEED-005", "ELECTRONIC"},
			{"INV-002", "RET-002", 597_000, "PAID", "ORD-SEED-010", "ELECTRONIC"},
			{"INV-003", "RET-003", 398_000, "PAID", "ORD-SEED-015", "ELECTRONIC"},
			{"INV-004", "RET-003", 425_000, "PENDING_COLLECTION", "ORD-SEED-016", "CASH"},
			{"INV-005", "RET-004", 320_000, "PAID", "ORD-SEED-022", "CASH"},
		}
		for _, inv := range seedInvoices {
			mutations = append(mutations, spanner.Insert("MasterInvoices",
				[]string{"InvoiceId", "RetailerId", "Total", "State", "OrderId", "PaymentMode", "CreatedAt"},
				[]interface{}{inv.invoiceID, inv.retailerID, inv.total, inv.state, inv.orderID, inv.paymentMode, spanner.CommitTimestamp}))
		}

		return txn.BufferWrite(mutations)
	})

	if err != nil {
		log.Fatalf("[SEED] Spanner insert failed: %v", err)
	}

	totalProducts := 0
	for _, supplier := range demoSuppliers {
		totalProducts += len(supplier.products)
	}

	log.Println("[SEED] Spanner data injected:")
	log.Printf("  • %d Suppliers  linked to retailer discovery", len(demoSuppliers))
	log.Printf("  • %d Categories populated across discovery", len(demoCategories))
	log.Printf("  • %d SKUs       distributed across supplier catalogs", totalProducts)
	log.Printf("  • %d Retailers  (all VERIFIED, GPS-indexed)", len(demoRetailers))
	log.Printf("  • %d Drivers    across %d suppliers", len(demoDrivers), 3)
	log.Printf("  • %d Payloaders across %d supplier warehouses", len(demoPayloaders), 4)
	log.Println("  • 24 Orders     (12 states × multi-supplier × multi-retailer)")
	log.Println("  • 46 Line Items (realistic SKU distribution)")
	log.Println("  • 5  Invoices   (PAID + PENDING_COLLECTION)")
	log.Println("  • 2  Payment Configs (PAYME + CLICK, AES-256-GCM encrypted)")
}

func seedRedisGeo(ctx context.Context, rdb *redis.Client) {
	for _, ret := range demoRetailers {
		if err := rdb.GeoAdd(ctx, geoKey, &redis.GeoLocation{
			Name:      "r:" + ret.id,
			Longitude: ret.lng,
			Latitude:  ret.lat,
		}).Err(); err != nil {
			log.Fatalf("[SEED] Redis GEOADD failed for %s: %v", ret.id, err)
		}
		log.Printf("[SEED] Redis GEOADD → %s : r:%s @ (%.4f, %.4f)",
			geoKey, ret.id, ret.lat, ret.lng)
	}
}

func seedKafkaTopics(brokerAddress string) {
	if brokerAddress == "" {
		brokerAddress = "localhost:9092"
	}

	topics := []string{
		"lab-logistics-events",
		"lab-logistics-events-dlq",
		"lab-driver-sync-events",
		"orders.completed",
		"orders.dispatched",
	}

	conn, err := kafka.Dial("tcp", brokerAddress)
	if err != nil {
		log.Printf("[SEED] Kafka connection failed at %s: %v (topics skipped)", brokerAddress, err)
		return
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		log.Printf("[SEED] Kafka controller fetch failed: %v (topics skipped)", err)
		return
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		log.Printf("[SEED] Kafka controller dial failed: %v (topics skipped)", err)
		return
	}
	defer controllerConn.Close()

	for _, topic := range topics {
		err := controllerConn.CreateTopics(kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Printf("[SEED] Kafka topic '%s' creation failed: %v", topic, err)
		} else {
			log.Printf("[SEED] Kafka topic '%s' ready.", topic)
		}
	}
}
