// Command gen-migration-fixtures synthesizes anonymized zero-PII Uzbekistan
// golden migration fixture files into testdata/migrations for M11 verification.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
)

func main() {
	outDir := flag.String("out", "../../testdata/migrations", "output directory for generated fixtures")
	flag.Parse()

	absOut, err := filepath.Abs(*outDir)
	if err != nil {
		log.Fatalf("resolve output dir: %v", err)
	}
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", absOut, err)
	}

	if err := writeSuite1AllInOneXLSX(filepath.Join(absOut, "suite1_all_in_one_25k.xlsx")); err != nil {
		log.Fatalf("suite1 xlsx: %v", err)
	}
	if err := writeSuite2BIFF8XLS(filepath.Join(absOut, "suite2_biff8_legacy.xls")); err != nil {
		log.Fatalf("suite2 xls: %v", err)
	}
	if err := writeSuite2MessyCP1251CSV(filepath.Join(absOut, "suite2_messy_cp1251_billz.csv")); err != nil {
		log.Fatalf("suite2 csv: %v", err)
	}
	if err := writeSuite3CommerceMLXML(filepath.Join(absOut, "suite3_commerceml_1c.xml")); err != nil {
		log.Fatalf("suite3 xml: %v", err)
	}
	if err := writeSuite4CompositeColdChainJSON(filepath.Join(absOut, "suite4_composite_coldchain.json")); err != nil {
		log.Fatalf("suite4 json: %v", err)
	}
	if err := writeSuite5InvariantsJSON(filepath.Join(absOut, "suite5_financial_and_inn_invariants.json")); err != nil {
		log.Fatalf("suite5 json: %v", err)
	}
	if err := writeFixturesReadme(filepath.Join(absOut, "README.md")); err != nil {
		log.Fatalf("readme: %v", err)
	}

	log.Printf("generated golden migration fixtures in %s", absOut)
}

