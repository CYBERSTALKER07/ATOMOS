package migration

import (
	"strings"
	"unicode"
)

// CanonicalField describes a target PegasusX field inside an ingestion pack,
// along with multilingual header aliases across 1C, Billz, MoySklad, Smartup, Excel, and CommerceML.
type CanonicalField struct {
	Key          string   `json:"key"`
	LabelEN      string   `json:"label_en"`
	LabelRU      string   `json:"label_ru"`
	LabelUZ      string   `json:"label_uz"`
	Required     bool     `json:"required"`
	Type         string   `json:"type"` // string, money, number, date, phone, inn, bool
	StrongAnchor bool     `json:"strong_anchor"`
	Aliases      []string `json:"aliases"`
}

// RequiredGroup expresses "at least one field in this group must be mapped"
// (for example, SKU or Barcode for products, INN or Phone for customers).
type RequiredGroup struct {
	Name   string
	Fields []string
}

var packTaxonomies = map[Pack][]CanonicalField{
	PackWarehouses: {
		{
			Key: "external_warehouse_id", LabelEN: "Warehouse Code", LabelRU: "Код склада", LabelUZ: "Ombor kodi",
			Type: "string", StrongAnchor: true,
			Aliases: []string{
				"external_warehouse_id", "warehouse_id", "warehouse_code", "wh_id", "wh_code",
				"код склада", "ид склада", "идсклада", "код_склада", "id склада",
				"ombor kodi", "ombor_kodi", "ombor id", "омбор коди",
			},
		},
		{
			Key: "name", LabelEN: "Warehouse Name", LabelRU: "Название склада", LabelUZ: "Ombor nomi",
			Required: true, Type: "string", StrongAnchor: true,
			Aliases: []string{
				"warehouse_name", "warehouse", "wh_name", "name", "ombor", "ombor nomi",
				"ombor_nomi", "название склада", "склад", "наименование склада", "омбор", "омбор номи",
			},
		},
		{
			Key: "address", LabelEN: "Address", LabelRU: "Адрес", LabelUZ: "Manzil",
			Type: "string",
			Aliases: []string{"address", "wh_address", "адрес", "адрес склада", "manzil", "ombor manzili", "манзил"},
		},
		{
			Key: "city", LabelEN: "City / Region", LabelRU: "Город", LabelUZ: "Shahar",
			Type: "string",
			Aliases: []string{"city", "region", "город", "регион", "shahar", "viloyat", "шаҳар"},
		},
		{
			Key: "zone", LabelEN: "Zone", LabelRU: "Зона", LabelUZ: "Zona",
			Type: "string",
			Aliases: []string{"zone", "wh_zone", "зона", "зона хранения", "hudud", "zona"},
		},
		{
			Key: "temperature_mode", LabelEN: "Temperature Mode", LabelRU: "Температурный режим", LabelUZ: "Harorat rejimi",
			Type: "string", StrongAnchor: true,
			Aliases: []string{"temperature_mode", "temp_mode", "температура", "температурный режим", "harorat", "harorat rejimi"},
		},
		{
			Key: "is_default", LabelEN: "Default Warehouse", LabelRU: "Основной склад", LabelUZ: "Asosiy ombor",
			Type: "bool",
			Aliases: []string{"is_default", "default_warehouse", "основной", "основной склад", "asosiy ombor", "асосий омбор"},
		},
	},
	PackProductsPrices: {
		{
			Key: "sku", LabelEN: "SKU / Article", LabelRU: "Артикул / SKU", LabelUZ: "Artikul / SKU",
			Type: "string", StrongAnchor: true,
			Aliases: []string{
				"sku", "product_sku", "item_code", "article", "vendor_code", "external_sku_id", "code", "kod",
				"артикул", "код товара", "код_товара", "код номенклатуры", "sku товара",
				"artikul", "tovar kodi", "mahsulot kodi", "kod tovar", "артикул товару", "товар коди",
			},
		},
		{
			Key: "barcode", LabelEN: "Barcode / GTIN", LabelRU: "Штрихкод", LabelUZ: "Shtrix kod",
			Type: "string", StrongAnchor: true,
			Aliases: []string{
				"barcode", "gtin", "ean", "ean13", "upc", "штрихкод", "штрих-код", "штрих код",
				"shtrix", "shtrix kod", "shtrix-kod", "bar code", "штрихкоди",
			},
		},
		{
			Key: "name", LabelEN: "Product Name", LabelRU: "Наименование товара", LabelUZ: "Mahsulot nomi",
			Required: true, Type: "string", StrongAnchor: true,
			Aliases: []string{
				"name", "product_name", "item_name", "title", "goods_name",
				"наименование", "название", "товар", "номенклатура", "наименование товара", "название товара",
				"nomi", "tovar nomi", "mahsulot nomi", "tovar", "номи", "маҳсулот номи", "товар номи",
			},
		},
		{
			Key: "unit", LabelEN: "Unit (UoM)", LabelRU: "Единица измерения", LabelUZ: "O'lchov birligi",
			Type: "string",
			Aliases: []string{
				"unit", "uom", "base_uom", "measure", "ед", "ед.изм", "ед изм", "единица", "единица измерения", "базоваяединица",
				"birlik", "olchov birligi", "o'lchov birligi", "бирлик", "ўлчов бирлиги",
			},
		},
		{
			Key: "price", LabelEN: "Wholesale Price", LabelRU: "Оптовая цена", LabelUZ: "Ulgurji narx",
			Type: "money", StrongAnchor: true,
			Aliases: []string{
				"price", "wholesale_price", "sale_price", "unit_price", "sell_price", "opt_price",
				"цена", "оптовая цена", "цена продажи", "прайс", "стоимость", "цена за единицу",
				"narx", "narxi", "ulgurji narx", "ulgurji", "sotuv narxi", "нарх", "нархи", "улгуржи нарх",
			},
		},
		{
			Key: "retail_price", LabelEN: "Recommended Retail Price", LabelRU: "Розничная цена", LabelUZ: "Chakana narx",
			Type: "money",
			Aliases: []string{"retail_price", "rrp", "розничная цена", "розница", "chakana narx", "чакана нарх"},
		},
		{
			Key: "cost_price", LabelEN: "Cost Price", LabelRU: "Себестоимость", LabelUZ: "Tannarx",
			Type: "money",
			Aliases: []string{"cost", "cost_price", "purchase_price", "закупочная цена", "себестоимость", "приходная цена", "tannarx", "xarid narxi", "таннарх"},
		},
		{
			Key: "currency", LabelEN: "Currency", LabelRU: "Валюта", LabelUZ: "Valyuta",
			Type: "string",
			Aliases: []string{"currency", "curr", "валюта", "valyuta", "валютаси"},
		},
		{
			Key: "category", LabelEN: "Category", LabelRU: "Категория", LabelUZ: "Kategoriya",
			Type: "string",
			Aliases: []string{"category", "group", "категория", "группа", "группа товаров", "kategoriya", "turkum", "guruh", "категорияси"},
		},
		{
			Key: "brand", LabelEN: "Brand", LabelRU: "Бренд", LabelUZ: "Brend",
			Type: "string",
			Aliases: []string{"brand", "manufacturer", "бренд", "производитель", "торговая марка", "brend", "ishlab chiqaruvchi"},
		},
		{
			Key: "vat_rate", LabelEN: "VAT Rate (%)", LabelRU: "Ставка НДС", LabelUZ: "QQS stavkasi",
			Type: "number",
			Aliases: []string{"vat_rate", "vat", "nds", "ндс", "ставка ндс", "qqs", "ққс"},
		},
		{
			Key: "pack_size", LabelEN: "Pack Size", LabelRU: "Вложение / Упаковка", LabelUZ: "Qadoq soni",
			Type: "number",
			Aliases: []string{"pack_size", "box_qty", "кратность", "в упаковке", "qadoq", "korobka"},
		},
		{
			Key: "shelf_life_days", LabelEN: "Shelf Life (Days)", LabelRU: "Срок годности (дней)", LabelUZ: "Yaroqlilik muddati (kun)",
			Type: "number",
			Aliases: []string{"shelf_life_days", "срок годности дней", "yaroqlilik kun"},
		},
		{
			Key: "description", LabelEN: "Description", LabelRU: "Описание", LabelUZ: "Tavsif",
			Type: "string",
			Aliases: []string{"description", "desc", "описание", "комментарий", "tavsif", "izoh"},
		},
	},
	PackCustomers: {
		{
			Key: "external_customer_id", LabelEN: "Customer Code", LabelRU: "Код клиента", LabelUZ: "Mijoz kodi",
			Type: "string",
			Aliases: []string{"external_customer_id", "customer_id", "client_id", "код клиента", "код контрагента", "mijoz kodi"},
		},
		{
			Key: "customer_name", LabelEN: "Customer / Store Name", LabelRU: "Наименование контрагента", LabelUZ: "Mijoz / Do'kon nomi",
			Required: true, Type: "string", StrongAnchor: true,
			Aliases: []string{
				"customer_name", "customer", "client", "client_name", "company", "company_name", "store", "shop_name", "retailer_name", "buyer", "counterparty",
				"контрагент", "покупатель", "клиент", "наименование клиента", "магазин", "торговая точка", "организация",
				"mijoz", "mijoz nomi", "dokon", "do'kon nomi", "xaridor", "klient", "мижоз", "харидор", "дўкон", "клиент",
			},
		},
		{
			Key: "inn", LabelEN: "Tax ID (INN / STIR)", LabelRU: "ИНН", LabelUZ: "STIR (INN)",
			Type: "inn", StrongAnchor: true,
			Aliases: []string{"inn", "stir", "tax_id", "tin", "pinfl", "инн", "стир", "инн покупателя", "бин", "инн контрагента"},
		},
		{
			Key: "phone", LabelEN: "Phone", LabelRU: "Телефон", LabelUZ: "Telefon",
			Type: "phone", StrongAnchor: true,
			Aliases: []string{"phone", "customer_phone", "mobile", "tel", "телефон", "номер телефона", "тел", "контакты", "telefon", "tel raqam", "телефон рақами"},
		},
		{
			Key: "address", LabelEN: "Delivery Address", LabelRU: "Адрес", LabelUZ: "Manzil",
			Type: "string",
			Aliases: []string{"address", "delivery_address", "customer_address", "адрес", "адрес доставки", "юридический адрес", "manzil", "yetkazish manzili", "манзил"},
		},
		{
			Key: "district", LabelEN: "District / Territory", LabelRU: "Район", LabelUZ: "Tuman",
			Type: "string",
			Aliases: []string{"district", "area", "territory", "район", "округ", "tuman", "hudud", "туман"},
		},
		{
			Key: "credit_limit", LabelEN: "Credit Limit", LabelRU: "Кредитный лимит", LabelUZ: "Kredit limiti",
			Type: "money", StrongAnchor: true,
			Aliases: []string{"credit_limit", "лимит кредита", "кредитный лимит", "лимит", "kredit limiti", "qarz limiti", "кредит лимити"},
		},
		{
			Key: "payment_terms_days", LabelEN: "Payment Terms (Days)", LabelRU: "Отсрочка (дней)", LabelUZ: "To'lov muddati (kun)",
			Type: "number",
			Aliases: []string{"payment_terms_days", "payment_terms", "отсрочка", "дней отсрочки", "tolov muddati"},
		},
		{
			Key: "email", LabelEN: "Email", LabelRU: "Email", LabelUZ: "Elektron pochta",
			Type: "string",
			Aliases: []string{"email", "e_mail", "почта", "email контрагента"},
		},
	},
	PackStock: {
		{
			Key: "sku", LabelEN: "SKU / Barcode", LabelRU: "Артикул / Штрихкод", LabelUZ: "SKU / Shtrix",
			Required: true, Type: "string",
			Aliases: []string{
				"sku", "external_sku_id", "article", "barcode", "item_code", "product_name",
				"артикул", "код товара", "штрихкод", "tovar kodi", "artikul", "shtrix",
			},
		},
		{
			Key: "warehouse_id", LabelEN: "Warehouse", LabelRU: "Склад", LabelUZ: "Ombor",
			Type: "string", StrongAnchor: true,
			Aliases: []string{"warehouse_id", "warehouse_name", "warehouse", "wh", "склад", "код склада", "название склада", "ombor", "ombor nomi", "sklad", "омбор"},
		},
		{
			Key: "quantity_on_hand", LabelEN: "Quantity On Hand", LabelRU: "Остаток на складе", LabelUZ: "Ombordagi qoldiq",
			Required: true, Type: "number", StrongAnchor: true,
			Aliases: []string{
				"quantity_on_hand", "qty_on_hand", "on_hand", "stock", "stock_qty", "qty", "quantity", "available",
				"остаток", "остатки", "остаток на складе", "количество на складе", "в наличии", "доступно", "свободный остаток", "факт",
				"qoldiq", "ostatok", "soni", "miqdor", "miqdori", "mavjud", "қолдиқ", "остаток", "сони",
			},
		},
		{
			Key: "reserved_qty", LabelEN: "Reserved Qty", LabelRU: "Резерв", LabelUZ: "Band qilingan",
			Type: "number",
			Aliases: []string{"reserved_qty", "reserved", "резерв", "в резерве", "зарезервировано", "band", "rezerv"},
		},
		{
			Key: "cost_price", LabelEN: "Unit Cost", LabelRU: "Себестоимость", LabelUZ: "Tannarx",
			Type: "money",
			Aliases: []string{"cost_price", "unit_cost", "себестоимость", "учетная цена", "tannarx"},
		},
		{
			Key: "unit", LabelEN: "Unit", LabelRU: "Единица", LabelUZ: "Birlik",
			Type: "string",
			Aliases: []string{"unit", "uom", "ед", "единица", "birlik"},
		},
	},
	PackLots: {
		{
			Key: "sku", LabelEN: "SKU / Barcode", LabelRU: "Артикул / Штрихкод", LabelUZ: "SKU / Shtrix",
			Required: true, Type: "string",
			Aliases: []string{"sku", "external_sku_id", "article", "barcode", "артикул", "код товара", "штрихкод", "tovar kodi", "artikul"},
		},
		{
			Key: "warehouse_id", LabelEN: "Warehouse", LabelRU: "Склад", LabelUZ: "Ombor",
			Type: "string",
			Aliases: []string{"warehouse_id", "warehouse_name", "warehouse", "склад", "ombor", "sklad"},
		},
		{
			Key: "lot_number", LabelEN: "Lot / Batch Code", LabelRU: "Партия / Серия", LabelUZ: "Partiya / Seriya",
			Required: true, Type: "string", StrongAnchor: true,
			Aliases: []string{
				"lot_number", "lot", "lot_code", "batch", "batch_number", "batch_code", "series", "serial",
				"партия", "серия", "номер партии", "код партии",
				"partiya", "seriya", "partiya raqami", "партия", "серия",
			},
		},
		{
			Key: "expiry_date", LabelEN: "Expiry Date", LabelRU: "Срок годности", LabelUZ: "Yaroqlilik muddati",
			Required: true, Type: "date", StrongAnchor: true,
			Aliases: []string{
				"expiry_date", "expiration_date", "exp_date", "best_before", "use_by",
				"срок годности", "годен до", "дата окончания", "срок",
				"yaroqlilik muddati", "yaroqlilik", "srok", "muddat", "яроқлилик муддати", "годен до",
			},
		},
		{
			Key: "production_date", LabelEN: "Production Date", LabelRU: "Дата производства", LabelUZ: "Ishlab chiqarilgan sana",
			Type: "date", StrongAnchor: true,
			Aliases: []string{"production_date", "mfg_date", "manufactured_at", "дата производства", "дата изготовления", "ishlab chiqarilgan sana"},
		},
		{
			Key: "quantity_on_hand", LabelEN: "Lot Quantity", LabelRU: "Количество в партии", LabelUZ: "Partiya qoldig'i",
			Required: true, Type: "number",
			Aliases: []string{"quantity_on_hand", "lot_qty", "qty", "quantity", "stock", "остаток", "количество", "qoldiq", "miqdor", "soni"},
		},
		{
			Key: "cost_price", LabelEN: "Lot Unit Cost", LabelRU: "Себестоимость партии", LabelUZ: "Partiya tannarxi",
			Type: "money",
			Aliases: []string{"cost_price", "unit_cost", "себестоимость", "tannarx"},
		},
	},
	PackHistoricalOrders: {
		{
			Key: "external_order_no", LabelEN: "Order / Doc Number", LabelRU: "Номер заказа / документа", LabelUZ: "Buyurtma / Hujjat raqami",
			Required: true, Type: "string", StrongAnchor: true,
			Aliases: []string{
				"external_order_no", "order_no", "order_number", "doc_number", "doc_no", "document_no", "invoice_no",
				"номер заказа", "номер документа", "номер накладной", "реализация", "продажа", "чек",
				"buyurtma raqami", "hujjat raqami", "nakladnoy", "savdo raqami", "буюртма рақами", "ҳужжат",
			},
		},
		{
			Key: "order_date", LabelEN: "Order Date", LabelRU: "Дата заказа", LabelUZ: "Buyurtma sanasi",
			Required: true, Type: "date", StrongAnchor: true,
			Aliases: []string{"order_date", "doc_date", "sale_date", "date", "created_at", "дата", "дата заказа", "дата документа", "sana", "buyurtma sanasi", "сана"},
		},
		{
			Key: "customer_name", LabelEN: "Customer", LabelRU: "Контрагент", LabelUZ: "Mijoz",
			Type: "string",
			Aliases: []string{"customer_name", "customer", "client", "buyer", "контрагент", "покупатель", "клиент", "mijoz", "xaridor"},
		},
		{
			Key: "customer_inn", LabelEN: "Customer INN", LabelRU: "ИНН клиента", LabelUZ: "Mijoz STIR",
			Type: "inn",
			Aliases: []string{"customer_inn", "inn", "stir", "инн", "инн покупателя"},
		},
		{
			Key: "customer_phone", LabelEN: "Customer Phone", LabelRU: "Телефон клиента", LabelUZ: "Mijoz telefoni",
			Type: "phone",
			Aliases: []string{"customer_phone", "phone", "телефон", "telefon"},
		},
		{
			Key: "sku", LabelEN: "SKU", LabelRU: "Артикул", LabelUZ: "SKU",
			Type: "string",
			Aliases: []string{"sku", "external_sku_id", "article", "barcode", "product_name", "артикул", "код товара", "номенклатура", "товар"},
		},
		{
			Key: "quantity", LabelEN: "Quantity", LabelRU: "Количество", LabelUZ: "Miqdor",
			Type: "number",
			Aliases: []string{"quantity", "qty", "количество", "кол-во", "miqdor", "soni"},
		},
		{
			Key: "unit_price", LabelEN: "Unit Price", LabelRU: "Цена", LabelUZ: "Narx",
			Type: "money",
			Aliases: []string{"unit_price", "price", "цена", "narx"},
		},
		{
			Key: "total_amount", LabelEN: "Total Amount", LabelRU: "Сумма документа", LabelUZ: "Jami summa",
			Required: true, Type: "money", StrongAnchor: true,
			Aliases: []string{"total_amount", "amount", "total", "sum", "summa", "сумма", "итого", "сумма документа", "сумма заказа", "jami", "summa", "жами", "сумма"},
		},
		{
			Key: "status", LabelEN: "Historical Status", LabelRU: "Статус", LabelUZ: "Holat",
			Type: "string",
			Aliases: []string{"status", "state", "статус", "состояние", "holat", "statusi"},
		},
		{
			Key: "external_fiscal_ref", LabelEN: "Fiscal / OFD Ref", LabelRU: "Фискальный чек / ОФД", LabelUZ: "Fiskal chek / OFD",
			Type: "string", StrongAnchor: true,
			Aliases: []string{"external_fiscal_ref", "fiscal_ref", "ofd_check", "fiscal_sign", "чек офд", "фпд", "фискальный признак", "fiskal"},
		},
	},
	PackNasiyaAR: {
		{
			Key: "external_invoice_id", LabelEN: "Debt / Invoice ID", LabelRU: "Номер акта / накладной", LabelUZ: "Qarz / Hujjat raqami",
			Type: "string",
			Aliases: []string{"external_invoice_id", "invoice_id", "doc_number", "act_no", "номер акта", "акт сверки", "накладная", "hujjat raqami", "dalolatnoma"},
		},
		{
			Key: "customer_inn", LabelEN: "Customer INN", LabelRU: "ИНН должника", LabelUZ: "Qarzdor STIR",
			Type: "inn",
			Aliases: []string{"customer_inn", "inn", "stir", "инн", "инн контрагента"},
		},
		{
			Key: "customer_name", LabelEN: "Customer Name", LabelRU: "Должник / Контрагент", LabelUZ: "Qarzdor / Mijoz",
			Required: true, Type: "string",
			Aliases: []string{"customer_name", "debtor", "client", "должник", "контрагент", "покупатель", "mijoz", "qarzdor", "мижоз", "қарздор"},
		},
		{
			Key: "customer_phone", LabelEN: "Customer Phone", LabelRU: "Телефон", LabelUZ: "Telefon",
			Type: "phone",
			Aliases: []string{"customer_phone", "phone", "телефон", "telefon"},
		},
		{
			Key: "balance_due", LabelEN: "Opening Debt Balance", LabelRU: "Сумма задолженности (Долг)", LabelUZ: "Nasiya / Qarz qoldig'i",
			Required: true, Type: "money", StrongAnchor: true,
			Aliases: []string{
				"balance_due", "debt", "debt_amount", "ar_balance", "receivable", "outstanding_amount", "nasiya", "qarz",
				"долг", "задолженность", "дебиторская задолженность", "дебиторка", "сумма долга", "остаток долга", "сальдо",
				"nasiya summasi", "qarz qoldigi", "qarz summasi", "nasiya", "қарз", "насия", "қарздорлик",
			},
		},
		{
			Key: "original_amount", LabelEN: "Original Invoice Amount", LabelRU: "Первоначальная сумма", LabelUZ: "Asl summa",
			Type: "money",
			Aliases: []string{"original_amount", "invoice_amount", "начальная сумма", "первоначальная сумма", "asl summa"},
		},
		{
			Key: "due_date", LabelEN: "Due Date", LabelRU: "Срок оплаты", LabelUZ: "To'lov muddati",
			Type: "date", StrongAnchor: true,
			Aliases: []string{"due_date", "pay_by", "repayment_date", "срок оплаты", "дата погашения", "tolov muddati", "muddat", "тўлов муддати"},
		},
		{
			Key: "issued_date", LabelEN: "Issued Date", LabelRU: "Дата возникновения", LabelUZ: "Berilgan sana",
			Type: "date",
			Aliases: []string{"issued_date", "doc_date", "дата долга", "дата документа", "berilgan sana"},
		},
		{
			Key: "currency", LabelEN: "Currency", LabelRU: "Валюта", LabelUZ: "Valyuta",
			Type: "string",
			Aliases: []string{"currency", "валюта", "valyuta"},
		},
	},
	PackOpenOrders: {
		{
			Key: "external_order_no", LabelEN: "Open Order Number", LabelRU: "Номер открытого заказа", LabelUZ: "Ochiq buyurtma raqami",
			Required: true, Type: "string", StrongAnchor: true,
			Aliases: []string{"external_order_no", "order_no", "open_order_no", "номер заказа", "заказ", "buyurtma raqami", "буюртма"},
		},
		{
			Key: "delivery_date", LabelEN: "Planned Delivery Date", LabelRU: "Дата доставки", LabelUZ: "Yetkazib berish sanasi",
			Required: true, Type: "date", StrongAnchor: true,
			Aliases: []string{
				"delivery_date", "requested_delivery_date", "ship_date", "eta",
				"дата доставки", "дата отгрузки", "плановая дата",
				"yetkazish sanasi", "yetkazib berish sanasi", "dostavka sanasi", "етказиб бериш санаси",
			},
		},
		{
			Key: "customer_name", LabelEN: "Customer Name", LabelRU: "Покупатель", LabelUZ: "Mijoz",
			Type: "string",
			Aliases: []string{"customer_name", "customer", "client", "покупатель", "клиент", "контрагент", "mijoz"},
		},
		{
			Key: "customer_inn", LabelEN: "Customer INN", LabelRU: "ИНН покупателя", LabelUZ: "Mijoz STIR",
			Type: "inn",
			Aliases: []string{"customer_inn", "inn", "stir", "инн"},
		},
		{
			Key: "customer_phone", LabelEN: "Customer Phone", LabelRU: "Телефон", LabelUZ: "Telefon",
			Type: "phone",
			Aliases: []string{"customer_phone", "phone", "телефон", "telefon"},
		},
		{
			Key: "delivery_address", LabelEN: "Delivery Address", LabelRU: "Адрес доставки", LabelUZ: "Yetkazish manzili",
			Type: "string", StrongAnchor: true,
			Aliases: []string{"delivery_address", "ship_to", "address", "адрес доставки", "точка доставки", "manzil", "yetkazish manzili"},
		},
		{
			Key: "sku", LabelEN: "SKU", LabelRU: "Артикул", LabelUZ: "SKU",
			Required: true, Type: "string",
			Aliases: []string{"sku", "external_sku_id", "article", "barcode", "артикул", "код товара", "номенклатура"},
		},
		{
			Key: "quantity", LabelEN: "Ordered Quantity", LabelRU: "Количество в заказе", LabelUZ: "Buyurtma miqdori",
			Required: true, Type: "number",
			Aliases: []string{"quantity", "qty", "ordered_qty", "количество", "miqdor", "soni"},
		},
		{
			Key: "unit_price", LabelEN: "Unit Price", LabelRU: "Цена", LabelUZ: "Narx",
			Type: "money",
			Aliases: []string{"unit_price", "price", "цена", "narx"},
		},
		{
			Key: "warehouse_id", LabelEN: "Warehouse", LabelRU: "Склад отгрузки", LabelUZ: "Ombor",
			Type: "string",
			Aliases: []string{"warehouse_id", "warehouse", "склад", "ombor"},
		},
	},
}

