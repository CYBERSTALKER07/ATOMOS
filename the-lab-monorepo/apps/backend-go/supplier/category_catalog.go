package supplier

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloud.google.com/go/spanner"
)

type canonicalCategory struct {
	ID          string
	DisplayName string
	Icon        string
	SortOrder   int64
}

var canonicalCategories = []canonicalCategory{
	// ── Beverages ──
	{ID: "cat-water", DisplayName: "Water", Icon: "drop.fill", SortOrder: 10},
	{ID: "cat-sparkling-water", DisplayName: "Sparkling & Flavored Water", Icon: "bubbles.and.sparkles.fill", SortOrder: 11},
	{ID: "cat-soft-drinks", DisplayName: "Soft Drinks & Soda", Icon: "sparkles", SortOrder: 20},
	{ID: "cat-juice", DisplayName: "Juice & Nectars", Icon: "carton.fill", SortOrder: 30},
	{ID: "cat-tea-coffee", DisplayName: "Tea & Coffee", Icon: "cup.and.saucer.fill", SortOrder: 40},
	{ID: "cat-energy-sports", DisplayName: "Energy & Sports Drinks", Icon: "bolt.fill", SortOrder: 50},
	{ID: "cat-powdered-drinks", DisplayName: "Powdered Drink Mixes", Icon: "mug.fill", SortOrder: 51},
	{ID: "cat-kombucha", DisplayName: "Kombucha & Probiotic Drinks", Icon: "leaf.fill", SortOrder: 52},
	{ID: "cat-plant-milk", DisplayName: "Plant-Based Milk", Icon: "leaf.circle.fill", SortOrder: 53},
	// ── Dairy & Refrigerated ──
	{ID: "cat-milk", DisplayName: "Milk & Cream", Icon: "carton.fill", SortOrder: 60},
	{ID: "cat-yogurt", DisplayName: "Yogurt & Kefir", Icon: "cup.and.saucer.fill", SortOrder: 61},
	{ID: "cat-cheese", DisplayName: "Cheese", Icon: "fork.knife", SortOrder: 70},
	{ID: "cat-butter-margarine", DisplayName: "Butter & Margarine", Icon: "square.fill", SortOrder: 71},
	{ID: "cat-eggs", DisplayName: "Eggs", Icon: "oval.fill", SortOrder: 72},
	{ID: "cat-deli-meats", DisplayName: "Deli Meats & Charcuterie", Icon: "fork.knife", SortOrder: 73},
	{ID: "cat-hummus-dips", DisplayName: "Hummus, Dips & Spreads", Icon: "circle.fill", SortOrder: 74},
	{ID: "cat-tofu-tempeh", DisplayName: "Tofu, Tempeh & Meat Alternatives", Icon: "leaf.fill", SortOrder: 75},
	{ID: "cat-fresh-pasta-sauces", DisplayName: "Fresh Pasta & Sauces", Icon: "takeoutbag.and.cup.and.straw", SortOrder: 76},
	// ── Bakery ──
	{ID: "cat-bread", DisplayName: "Bread & Rolls", Icon: "birthday.cake.fill", SortOrder: 80},
	{ID: "cat-tortillas-wraps", DisplayName: "Tortillas & Wraps", Icon: "circle.grid.2x2.fill", SortOrder: 81},
	{ID: "cat-bagels-english-muffins", DisplayName: "Bagels & English Muffins", Icon: "circle.fill", SortOrder: 82},
	{ID: "cat-cakes-pastries", DisplayName: "Cakes & Pastries", Icon: "birthday.cake.fill", SortOrder: 83},
	{ID: "cat-cookies-brownies", DisplayName: "Cookies & Brownies", Icon: "circle.grid.2x2.fill", SortOrder: 84},
	{ID: "cat-pita-naan", DisplayName: "Pita, Naan & Flatbread", Icon: "circle.fill", SortOrder: 85},
	{ID: "cat-baking-mixes", DisplayName: "Baking Mixes", Icon: "takeoutbag.and.cup.and.straw.fill", SortOrder: 86},
	// ── Breakfast ──
	{ID: "cat-cereal", DisplayName: "Cereal", Icon: "sun.max.fill", SortOrder: 90},
	{ID: "cat-oatmeal-porridge", DisplayName: "Oatmeal & Porridge", Icon: "cup.and.saucer.fill", SortOrder: 91},
	{ID: "cat-granola-bars", DisplayName: "Granola & Cereal Bars", Icon: "rectangle.fill", SortOrder: 92},
	{ID: "cat-pancake-waffle", DisplayName: "Pancake & Waffle Mix", Icon: "circle.grid.2x2.fill", SortOrder: 93},
	{ID: "cat-syrup-honey", DisplayName: "Syrup, Honey & Jam", Icon: "drop.fill", SortOrder: 94},
	// ── Pantry Staples ──
	{ID: "cat-rice", DisplayName: "Rice", Icon: "takeoutbag.and.cup.and.straw.fill", SortOrder: 100},
	{ID: "cat-pasta-noodles", DisplayName: "Pasta & Noodles", Icon: "takeoutbag.and.cup.and.straw.fill", SortOrder: 101},
	{ID: "cat-flour-baking", DisplayName: "Flour & Baking Ingredients", Icon: "shippingbox.fill", SortOrder: 102},
	{ID: "cat-sugar-sweeteners", DisplayName: "Sugar & Sweeteners", Icon: "cube.fill", SortOrder: 103},
	{ID: "cat-cooking-oils", DisplayName: "Cooking Oils", Icon: "drop.fill", SortOrder: 104},
	{ID: "cat-vinegar", DisplayName: "Vinegar", Icon: "drop.fill", SortOrder: 105},
	{ID: "cat-canned-vegetables", DisplayName: "Canned Vegetables", Icon: "cylinder.fill", SortOrder: 110},
	{ID: "cat-canned-fruit", DisplayName: "Canned Fruit", Icon: "cylinder.fill", SortOrder: 111},
	{ID: "cat-canned-beans-legumes", DisplayName: "Canned Beans & Legumes", Icon: "cylinder.fill", SortOrder: 112},
	{ID: "cat-canned-meat-fish", DisplayName: "Canned Meat & Fish", Icon: "cylinder.fill", SortOrder: 113},
	{ID: "cat-canned-soup", DisplayName: "Canned Soup & Broth", Icon: "cylinder.fill", SortOrder: 114},
	{ID: "cat-tomato-products", DisplayName: "Tomato Sauce & Paste", Icon: "cylinder.fill", SortOrder: 115},
	{ID: "cat-condiments", DisplayName: "Condiments & Ketchup", Icon: "takeoutbag.and.cup.and.straw", SortOrder: 120},
	{ID: "cat-mayonnaise-dressings", DisplayName: "Mayonnaise & Dressings", Icon: "drop.fill", SortOrder: 121},
	{ID: "cat-mustard-hot-sauce", DisplayName: "Mustard & Hot Sauce", Icon: "flame.fill", SortOrder: 122},
	{ID: "cat-soy-asian-sauces", DisplayName: "Soy & Asian Sauces", Icon: "drop.fill", SortOrder: 123},
	{ID: "cat-bbq-marinades", DisplayName: "BBQ Sauce & Marinades", Icon: "flame.fill", SortOrder: 124},
	{ID: "cat-spices-herbs", DisplayName: "Spices & Dried Herbs", Icon: "flame.fill", SortOrder: 130},
	{ID: "cat-salt-pepper", DisplayName: "Salt & Pepper", Icon: "circle.fill", SortOrder: 131},
	{ID: "cat-dried-beans-lentils", DisplayName: "Dried Beans & Lentils", Icon: "leaf.fill", SortOrder: 132},
	{ID: "cat-grains-couscous", DisplayName: "Grains, Quinoa & Couscous", Icon: "leaf.fill", SortOrder: 133},
	// ── Snacks ──
	{ID: "cat-chips-crisps", DisplayName: "Chips & Crisps", Icon: "shippingbox.fill", SortOrder: 140},
	{ID: "cat-popcorn", DisplayName: "Popcorn", Icon: "popcorn.fill", SortOrder: 141},
	{ID: "cat-pretzels-crackers", DisplayName: "Pretzels & Crackers", Icon: "circle.grid.2x2.fill", SortOrder: 142},
	{ID: "cat-nuts-seeds", DisplayName: "Nuts & Seeds", Icon: "leaf.fill", SortOrder: 150},
	{ID: "cat-dried-fruit", DisplayName: "Dried Fruit & Trail Mix", Icon: "leaf.fill", SortOrder: 151},
	{ID: "cat-jerky-meat-snacks", DisplayName: "Jerky & Meat Snacks", Icon: "fork.knife", SortOrder: 152},
	{ID: "cat-protein-bars", DisplayName: "Protein & Nutrition Bars", Icon: "rectangle.fill", SortOrder: 153},
	{ID: "cat-rice-cakes", DisplayName: "Rice Cakes & Puffed Snacks", Icon: "circle.fill", SortOrder: 154},
	// ── Candy & Sweets ──
	{ID: "cat-chocolate", DisplayName: "Chocolate", Icon: "heart.fill", SortOrder: 160},
	{ID: "cat-gummy-candy", DisplayName: "Gummy & Chewy Candy", Icon: "heart.fill", SortOrder: 161},
	{ID: "cat-hard-candy-mints", DisplayName: "Hard Candy & Mints", Icon: "circle.fill", SortOrder: 162},
	{ID: "cat-chewing-gum", DisplayName: "Chewing Gum", Icon: "circle.fill", SortOrder: 163},
	{ID: "cat-biscuits-cookies", DisplayName: "Biscuits & Cookies", Icon: "circle.grid.2x2.fill", SortOrder: 170},
	{ID: "cat-ice-cream-toppings", DisplayName: "Ice Cream Toppings & Cones", Icon: "snowflake", SortOrder: 171},
	// ── Fresh Produce ──
	{ID: "cat-fresh-fruit", DisplayName: "Fresh Fruit", Icon: "apple.logo", SortOrder: 180},
	{ID: "cat-fresh-vegetables", DisplayName: "Fresh Vegetables", Icon: "carrot.fill", SortOrder: 190},
	{ID: "cat-herbs-salads", DisplayName: "Fresh Herbs & Salads", Icon: "leaf.circle.fill", SortOrder: 200},
	{ID: "cat-organic-produce", DisplayName: "Organic Produce", Icon: "leaf.fill", SortOrder: 201},
	{ID: "cat-mushrooms", DisplayName: "Mushrooms", Icon: "leaf.fill", SortOrder: 202},
	{ID: "cat-potatoes-onions", DisplayName: "Potatoes, Onions & Root Veg", Icon: "circle.fill", SortOrder: 203},
	// ── Meat & Seafood ──
	{ID: "cat-beef", DisplayName: "Beef", Icon: "hare.fill", SortOrder: 210},
	{ID: "cat-chicken-turkey", DisplayName: "Chicken & Turkey", Icon: "hare.fill", SortOrder: 211},
	{ID: "cat-pork", DisplayName: "Pork", Icon: "hare.fill", SortOrder: 212},
	{ID: "cat-lamb", DisplayName: "Lamb & Mutton", Icon: "hare.fill", SortOrder: 213},
	{ID: "cat-ground-meat", DisplayName: "Ground Meat & Mince", Icon: "hare.fill", SortOrder: 214},
	{ID: "cat-sausages-hotdogs", DisplayName: "Sausages & Hot Dogs", Icon: "fork.knife", SortOrder: 215},
	{ID: "cat-fresh-fish", DisplayName: "Fresh Fish", Icon: "fish.fill", SortOrder: 220},
	{ID: "cat-shrimp-shellfish", DisplayName: "Shrimp & Shellfish", Icon: "fish.fill", SortOrder: 221},
	{ID: "cat-smoked-fish", DisplayName: "Smoked & Cured Fish", Icon: "fish.fill", SortOrder: 222},
	// ── Frozen ──
	{ID: "cat-frozen-meals", DisplayName: "Frozen Meals & Entrées", Icon: "snowflake", SortOrder: 230},
	{ID: "cat-frozen-pizza", DisplayName: "Frozen Pizza", Icon: "snowflake", SortOrder: 231},
	{ID: "cat-frozen-vegetables", DisplayName: "Frozen Vegetables", Icon: "snowflake", SortOrder: 232},
	{ID: "cat-frozen-fruit", DisplayName: "Frozen Fruit", Icon: "snowflake", SortOrder: 233},
	{ID: "cat-frozen-meat-seafood", DisplayName: "Frozen Meat & Seafood", Icon: "snowflake", SortOrder: 234},
	{ID: "cat-frozen-snacks", DisplayName: "Frozen Snacks & Appetizers", Icon: "snowflake", SortOrder: 235},
	{ID: "cat-frozen-breakfast", DisplayName: "Frozen Breakfast", Icon: "snowflake", SortOrder: 236},
	{ID: "cat-ice-cream", DisplayName: "Ice Cream & Frozen Desserts", Icon: "snowflake", SortOrder: 237},
	{ID: "cat-frozen-fries-potatoes", DisplayName: "Frozen Fries & Potatoes", Icon: "snowflake", SortOrder: 238},
	// ── International & Specialty ──
	{ID: "cat-mexican-food", DisplayName: "Mexican & Latin Food", Icon: "globe.americas.fill", SortOrder: 240},
	{ID: "cat-asian-food", DisplayName: "Asian Food", Icon: "globe.asia.australia.fill", SortOrder: 241},
	{ID: "cat-indian-food", DisplayName: "Indian Food", Icon: "globe.asia.australia.fill", SortOrder: 242},
	{ID: "cat-middle-eastern-food", DisplayName: "Middle Eastern Food", Icon: "globe.europe.africa.fill", SortOrder: 243},
	{ID: "cat-italian-food", DisplayName: "Italian Specialty", Icon: "globe.europe.africa.fill", SortOrder: 244},
	{ID: "cat-kosher", DisplayName: "Kosher", Icon: "star.fill", SortOrder: 245},
	{ID: "cat-halal", DisplayName: "Halal", Icon: "star.fill", SortOrder: 246},
	{ID: "cat-gluten-free", DisplayName: "Gluten-Free", Icon: "leaf.fill", SortOrder: 247},
	{ID: "cat-vegan-plant-based", DisplayName: "Vegan & Plant-Based", Icon: "leaf.fill", SortOrder: 248},
	{ID: "cat-organic-natural", DisplayName: "Organic & Natural", Icon: "leaf.fill", SortOrder: 249},
	// ── Baby & Kids ──
	{ID: "cat-baby-formula", DisplayName: "Baby Formula", Icon: "figure.and.child.holdinghands", SortOrder: 250},
	{ID: "cat-baby-food", DisplayName: "Baby Food & Snacks", Icon: "figure.and.child.holdinghands", SortOrder: 251},
	{ID: "cat-diapers", DisplayName: "Diapers & Wipes", Icon: "stroller.fill", SortOrder: 260},
	{ID: "cat-baby-bath-skin", DisplayName: "Baby Bath & Skin Care", Icon: "stroller.fill", SortOrder: 261},
	{ID: "cat-baby-feeding", DisplayName: "Baby Bottles & Feeding", Icon: "stroller.fill", SortOrder: 262},
	// ── Health & Wellness ──
	{ID: "cat-medicine-otc", DisplayName: "OTC Medicine & Pain Relief", Icon: "cross.fill", SortOrder: 340},
	{ID: "cat-cold-flu", DisplayName: "Cold, Flu & Allergy", Icon: "cross.fill", SortOrder: 341},
	{ID: "cat-digestive-health", DisplayName: "Digestive Health", Icon: "cross.fill", SortOrder: 342},
	{ID: "cat-first-aid", DisplayName: "First Aid & Bandages", Icon: "cross.case.fill", SortOrder: 343},
	{ID: "cat-vitamins", DisplayName: "Vitamins & Supplements", Icon: "pills.fill", SortOrder: 350},
	{ID: "cat-protein-powder", DisplayName: "Protein Powder & Shakes", Icon: "bolt.fill", SortOrder: 351},
	{ID: "cat-eye-ear-care", DisplayName: "Eye & Ear Care", Icon: "eye.fill", SortOrder: 352},
	{ID: "cat-diabetes-care", DisplayName: "Diabetes Care", Icon: "cross.fill", SortOrder: 353},
	{ID: "cat-mobility-aids", DisplayName: "Mobility & Daily Living Aids", Icon: "figure.walk", SortOrder: 354},
	// ── Personal Care & Beauty ──
	{ID: "cat-shampoo-conditioner", DisplayName: "Shampoo & Conditioner", Icon: "person.crop.circle.badge.checkmark", SortOrder: 290},
	{ID: "cat-hair-styling", DisplayName: "Hair Styling & Treatment", Icon: "person.crop.circle.badge.checkmark", SortOrder: 291},
	{ID: "cat-hair-color", DisplayName: "Hair Color & Dye", Icon: "paintbrush.fill", SortOrder: 292},
	{ID: "cat-body-wash-soap", DisplayName: "Body Wash & Bar Soap", Icon: "drop.fill", SortOrder: 270},
	{ID: "cat-deodorant", DisplayName: "Deodorant & Antiperspirant", Icon: "person.crop.circle.fill", SortOrder: 271},
	{ID: "cat-lotion-moisturizer", DisplayName: "Lotion & Moisturizer", Icon: "drop.fill", SortOrder: 300},
	{ID: "cat-sunscreen", DisplayName: "Sunscreen & Sun Care", Icon: "sun.max.fill", SortOrder: 301},
	{ID: "cat-face-care", DisplayName: "Face Care & Cleansers", Icon: "sparkles", SortOrder: 302},
	{ID: "cat-lip-care", DisplayName: "Lip Care & Balm", Icon: "mouth.fill", SortOrder: 303},
	{ID: "cat-oral-care", DisplayName: "Toothpaste & Oral Care", Icon: "mouth.fill", SortOrder: 280},
	{ID: "cat-mouthwash", DisplayName: "Mouthwash & Floss", Icon: "mouth.fill", SortOrder: 281},
	{ID: "cat-shaving", DisplayName: "Shaving & Razors", Icon: "mustache.fill", SortOrder: 330},
	{ID: "cat-mens-grooming", DisplayName: "Men's Grooming", Icon: "mustache.fill", SortOrder: 331},
	{ID: "cat-feminine-care", DisplayName: "Feminine Care", Icon: "cross.case.fill", SortOrder: 320},
	{ID: "cat-cotton-pads", DisplayName: "Cotton, Swabs & Pads", Icon: "circle.fill", SortOrder: 321},
	{ID: "cat-cosmetics-face", DisplayName: "Face Makeup & Foundation", Icon: "paintbrush.fill", SortOrder: 310},
	{ID: "cat-cosmetics-eyes", DisplayName: "Eye Makeup & Mascara", Icon: "eye.fill", SortOrder: 311},
	{ID: "cat-cosmetics-lips", DisplayName: "Lipstick & Lip Gloss", Icon: "mouth.fill", SortOrder: 312},
	{ID: "cat-nail-care", DisplayName: "Nail Polish & Nail Care", Icon: "paintbrush.fill", SortOrder: 313},
	{ID: "cat-fragrance", DisplayName: "Perfume & Fragrance", Icon: "sparkles", SortOrder: 314},
	{ID: "cat-hair-tools", DisplayName: "Hair Dryers, Irons & Tools", Icon: "bolt.fill", SortOrder: 315},
	// ── Household Cleaning ──
	{ID: "cat-all-purpose-cleaners", DisplayName: "All-Purpose Cleaners", Icon: "sparkles", SortOrder: 360},
	{ID: "cat-bathroom-cleaners", DisplayName: "Bathroom Cleaners", Icon: "sparkles", SortOrder: 361},
	{ID: "cat-kitchen-cleaners", DisplayName: "Kitchen & Oven Cleaners", Icon: "sparkles", SortOrder: 362},
	{ID: "cat-glass-cleaners", DisplayName: "Glass & Window Cleaners", Icon: "sparkles", SortOrder: 363},
	{ID: "cat-floor-care", DisplayName: "Floor Care & Mopping", Icon: "sparkles", SortOrder: 364},
	{ID: "cat-disinfectants", DisplayName: "Disinfectants & Sanitizers", Icon: "cross.fill", SortOrder: 365},
	{ID: "cat-laundry-detergent", DisplayName: "Laundry Detergent", Icon: "washer.fill", SortOrder: 370},
	{ID: "cat-fabric-softener", DisplayName: "Fabric Softener & Dryer Sheets", Icon: "washer.fill", SortOrder: 371},
	{ID: "cat-stain-removers", DisplayName: "Stain Removers", Icon: "drop.triangle.fill", SortOrder: 372},
	{ID: "cat-dishwashing", DisplayName: "Dish Soap & Dishwasher Pods", Icon: "drop.triangle.fill", SortOrder: 390},
	{ID: "cat-trash-bags", DisplayName: "Trash Bags", Icon: "trash.fill", SortOrder: 391},
	{ID: "cat-sponges-cloths", DisplayName: "Sponges, Cloths & Brushes", Icon: "paintbrush.fill", SortOrder: 392},
	{ID: "cat-air-fresheners", DisplayName: "Air Fresheners & Candles", Icon: "flame.fill", SortOrder: 393},
	{ID: "cat-pest-control", DisplayName: "Pest Control & Insect Repellent", Icon: "ant.fill", SortOrder: 394},
	// ── Paper & Disposable ──
	{ID: "cat-toilet-paper", DisplayName: "Toilet Paper", Icon: "newspaper.fill", SortOrder: 380},
	{ID: "cat-paper-towels", DisplayName: "Paper Towels", Icon: "newspaper.fill", SortOrder: 381},
	{ID: "cat-facial-tissue", DisplayName: "Facial Tissue", Icon: "newspaper.fill", SortOrder: 382},
	{ID: "cat-napkins", DisplayName: "Napkins", Icon: "newspaper.fill", SortOrder: 383},
	{ID: "cat-plates-cups-disposable", DisplayName: "Disposable Plates, Cups & Cutlery", Icon: "cup.and.saucer.fill", SortOrder: 384},
	{ID: "cat-food-wrap-bags", DisplayName: "Food Wrap, Foil & Zip Bags", Icon: "archivebox.fill", SortOrder: 385},
	// ── Kitchen & Dining ──
	{ID: "cat-cookware", DisplayName: "Cookware & Pots", Icon: "frying.pan.fill", SortOrder: 410},
	{ID: "cat-bakeware", DisplayName: "Bakeware", Icon: "frying.pan.fill", SortOrder: 411},
	{ID: "cat-kitchen-utensils", DisplayName: "Kitchen Utensils & Gadgets", Icon: "fork.knife", SortOrder: 412},
	{ID: "cat-knives-cutting", DisplayName: "Knives & Cutting Boards", Icon: "fork.knife", SortOrder: 413},
	{ID: "cat-food-storage", DisplayName: "Food Storage & Containers", Icon: "archivebox.fill", SortOrder: 414},
	{ID: "cat-water-bottles", DisplayName: "Water Bottles & Tumblers", Icon: "drop.fill", SortOrder: 415},
	{ID: "cat-dinnerware", DisplayName: "Dinnerware & Glassware", Icon: "cup.and.saucer.fill", SortOrder: 416},
	{ID: "cat-small-appliances", DisplayName: "Small Kitchen Appliances", Icon: "bolt.fill", SortOrder: 417},
	// ── Home & Living ──
	{ID: "cat-bedding", DisplayName: "Bedding & Sheets", Icon: "bed.double.fill", SortOrder: 400},
	{ID: "cat-pillows-blankets", DisplayName: "Pillows & Blankets", Icon: "bed.double.fill", SortOrder: 401},
	{ID: "cat-towels", DisplayName: "Bath Towels & Mats", Icon: "square.fill", SortOrder: 402},
	{ID: "cat-curtains-blinds", DisplayName: "Curtains & Blinds", Icon: "square.fill", SortOrder: 403},
	{ID: "cat-home-decor", DisplayName: "Home Décor & Frames", Icon: "photo.fill", SortOrder: 404},
	{ID: "cat-candles-home-fragrance", DisplayName: "Candles & Home Fragrance", Icon: "flame.fill", SortOrder: 405},
	{ID: "cat-storage-organization", DisplayName: "Storage & Organization", Icon: "archivebox.fill", SortOrder: 406},
	{ID: "cat-hangers-laundry-supplies", DisplayName: "Hangers & Laundry Supplies", Icon: "archivebox.fill", SortOrder: 407},
	{ID: "cat-light-bulbs", DisplayName: "Light Bulbs", Icon: "lightbulb.fill", SortOrder: 470},
	{ID: "cat-batteries", DisplayName: "Batteries", Icon: "battery.100", SortOrder: 471},
	{ID: "cat-extension-cords", DisplayName: "Extension Cords & Power Strips", Icon: "cable.connector", SortOrder: 472},
	// ── Garden & Outdoor ──
	{ID: "cat-plants-seeds", DisplayName: "Plants & Seeds", Icon: "leaf.fill", SortOrder: 500},
	{ID: "cat-soil-fertilizer", DisplayName: "Soil & Fertilizer", Icon: "leaf.fill", SortOrder: 501},
	{ID: "cat-garden-tools", DisplayName: "Garden Tools", Icon: "wrench.fill", SortOrder: 502},
	{ID: "cat-outdoor-furniture", DisplayName: "Outdoor Furniture", Icon: "chair.fill", SortOrder: 503},
	{ID: "cat-grills-charcoal", DisplayName: "Grills & Charcoal", Icon: "flame.fill", SortOrder: 504},
	// ── Pets ──
	{ID: "cat-dog-food", DisplayName: "Dog Food", Icon: "pawprint.fill", SortOrder: 420},
	{ID: "cat-cat-food", DisplayName: "Cat Food", Icon: "pawprint.fill", SortOrder: 421},
	{ID: "cat-pet-treats", DisplayName: "Pet Treats", Icon: "pawprint.fill", SortOrder: 422},
	{ID: "cat-pet-toys", DisplayName: "Pet Toys", Icon: "pawprint.circle.fill", SortOrder: 423},
	{ID: "cat-pet-grooming", DisplayName: "Pet Grooming & Health", Icon: "pawprint.circle.fill", SortOrder: 430},
	{ID: "cat-litter-waste", DisplayName: "Cat Litter & Waste Bags", Icon: "pawprint.fill", SortOrder: 431},
	{ID: "cat-pet-beds-carriers", DisplayName: "Pet Beds & Carriers", Icon: "pawprint.circle.fill", SortOrder: 432},
	// ── Office & School ──
	{ID: "cat-pens-pencils", DisplayName: "Pens, Pencils & Markers", Icon: "pencil.and.outline", SortOrder: 440},
	{ID: "cat-notebooks-paper", DisplayName: "Notebooks & Paper", Icon: "newspaper.fill", SortOrder: 441},
	{ID: "cat-binders-folders", DisplayName: "Binders, Folders & Filing", Icon: "archivebox.fill", SortOrder: 442},
	{ID: "cat-tape-glue-scissors", DisplayName: "Tape, Glue & Scissors", Icon: "scissors", SortOrder: 443},
	{ID: "cat-printer-ink", DisplayName: "Printer Ink & Toner", Icon: "printer.fill", SortOrder: 444},
	{ID: "cat-backpacks-bags", DisplayName: "Backpacks & School Bags", Icon: "bag.fill", SortOrder: 445},
	// ── Toys & Games ──
	{ID: "cat-action-figures", DisplayName: "Action Figures & Dolls", Icon: "gamecontroller.fill", SortOrder: 450},
	{ID: "cat-building-sets", DisplayName: "Building Sets & Blocks", Icon: "square.stack.3d.up.fill", SortOrder: 451},
	{ID: "cat-board-games-puzzles", DisplayName: "Board Games & Puzzles", Icon: "gamecontroller.fill", SortOrder: 452},
	{ID: "cat-outdoor-play", DisplayName: "Outdoor Play & Sports Toys", Icon: "figure.run", SortOrder: 453},
	{ID: "cat-arts-crafts", DisplayName: "Arts & Crafts", Icon: "paintbrush.fill", SortOrder: 454},
	{ID: "cat-stuffed-animals", DisplayName: "Stuffed Animals & Plush", Icon: "heart.fill", SortOrder: 455},
	// ── Electronics & Accessories ──
	{ID: "cat-phone-accessories", DisplayName: "Phone Cases & Accessories", Icon: "iphone", SortOrder: 480},
	{ID: "cat-chargers-cables", DisplayName: "Chargers & Cables", Icon: "cable.connector", SortOrder: 481},
	{ID: "cat-headphones-earbuds", DisplayName: "Headphones & Earbuds", Icon: "headphones", SortOrder: 482},
	{ID: "cat-memory-cards-usb", DisplayName: "Memory Cards & USB Drives", Icon: "externaldrive.fill", SortOrder: 483},
	{ID: "cat-smart-home", DisplayName: "Smart Home Devices", Icon: "homekit", SortOrder: 484},
	// ── Automotive ──
	{ID: "cat-motor-oil", DisplayName: "Motor Oil & Fluids", Icon: "car.fill", SortOrder: 510},
	{ID: "cat-car-cleaning", DisplayName: "Car Cleaning & Detailing", Icon: "car.fill", SortOrder: 511},
	{ID: "cat-car-fresheners", DisplayName: "Car Air Fresheners", Icon: "car.fill", SortOrder: 512},
	{ID: "cat-car-accessories", DisplayName: "Car Accessories", Icon: "car.fill", SortOrder: 513},
	// ── Party & Seasonal ──
	{ID: "cat-party-supplies", DisplayName: "Party Supplies & Balloons", Icon: "balloon.2.fill", SortOrder: 460},
	{ID: "cat-gift-wrap", DisplayName: "Gift Wrap & Bags", Icon: "gift.fill", SortOrder: 461},
	{ID: "cat-greeting-cards", DisplayName: "Greeting Cards", Icon: "envelope.fill", SortOrder: 462},
	{ID: "cat-seasonal", DisplayName: "Seasonal & Holiday Items", Icon: "calendar", SortOrder: 490},
	// ── Tobacco & Adjacent ──
	{ID: "cat-tobacco", DisplayName: "Tobacco Products", Icon: "smoke.fill", SortOrder: 520},
	{ID: "cat-lighters-matches", DisplayName: "Lighters & Matches", Icon: "flame.fill", SortOrder: 521},
	// ── Other ──
	{ID: "cat-other", DisplayName: "Other", Icon: "square.grid.2x2.fill", SortOrder: 999},
}