func writeSuite1AllInOneXLSX(path string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := f.SetSheetName("Sheet1", "Склады"); err != nil {
		return err
	}
	writeStreamSheet := func(sheet string, headers []any, count int, rowFn func(i int) []any) error {
		if _, err := f.NewSheet(sheet); err != nil && sheet != "Склады" {
			return err
		}
		sw, err := f.NewStreamWriter(sheet)
		if err != nil {
			return err
		}
		if err := sw.SetRow("A1", headers); err != nil {
			return err
		}
		for i := 1; i <= count; i++ {
			cell, err := excelize.CoordinatesToCellName(1, i+1)
			if err != nil {
				return err
			}
			if err := sw.SetRow(cell, rowFn(i)); err != nil {
				return err
			}
		}
		return sw.Flush()
	}

	// 1. Warehouses (5 rows)
	whHeaders := []any{"external_warehouse_id", "name", "city", "address", "temperature_mode", "is_default"}
	warehouses := [][]any{
		{"UZ-WH-01", "Tashkent Chilonzor Hub", "Tashkent", "Chilonzor district, Bunyodkor 10", "ambient", "true"},
		{"UZ-WH-02", "Samarqand Registon Depot", "Samarkand", "Registon avenue 24", "ambient", "false"},
		{"UZ-WH-03", "Buxoro Markaz Cold", "Bukhara", "Bahauddin Naqshband 88", "cold", "false"},
		{"UZ-WH-04", "Fargona Vodiy Storage", "Fergana", "Mustaqillik street 15", "ambient", "false"},
		{"UZ-WH-05", "Namangan North Node", "Namangan", "Navoi street 42", "ambient", "false"},
	}
	if err := writeStreamSheet("Склады", whHeaders, len(warehouses), func(i int) []any {
		return warehouses[i-1]
	}); err != nil {
		return err
	}

	// 2. Products & Prices (18,000 rows)
	prodHeaders := []any{"sku", "barcode", "name", "unit", "price", "currency"}
	if err := writeStreamSheet("Номенклатура_Цены", prodHeaders, 18000, func(i int) []any {
		priceUZS := 5000 + (i%250)*500
		unit := "pcs"
		if i%5 == 0 {
			unit = "box"
		}
		return []any{
			fmt.Sprintf("SKU-UZ-%05d", i),
			fmt.Sprintf("478000%07d", i),
			fmt.Sprintf("Synthetic Beverage Item %05d", i),
			unit,
			fmt.Sprintf("%d", priceUZS),
			"UZS",
		}
	}); err != nil {
		return err
	}

	// 3. Stock (4,000 rows)
	stockHeaders := []any{"sku", "warehouse_id", "quantity_on_hand"}
	if err := writeStreamSheet("Остатки", stockHeaders, 4000, func(i int) []any {
		wh := fmt.Sprintf("UZ-WH-%02d", ((i-1)%5)+1)
		return []any{
			fmt.Sprintf("SKU-UZ-%05d", i),
			wh,
			fmt.Sprintf("%d", 20+(i%120)),
		}
	}); err != nil {
		return err
	}

	// 4. Customers (1,500 rows with synthetic 9-digit INN and +99890 test phones)
	custHeaders := []any{"external_customer_id", "customer_name", "inn", "phone", "address", "city", "credit_limit"}
	cities := []string{"Tashkent", "Samarkand", "Bukhara", "Fergana", "Andijan"}
	if err := writeStreamSheet("Контрагенты", custHeaders, 1500, func(i int) []any {
		inn := fmt.Sprintf("300%06d", i)
		phone := fmt.Sprintf("+99890%07d", 1000000+i)
		return []any{
			fmt.Sprintf("CUST-UZ-%04d", i),
			fmt.Sprintf("OOO Demo Savdo %04d", i),
			inn,
			phone,
			fmt.Sprintf("Test street %d", i),
			cities[i%len(cities)],
			fmt.Sprintf("%d", 5000000+(i%10)*1000000),
		}
	}); err != nil {
		return err
	}

	// 5. Nasiya AR (1,000 rows against Customers 1..1000)
	arHeaders := []any{"external_invoice_id", "customer_inn", "customer_name", "balance_due", "due_date", "currency"}
	if err := writeStreamSheet("Дебиторка_Насия", arHeaders, 1000, func(i int) []any {
		return []any{
			fmt.Sprintf("AR-UZ-%04d", i),
			fmt.Sprintf("300%06d", i),
			fmt.Sprintf("OOO Demo Savdo %04d", i),
			fmt.Sprintf("%d", 250000+(i%50)*10000),
			"2026-12-15",
			"UZS",
		}
	}); err != nil {
		return err
	}

	// 6. Historical Orders (1,000 rows -> total rows = 5+18000+4000+1500+1000+1000 = 25,505)
	ordHeaders := []any{"external_order_no", "order_date", "customer_inn", "customer_name", "sku", "quantity", "unit_price", "total_amount", "status"}
	if err := writeStreamSheet("Заказы_История", ordHeaders, 1000, func(i int) []any {
		qty := int64(2 + (i % 5))
		unitPrice := int64(5000 + (i%250)*500)
		return []any{
			fmt.Sprintf("HIST-UZ-%04d", i),
			"2026-10-10",
			fmt.Sprintf("300%06d", i),
			fmt.Sprintf("OOO Demo Savdo %04d", i),
			fmt.Sprintf("SKU-UZ-%05d", i),
			fmt.Sprintf("%d", qty),
			fmt.Sprintf("%d", unitPrice),
			fmt.Sprintf("%d", qty*unitPrice),
			"DELIVERED",
		}
	}); err != nil {
		return err
	}

	return f.SaveAs(path)
}

func writeSuite2BIFF8XLS(path string) error {
	workbook := buildUTF16BIFF8Workbook("BillzОстатки", [][]string{
		{"Артикул", "Наименование", "Остаток", "Цена"},
		{"BLZ-SKU-01", "Чай Зеленый 100г", "40", "18500"},
		{"BLZ-SKU-02", "Сок Натуральный 1л", "120", "14000"},
	})
	ole := wrapOLE2CFB(workbook)
	return os.WriteFile(path, ole, 0o644)
}

func writeSuite2MessyCP1251CSV(path string) error {
	// Semicolon-separated Billz/Smartup CSV with CP1251 encoding + 10x price anomaly for warning verification.
	utf8CSV := strings.Join([]string{
		"Артикул;Наименование товара;Ед.изм;Оптовая цена;Розничная цена;Штрихкод",
		"BLZ-SKU-01;Чай Зеленый 100г;pcs;18500;22000;4780010000011",
		"BLZ-SKU-01;Чай Зеленый 100г;pcs;185000;220000;4780010000011",
		"BLZ-SKU-03;Вода Газированная 1.5л;pcs;4500;6000;4780010000035",
	}, "\r\n") + "\r\n"

	encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(utf8CSV))
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