var packRequiredGroups = map[Pack][]RequiredGroup{
	PackWarehouses: {
		{Name: "warehouse_identity", Fields: []string{"name"}},
	},
	PackProductsPrices: {
		{Name: "product_name", Fields: []string{"name"}},
		{Name: "product_identifier", Fields: []string{"sku", "barcode"}},
	},
	PackCustomers: {
		{Name: "customer_name", Fields: []string{"customer_name"}},
		{Name: "customer_contact_or_tax", Fields: []string{"inn", "phone"}},
	},
	PackStock: {
		{Name: "sku", Fields: []string{"sku"}},
		{Name: "quantity_on_hand", Fields: []string{"quantity_on_hand"}},
	},
	PackLots: {
		{Name: "sku", Fields: []string{"sku"}},
		{Name: "lot_number", Fields: []string{"lot_number"}},
		{Name: "expiry_date", Fields: []string{"expiry_date"}},
		{Name: "quantity_on_hand", Fields: []string{"quantity_on_hand"}},
	},
	PackHistoricalOrders: {
		{Name: "external_order_no", Fields: []string{"external_order_no"}},
		{Name: "order_date", Fields: []string{"order_date"}},
		{Name: "total_amount", Fields: []string{"total_amount"}},
	},
	PackNasiyaAR: {
		{Name: "customer_identity", Fields: []string{"customer_inn", "customer_phone", "customer_name"}},
		{Name: "balance_due", Fields: []string{"balance_due"}},
	},
	PackOpenOrders: {
		{Name: "external_order_no", Fields: []string{"external_order_no"}},
		{Name: "delivery_date", Fields: []string{"delivery_date"}},
		{Name: "sku", Fields: []string{"sku"}},
		{Name: "quantity", Fields: []string{"quantity"}},
	},
}