var canonicalCategoryIndex = func() map[string]canonicalCategory {
	index := make(map[string]canonicalCategory, len(canonicalCategories))
	for _, category := range canonicalCategories {
		index[category.ID] = category
	}
	return index
}()

var categorySeedState struct {
	mu   sync.Mutex
	done bool
}

func ensureCanonicalCategoriesSeeded(ctx context.Context, client *spanner.Client) error {
	categorySeedState.mu.Lock()
	defer categorySeedState.mu.Unlock()

	if categorySeedState.done {
		return nil
	}

	mutations := make([]*spanner.Mutation, 0, len(canonicalCategories)*2)
	for _, category := range canonicalCategories {
		mutations = append(mutations,
			spanner.InsertOrUpdate(
				"PlatformCategories",
				[]string{"CategoryId", "DisplayName", "IconUrl", "DisplayOrder"},
				[]interface{}{category.ID, category.DisplayName, category.Icon, category.SortOrder},
			),
			spanner.InsertOrUpdate(
				"Categories",
				[]string{"CategoryId", "Name", "Icon", "SortOrder", "CreatedAt"},
				[]interface{}{category.ID, category.DisplayName, category.Icon, category.SortOrder, spanner.CommitTimestamp},
			),
		)
	}

	if _, err := client.Apply(ctx, mutations); err != nil {
		return fmt.Errorf("seed canonical categories: %w", err)
	}

	categorySeedState.done = true
	log.Printf("[catalog] canonical categories seeded: %d", len(canonicalCategories))
	return nil
}