func writeSuite3CommerceMLXML(path string) error {
	xmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.10" ДатаФормирования="2026-11-14T10:00:00">
  <Классификатор>
    <Ид>cls-uz-01</Ид>
    <Наименование>Тестовый Классификатор</Наименование>
  </Классификатор>
  <Каталог>
    <Ид>cat-uz-01</Ид>
    <Товары>
      <Товар>
        <Ид>1C-PROD-01</Ид>
        <Артикул>1C-SKU-01</Артикул>
        <Штрихкод>4780090000017</Штрихкод>
        <Наименование>Масло Подсолнечное 1л</Наименование>
        <БазоваяЕдиница>pcs</БазоваяЕдиница>
      </Товар>
      <Товар>
        <Ид>1C-PROD-02</Ид>
        <Артикул>1C-SKU-02</Артикул>
        <Штрихкод>4780090000024</Штрихкод>
        <Наименование>Мука Высший Сорт 2кг</Наименование>
        <БазоваяЕдиница>pcs</БазоваяЕдиница>
      </Товар>
    </Товары>
  </Каталог>
  <ПакетПредложений>
    <Предложения>
      <Предложение>
        <Ид>1C-PROD-01</Ид>
        <Наименование>Масло Подсолнечное 1л</Наименование>
        <Цены>
          <Цена>
            <ЦенаЗаЕдиницу>24000</ЦенаЗаЕдиницу>
            <Валюта>UZS</Валюта>
          </Цена>
        </Цены>
        <Остатки>
          <Остаток>
            <Склад Ид="UZ-WH-1C" />
            <Количество>60</Количество>
          </Остаток>
        </Остатки>
      </Предложение>
      <Предложение>
        <Ид>1C-PROD-02</Ид>
        <Наименование>Мука Высший Сорт 2кг</Наименование>
        <Цены>
          <Цена>
            <ЦенаЗаЕдиницу>19000</ЦенаЗаЕдиницу>
            <Валюта>UZS</Валюта>
          </Цена>
        </Цены>
        <Остатки>
          <Остаток>
            <Склад Ид="UZ-WH-1C" />
            <Количество>85</Количество>
          </Остаток>
        </Остатки>
      </Предложение>
    </Предложения>
  </ПакетПредложений>
  <Контрагенты>
    <Контрагент>
      <Ид>1C-CP-01</Ид>
      <Наименование>OOO Registon Market</Наименование>
      <ПолноеНаименование>OOO Registon Market</ПолноеНаименование>
      <ИНН>301555001</ИНН>
      <Адрес><Представление>Samarkand, Universitetskiy blvd 5</Представление></Адрес>
      <Контакты>
        <Контакт><Тип>Телефон</Тип><Значение>+998901550001</Значение></Контакт>
      </Контакты>
    </Контрагент>
  </Контрагенты>
  <Документ>
    <Ид>1C-DOC-HIST-01</Ид>
    <Номер>RN-2026-001</Номер>
    <Дата>2026-10-15</Дата>
    <ХозОперация>Реализация товаров и услуг</ХозОперация>
    <Статус>Выполнено</Статус>
    <Валюта>UZS</Валюта>
    <Сумма>240000</Сумма>
    <Контрагенты>
      <Контрагент><Ид>1C-CP-01</Ид><Наименование>OOO Registon Market</Наименование><ИНН>301555001</ИНН></Контрагент>
    </Контрагенты>
    <Товары>
      <Товар><Ид>1C-PROD-01</Ид><Наименование>Масло Подсолнечное 1л</Наименование><Количество>10</Количество><ЦенаЗаЕдиницу>24000</ЦенаЗаЕдиницу><Сумма>240000</Сумма></Товар>
    </Товары>
  </Документ>
  <Документ>
    <Ид>1C-DOC-OPEN-01</Ид>
    <Номер>ZK-2026-099</Номер>
    <Дата>2026-11-18</Дата>
    <ХозОперация>Заказ покупателя</ХозОперация>
    <Статус>Новый</Статус>
    <Валюта>UZS</Валюта>
    <Сумма>190000</Сумма>
    <Контрагенты>
      <Контрагент><Ид>1C-CP-01</Ид><Наименование>OOO Registon Market</Наименование><ИНН>301555001</ИНН></Контрагент>
    </Контрагенты>
    <Товары>
      <Товар><Ид>1C-PROD-02</Ид><Наименование>Мука Высший Сорт 2кг</Наименование><Количество>10</Количество><ЦенаЗаЕдиницу>19000</ЦенаЗаЕдиницу><Сумма>190000</Сумма></Товар>
    </Товары>
  </Документ>
</КоммерческаяИнформация>`
	return os.WriteFile(path, []byte(xmlDoc), 0o644)
}

func writeSuite4CompositeColdChainJSON(path string) error {
	rows := []map[string]any{
		{
			"warehouse_id":     "WH-COLD-01",
			"warehouse_name":   "Tashkent Cold Chain Pharma Hub",
			"address":          "Yunusobod 19",
			"temperature_mode": "cold",
			"is_default":       "true",
			"sku":              "SKU-VAX-UZ",
			"barcode":          "4780080001012",
			"product_name":     "Cold Chain Probiotic Box",
			"unit":             "box",
			"price":            "120000",
			"currency":         "UZS",
			"lot_number":       "LOT-FEFO-EARLY",
			"production_date":  "2026-06-01",
			"expiry_date":      "2026-12-01",
			"quantity_on_hand": "15",
			"customer_name":    "Apteka Bogishamol Test",
			"phone":            "+998907770001",
			"inn":              "302888001",
			"external_order_no": "OPEN-COLD-101",
			"delivery_date":    "2026-11-18",
			"delivery_address": "Yunusobod 12, Shop 4",
			"quantity":         "10",
			"unit_price":       "120000",
			"total_amount":     "1200000",
			"status":           "PENDING",
		},
		{
			"warehouse_id":     "WH-COLD-01",
			"warehouse_name":   "Tashkent Cold Chain Pharma Hub",
			"address":          "Yunusobod 19",
			"temperature_mode": "cold",
			"is_default":       "true",
			"sku":              "SKU-VAX-UZ",
			"barcode":          "4780080001012",
			"product_name":     "Cold Chain Probiotic Box",
			"unit":             "box",
			"price":            "120000",
			"currency":         "UZS",
			"lot_number":       "LOT-FEFO-LATE",
			"production_date":  "2026-09-01",
			"expiry_date":      "2027-06-01",
			"quantity_on_hand": "30",
			"customer_name":    "Dokon Chorsu Informal",
			"phone":            "+998907770002",
			"inn":              "",
			"external_order_no": "OPEN-COLD-102",
			"delivery_date":    "2026-11-19",
			"delivery_address": "Chorsu Bazaar Stall 14",
			"quantity":         "12",
			"unit_price":       "120000",
			"total_amount":     "1440000",
			"status":           "SCHEDULED",
		},
	}
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeSuite5InvariantsJSON(path string) error {
	doc := map[string]any{
		"description": "Anonymized Suite 5 invariant fixture cases for financial, dedupe, auth bypass, INN, and 72h rollback",
		"imbalanced_nasiya_rows": []map[string]any{
			{"external_invoice_id": "INV-BAD-ZERO", "customer_inn": "309000001", "customer_name": "Zero Debt Store", "balance_due": "0", "currency": "UZS"},
			{"external_invoice_id": "INV-BAD-NEG", "customer_inn": "309000001", "customer_name": "Neg Debt Store", "balance_due": "-50000", "currency": "UZS"},
			{"external_invoice_id": "INV-BAD-FX", "customer_inn": "309000001", "customer_name": "FX Mismatch Store", "balance_due": "100000", "currency": "USD"},
		},
		"duplicate_historical_orders": []map[string]any{
			{"external_order_no": "ORD-DUP-001", "order_date": "2026-10-20", "total_amount": "500000", "status": "DELIVERED"},
			{"external_order_no": "ORD-DUP-001", "order_date": "2026-10-20", "total_amount": "500000", "status": "DELIVERED"},
		},
		"inn_lookup_cases": []map[string]any{
			{"inn": "301900001", "expected_status": "FOUND", "legal_name": "OOO Toshkent Logistika Test"},
			{"inn": "301900404", "expected_status": "NOT_FOUND"},
			{"inn": "301900503", "expected_status": "UNAVAILABLE"},
			{"inn": "301900409", "expected_status": "FOUND_WITH_EXISTING_TENANT", "legal_name": "OOO Navoi Mart Test"},
		},
		"rollback_window_hours": 72,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeFixturesReadme(path string) error {
	content := `# Golden Migration Fixtures (M11.1)

All files in this directory contain strictly anonymized, synthetic Uzbekistan test data (zero real PII).

1. ` + "`suite1_all_in_one_25k.xlsx`" + ` — Multi-sheet 25,505-row Excel workbook (` + "`Склады`" + `, ` + "`Номенклатура_Цены`" + `, ` + "`Остатки`" + `, ` + "`Контрагенты`" + `, ` + "`Дебиторка_Насия`" + `, ` + "`Заказы_История`" + `).
2. ` + "`suite2_biff8_legacy.xls`" + ` — Legacy OLE2 BIFF8 XLS workbook with Cyrillic ` + "`cp1251`" + `/UTF-16LE sheet and headers.
3. ` + "`suite2_messy_cp1251_billz.csv`" + ` — Semicolon-delimited Windows-1251 Billz/Smartup-style CSV with a 10x price jump anomaly.
4. ` + "`suite3_commerceml_1c.xml`" + ` — 1C CommerceML 2.10 XML bundle covering goods, offers + warehouse rests, counterparties, completed sales, and open buyer orders.
5. ` + "`suite4_composite_coldchain.json`" + ` — Single composite flat JSON array combining warehouses, products, cold-chain ` + "`StockLots`" + `, formal/informal customers, and open orders for FEFO reservation + cancellation testing.
6. ` + "`suite5_financial_and_inn_invariants.json`" + ` — Financial (` + "`nasiya`" + ` imbalance / currency mismatch), duplicate ` + "`OrderId`" + `, INN lookup states, and 72h rollback reference scenarios.

Regenerate cleanly via:
` + "```bash\ncd apps/backend-go && go run ./cmd/gen-migration-fixtures -out ../../testdata/migrations\n```" + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

func buildUTF16BIFF8Workbook(sheetName string, rows [][]string) []byte {
	var globals bytes.Buffer
	appendRec := func(opcode uint16, data []byte) {
		var hdr [4]byte
		binary.LittleEndian.PutUint16(hdr[0:2], opcode)
		binary.LittleEndian.PutUint16(hdr[2:4], uint16(len(data)))
		globals.Write(hdr[:])
		globals.Write(data)
	}

	bof := make([]byte, 16)
	binary.LittleEndian.PutUint16(bof[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bof[2:4], 0x0005)
	appendRec(0x0809, bof)

	var cp [2]byte
	binary.LittleEndian.PutUint16(cp[:], 1251)
	appendRec(0x0042, cp[:])

	var stringsList []string
	strIndex := make(map[string]uint32)
	for _, row := range rows {
		for _, cell := range row {
			if _, ok := strIndex[cell]; !ok {
				strIndex[cell] = uint32(len(stringsList))
				stringsList = append(stringsList, cell)
			}
		}
	}

	var sst bytes.Buffer
	_ = binary.Write(&sst, binary.LittleEndian, uint32(len(stringsList)))
	_ = binary.Write(&sst, binary.LittleEndian, uint32(len(stringsList)))
	for _, s := range stringsList {
		runes := []rune(s)
		_ = binary.Write(&sst, binary.LittleEndian, uint16(len(runes)))
		sst.WriteByte(0x01) // UTF-16LE
		sst.Write(encodeUTF16LE(s))
	}
	appendRec(0x00FC, sst.Bytes())

	sheetRunes := []rune(sheetName)
	sheetU16 := encodeUTF16LE(sheetName)
	bsPos := globals.Len()
	bsData := make([]byte, 8+len(sheetU16))
	bsData[6] = byte(len(sheetRunes))
	bsData[7] = 1
	copy(bsData[8:], sheetU16)
	appendRec(0x0085, bsData)
	appendRec(0x000A, nil)

	binary.LittleEndian.PutUint32(globals.Bytes()[bsPos+4:bsPos+8], uint32(globals.Len()))

	var ws bytes.Buffer
	appendWS := func(opcode uint16, data []byte) {
		var hdr [4]byte
		binary.LittleEndian.PutUint16(hdr[0:2], opcode)
		binary.LittleEndian.PutUint16(hdr[2:4], uint16(len(data)))
		ws.Write(hdr[:])
		ws.Write(data)
	}
	bofSheet := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofSheet[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofSheet[2:4], 0x0010)
	appendWS(0x0809, bofSheet)

	for rIdx, row := range rows {
		for cIdx, val := range row {
			b := make([]byte, 10)
			binary.LittleEndian.PutUint16(b[0:2], uint16(rIdx))
			binary.LittleEndian.PutUint16(b[2:4], uint16(cIdx))
			binary.LittleEndian.PutUint32(b[6:10], strIndex[val])
			appendWS(0x00FD, b)
		}
	}
	appendWS(0x000A, nil)
	globals.Write(ws.Bytes())
	return globals.Bytes()
}

func encodeUTF16LE(s string) []byte {
	var b bytes.Buffer
	for _, r := range s {
		var u [2]byte
		binary.LittleEndian.PutUint16(u[:], uint16(r))
		b.Write(u[:])
	}
	return b.Bytes()
}

func wrapOLE2CFB(workbook []byte) []byte {
	const sectorSize = 512
	if len(workbook) < 4096 {
		pad := make([]byte, 4096-len(workbook))
		workbook = append(append([]byte(nil), workbook...), pad...)
	}
	wbSectors := (len(workbook) + sectorSize - 1) / sectorSize
	totalSectors := 2 + wbSectors
	out := make([]byte, (1+totalSectors)*sectorSize)

	ole2Magic := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	copy(out[0:8], ole2Magic)
	binary.LittleEndian.PutUint16(out[24:26], 0x003E)
	binary.LittleEndian.PutUint16(out[26:28], 0x0003)
	binary.LittleEndian.PutUint16(out[28:30], 0xFFFE)
	binary.LittleEndian.PutUint16(out[30:32], 9)
	binary.LittleEndian.PutUint16(out[32:34], 6)
	binary.LittleEndian.PutUint32(out[44:48], 1)
	binary.LittleEndian.PutUint32(out[48:52], 1)
	binary.LittleEndian.PutUint32(out[56:60], 4096)
	binary.LittleEndian.PutUint32(out[60:64], 0xFFFFFFFE)
	binary.LittleEndian.PutUint32(out[68:72], 0xFFFFFFFE)
	for i := 0; i < 109; i++ {
		val := uint32(0xFFFFFFFF)
		if i == 0 {
			val = 0
		}
		binary.LittleEndian.PutUint32(out[76+i*4:80+i*4], val)
	}

	fat := out[sectorSize : 2*sectorSize]
	for i := 0; i < 128; i++ {
		binary.LittleEndian.PutUint32(fat[i*4:i*4+4], 0xFFFFFFFF)
	}
	binary.LittleEndian.PutUint32(fat[0:4], 0xFFFFFFFD)
	binary.LittleEndian.PutUint32(fat[4:8], 0xFFFFFFFE)
	for i := 0; i < wbSectors; i++ {
		next := uint32(2 + i + 1)
		if i == wbSectors-1 {
			next = 0xFFFFFFFE
		}
		binary.LittleEndian.PutUint32(fat[(2+i)*4:(2+i)*4+4], next)
	}

	dir := out[2*sectorSize : 3*sectorSize]
	writeDir := func(entry []byte, name string, objType byte, startSect, size, childSID uint32) {
		u16 := encodeUTF16LE(name)
		copy(entry[0:len(u16)], u16)
		binary.LittleEndian.PutUint16(entry[64:66], uint16(len(u16)+2))
		entry[66] = objType
		entry[67] = 1
		binary.LittleEndian.PutUint32(entry[68:72], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(entry[72:76], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(entry[76:80], childSID)
		binary.LittleEndian.PutUint32(entry[116:120], startSect)
		binary.LittleEndian.PutUint32(entry[120:124], size)
	}
	writeDir(dir[0:128], "Root Entry", 5, 0xFFFFFFFE, 0, 1)
	writeDir(dir[128:256], "Workbook", 2, 2, uint32(len(workbook)), 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[256+68:256+72], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[256+72:256+76], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[256+76:256+80], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[384+68:384+72], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[384+72:384+76], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[384+76:384+80], 0xFFFFFFFF)

	copy(out[3*sectorSize:], workbook)
	return out
}