// PackTaxonomy returns a copy of canonical field descriptors for a pack.
func PackTaxonomy(p Pack) []CanonicalField {
	list := packTaxonomies[p]
	out := make([]CanonicalField, len(list))
	copy(out, list)
	return out
}

// PackRequiredGroups returns semantic required-field rules (including OR groups like inn|phone).
func PackRequiredGroups(p Pack) []RequiredGroup {
	groups := packRequiredGroups[p]
	out := make([]RequiredGroup, len(groups))
	copy(out, groups)
	return out
}

// RequiredFieldsForPack lists every individual or grouped required field name for UI display.
func RequiredFieldsForPack(p Pack) []string {
	var out []string
	seen := make(map[string]bool)
	for _, g := range packRequiredGroups[p] {
		if len(g.Fields) == 1 && !seen[g.Fields[0]] {
			seen[g.Fields[0]] = true
			out = append(out, g.Fields[0])
		}
	}
	return out
}

// NormalizeHeaderToken normalizes multilingual CSV/Excel column headers for matching and fingerprinting.
func NormalizeHeaderToken(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ToLower(s)
	// Unify Uzbek modifier apostrophes.
	repl := strings.NewReplacer(
		"o'", "o", "o‘", "o", "oʼ", "o", "o`", "o",
		"g'", "g", "g‘", "g", "gʼ", "g", "g`", "g",
		"ў", "у", "қ", "к", "ғ", "г", "ҳ", "х",
		"-", " ", "_", " ", "/", " ", ".", " ", ":", " ", "(", " ", ")", " ",
	)
	s = repl.Replace(s)
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if unicode.IsSpace(r) && !prevSpace && b.Len() > 0 {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// MatchHeaderScore returns the best matching CanonicalField inside pack `p` and a similarity score (0..1).
func MatchHeaderScore(p Pack, rawHeader string) (CanonicalField, float64) {
	norm := NormalizeHeaderToken(rawHeader)
	if norm == "" {
		return CanonicalField{}, 0
	}
	normSnake := strings.ReplaceAll(norm, " ", "_")

	best := CanonicalField{}
	bestScore := 0.0

	for _, f := range packTaxonomies[p] {
		if normSnake == f.Key || norm == NormalizeHeaderToken(f.Key) {
			return f, 1.0
		}
		for _, alias := range f.Aliases {
			na := NormalizeHeaderToken(alias)
			if na == "" {
				continue
			}
			if norm == na {
				if 0.96 > bestScore {
					bestScore = 0.96
					best = f
				}
				continue
			}
			naRunes := len([]rune(na))
			if naRunes >= 4 && strings.Contains(norm, na) {
				// Prefer more specific multi-word aliases when a single header contains multiple alias substrings.
				specificity := float64(naRunes) / float64(maxIntRune(len([]rune(norm)), 1))
				score := 0.78 + specificity*0.14
				if score > 0.92 {
					score = 0.92
				}
				if score > bestScore {
					bestScore = score
					best = f
				}
			}
		}
	}
	return best, bestScore
}

func maxIntRune(a, b int) int {
	if a > b {
		return a
	}
	return b
}