func normalizeValidCategoryIDs(raw []string) ([]string, []string) {
	seen := map[string]struct{}{}
	valid := make([]string, 0, len(raw))
	invalid := make([]string, 0)
	for _, item := range raw {
		id := strings.TrimSpace(item)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := canonicalCategoryIndex[id]; ok {
			valid = append(valid, id)
		} else {
			invalid = append(invalid, id)
		}
	}
	return valid, invalid
}

func categoryDisplayNameByID(id string) string {
	if category, ok := canonicalCategoryIndex[id]; ok {
		return category.DisplayName
	}
	return ""
}

func categoryDisplayNames(ids []string) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name := categoryDisplayNameByID(id); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func primaryCategoryName(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return categoryDisplayNameByID(ids[0])
}

func loadSupplierCategoryAccess(ctx context.Context, client *spanner.Client, supplierID string) (bool, []string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT IFNULL(IsConfigured, false), COALESCE(OperatingCategories, [])
		      FROM Suppliers
		      WHERE SupplierId = @supplierId`,
		Params: map[string]interface{}{"supplierId": supplierID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return false, nil, fmt.Errorf("load supplier category access: %w", err)
	}

	var isConfigured bool
	var operatingCategories []string
	if err := row.Columns(&isConfigured, &operatingCategories); err != nil {
		return false, nil, fmt.Errorf("parse supplier category access: %w", err)
	}

	return isConfigured, operatingCategories, nil
}

func containsCategoryID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
