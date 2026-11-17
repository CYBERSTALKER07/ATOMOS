package parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
)

func TestParseArtifact_XLSX_25kRowsStreaming(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet1 := "Товары_и_Остатки"
	_ = f.SetSheetName("Sheet1", sheet1)
	_ = f.SetCellValue(sheet1, "A1", "Отчет по остаткам (экспорт 1С)")
	_ = f.SetCellValue(sheet1, "A2", "Артикул")
	_ = f.SetCellValue(sheet1, "B2", "Наименование")
	_ = f.SetCellValue(sheet1, "C2", "Склад")
	_ = f.SetCellValue(sheet1, "D2", "Остаток")

	const totalRows = 25000
	sw, err := f.NewStreamWriter(sheet1)
	if err != nil {
		t.Fatal(err)
	}
	_ = sw.SetRow("A1", []any{"Отчет по остаткам (экспорт 1С)"})
	_ = sw.SetRow("A2", []any{"Артикул", "Наименование", "Склад", "Остаток"})
	for i := 0; i < totalRows; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+3)
		_ = sw.SetRow(cell, []any{
			fmt.Sprintf("SKU-%06d", i+1),
			fmt.Sprintf("Product %d", i+1),
			"WH-TASHKENT-1",
			(i % 50) + 1,
		})
	}
	if err := sw.Flush(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}

	var chunkCount int
	var streamedTotal int64
	streams, sniff, err := ParseArtifact("big_export.xlsx", buf.Bytes(), "", func(streamName string, headers []string, chunk []ParsedRow) error {
		chunkCount++
		if len(chunk) > defaultStreamChunkSize {
			t.Fatalf("chunk size %d exceeds %d", len(chunk), defaultStreamChunkSize)
		}
		streamedTotal += int64(len(chunk))
		return nil
	})
	if err != nil {
		t.Fatalf("ParseArtifact xlsx 25k: %v", err)
	}
	if sniff.Format != FormatXLSX {
		t.Fatalf("format=%s want xlsx", sniff.Format)
	}
	if len(streams) != 1 || streams[0].TotalRows != totalRows || streamedTotal != totalRows {
		t.Fatalf("total rows mismatch: streams=%+v streamedTotal=%d", streams, streamedTotal)
	}
	if chunkCount != 5 {
		t.Fatalf("expected 5 x 5000 chunks, got %d", chunkCount)
	}
}

func TestParseArtifact_CP1251CSV_And_SpoofGuards(t *testing.T) {
	utf8CSV := "Код товара;Наименование;Цена;Остаток\r\nSKU-1;Чай зеленый №95;15000;42\r\nSKU-2;Масло подсолнечное;32000;12\r\n"
	cp1251Bytes, err := charmap.Windows1251.NewEncoder().Bytes([]byte(utf8CSV))
	if err != nil {
		t.Fatal(err)
	}

	var gotRows []ParsedRow
	streams, sniff, err := ParseArtifact("billz_dump.csv", cp1251Bytes, "", func(streamName string, headers []string, chunk []ParsedRow) error {
		gotRows = append(gotRows, chunk...)
		return nil
	})
	if err != nil {
		t.Fatalf("cp1251 csv parse failed: %v", err)
	}
	if sniff.Format != FormatCSV || streams[0].Encoding != "windows-1251" {
		t.Fatalf("unexpected encoding/format: sniff=%+v stream=%+v", sniff, streams[0])
	}
	if len(gotRows) != 2 || gotRows[0].Fields["Наименование"] != "Чай зеленый №95" {
		t.Fatalf("unexpected decoded cp1251 rows: %+v", gotRows)
	}

	// PDF bytes renamed as .xlsx must fail closed.
	if _, _, err := ParseArtifact("scan.xlsx", []byte("%PDF-1.7\n1 0 obj"), "", nil); !errors.Is(err, ErrFileFormatMismatch) {
		t.Fatalf("expected ErrFileFormatMismatch for PDF spoofed as xlsx, got %v", err)
	}

	// CSV bytes renamed as .xlsx should flag extension mismatch while allowing inspection.
	_, csvAsXLSX, err := ParseArtifact("raw.xlsx", []byte(utf8CSV), "", nil)
	if err != nil || !csvAsXLSX.ExtensionMismatch {
		t.Fatalf("expected extension mismatch warning for csv bytes in .xlsx filename, got sniff=%+v err=%v", csvAsXLSX, err)
	}
}

func TestParseArtifact_CommerceMLXML_MultiStream(t *testing.T) {
	xmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.10">
  <Каталог>
    <Товары>
      <Товар>
        <Ид>1c-guid-1</Ид>
        <Артикул>COLA-15</Артикул>
        <Наименование>Напиток Cola 1.5л</Наименование>
        <Штрихкод>4600001234567</Штрихкод>
        <БазоваяЕдиница>ШТ</БазоваяЕдиница>
      </Товар>
    </Товары>
  </Каталог>
  <ПакетПредложений>
    <Предложения>
      <Предложение>
        <Ид>1c-guid-1</Ид>
        <Наименование>Напиток Cola 1.5л</Наименование>
        <Цены><Цена><ЦенаЗаЕдиницу>12500</ЦенаЗаЕдиницу><Валюта>UZS</Валюта></Цена></Цены>
        <Остатки><Склад><Ид>WH-MAIN</Ид><КоличествоНаСкладе>240</КоличествоНаСкладе></Склад></Остатки>
      </Предложение>
    </Предложения>
  </ПакетПредложений>
  <Контрагенты>
    <Контрагент>
      <Ид>cp-1</Ид>
      <ОфициальноеНаименование>ООО Baraka Savdo</ОфициальноеНаименование>
      <ИНН>301234567</ИНН>
      <Контакты><Контакт><Тип>Телефон</Тип><Значение>+998901112233</Значение></Контакт></Контакты>
    </Контрагент>
  </Контрагенты>
  <Документ>
    <Ид>doc-100</Ид>
    <Номер>ORD-00100</Номер>
    <Дата>2026-10-15</Дата>
    <ХозОперация>Заказ товара</ХозОперация>
    <Сумма>250000</Сумма>
    <Статус>Принят</Статус>
    <Контрагенты><Контрагент><Ид>cp-1</Ид><ИНН>301234567</ИНН></Контрагент></Контрагенты>
  </Документ>
</КоммерческаяИнформация>`

	streams, sniff, err := ParseArtifact("onec_export.xml", []byte(xmlDoc), "", nil)
	if err != nil {
		t.Fatalf("ParseArtifact xml: %v", err)
	}
	if sniff.Format != FormatCommerceML || len(streams) != 4 {
		t.Fatalf("expected 4 CommerceML streams, got format=%s count=%d", sniff.Format, len(streams))
	}
	if streams[0].StreamName != "commerceml_products" || streams[1].StreamName != "commerceml_offers_stock" {
		t.Fatalf("unexpected stream order: %+v", streams)
	}
	if streams[1].SampleRows[0].Fields["warehouse_id"] != "WH-MAIN" || streams[1].SampleRows[0].Fields["quantity_on_hand"] != "240" {
		t.Fatalf("unexpected offer rest fields: %+v", streams[1].SampleRows[0].Fields)
	}
}

func TestParseArtifact_JSON_Composite(t *testing.T) {
	raw := `{
	  "products": [{"sku": "S-1", "name": "Flour", "meta": {"unit": "KG"}}],
	  "stock": [{"sku": "S-1", "warehouse_id": "WH-1", "qty": 15}]
	}`
	streams, sniff, err := ParseArtifact("dump.json", []byte(raw), "", nil)
	if err != nil {
		t.Fatalf("ParseArtifact json: %v", err)
	}
	if sniff.Format != FormatJSON || len(streams) != 2 {
		t.Fatalf("expected 2 JSON streams, got %d", len(streams))
	}
	if streams[0].SampleRows[0].Fields["meta.unit"] != "KG" {
		t.Fatalf("expected nested dot flattening, got %+v", streams[0].SampleRows[0].Fields)
	}
}

func TestParseArtifact_BIFF8XLS_And_EncryptedRejection(t *testing.T) {
	wb := buildMinimalBIFF8Workbook(false)
	cfb := wrapInMinimalOLE2CFB(wb)

	streams, sniff, err := ParseArtifact("legacy.xls", cfb, "", nil)
	if err != nil {
		t.Fatalf("ParseArtifact xls: %v", err)
	}
	if sniff.Format != FormatXLS || len(streams) != 1 {
		t.Fatalf("expected 1 BIFF8 sheet, got sniff=%+v len=%d", sniff, len(streams))
	}
	if streams[0].StreamName != "Лист1" || streams[0].TotalRows != 1 {
		t.Fatalf("unexpected sheet metadata: %+v", streams[0])
	}
	row := streams[0].SampleRows[0].Fields
	if row["SKU"] != "ITEM-7" || row["Qty"] != "15" {
		t.Fatalf("unexpected BIFF8 row values: %+v", row)
	}

	encWB := buildMinimalBIFF8Workbook(true)
	encCFB := wrapInMinimalOLE2CFB(encWB)
	if _, _, err := ParseArtifact("locked.xls", encCFB, "", nil); !errors.Is(err, ErrPasswordProtectedWorkbook) {
		t.Fatalf("expected ErrPasswordProtectedWorkbook for FILEPASS .xls, got %v", err)
	}
}

func appendBIFFRec(buf *bytes.Buffer, opcode uint16, data []byte) {
	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], opcode)
	binary.LittleEndian.PutUint16(hdr[2:4], uint16(len(data)))
	buf.Write(hdr[:])
	buf.Write(data)
}

func buildMinimalBIFF8Workbook(withFilePass bool) []byte {
	var globals bytes.Buffer
	// BOF8 Workbook globals (type=0x0005)
	bofGlobals := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofGlobals[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofGlobals[2:4], 0x0005)
	appendBIFFRec(&globals, biffOpcodeBOF8, bofGlobals)

	if withFilePass {
		appendBIFFRec(&globals, biffOpcodeFilePass, []byte{0x01, 0x00, 0x01, 0x00, 0x00, 0x00})
	}

	var cp [2]byte
	binary.LittleEndian.PutUint16(cp[:], 1251)
	appendBIFFRec(&globals, biffOpcodeCodepage, cp[:])

	// Build SST with 3 strings: "SKU", "Qty", "ITEM-7"
	var sst bytes.Buffer
	_ = binary.Write(&sst, binary.LittleEndian, uint32(3))
	_ = binary.Write(&sst, binary.LittleEndian, uint32(3))
	for _, s := range []string{"SKU", "Qty", "ITEM-7"} {
		_ = binary.Write(&sst, binary.LittleEndian, uint16(len(s)))
		sst.WriteByte(0x00) // compressed single-byte
		sst.WriteString(s)
	}
	appendBIFFRec(&globals, biffOpcodeSST, sst.Bytes())

	// Placeholder BoundSheet8 (offset filled after worksheet constructed)
	sheetNameUTF16 := encodeUTF16LE("Лист1")
	bsPos := globals.Len()
	bsData := make([]byte, 8+len(sheetNameUTF16))
	bsData[6] = 5 // 5 chars
	bsData[7] = 1 // UTF-16LE
	copy(bsData[8:], sheetNameUTF16)
	appendBIFFRec(&globals, biffOpcodeBoundSheet, bsData)
	appendBIFFRec(&globals, biffOpcodeEOF, nil)

	// Patch BOUNDSHEET8 stream offset to point immediately after globals EOF.
	sheetOffset := uint32(globals.Len())
	binary.LittleEndian.PutUint32(globals.Bytes()[bsPos+4:bsPos+8], sheetOffset)

	// Build Worksheet substream
	var ws bytes.Buffer
	bofSheet := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofSheet[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofSheet[2:4], 0x0010)
	appendBIFFRec(&ws, biffOpcodeBOF8, bofSheet)

	writeLabelSST := func(r, c uint16, idx uint32) {
		b := make([]byte, 10)
		binary.LittleEndian.PutUint16(b[0:2], r)
		binary.LittleEndian.PutUint16(b[2:4], c)
		binary.LittleEndian.PutUint32(b[6:10], idx)
		appendBIFFRec(&ws, biffOpcodeLabelSST, b)
	}
	writeRK := func(r, c uint16, intVal int32) {
		b := make([]byte, 10)
		binary.LittleEndian.PutUint16(b[0:2], r)
		binary.LittleEndian.PutUint16(b[2:4], c)
		binary.LittleEndian.PutUint32(b[6:10], uint32((intVal<<2)|0x02))
		appendBIFFRec(&ws, biffOpcodeRK, b)
	}

	writeLabelSST(0, 0, 0) // "SKU"
	writeLabelSST(0, 1, 1) // "Qty"
	writeLabelSST(1, 0, 2) // "ITEM-7"
	writeRK(1, 1, 15)      // 15
	appendBIFFRec(&ws, biffOpcodeEOF, nil)

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

// wrapInMinimalOLE2CFB builds a spec-compliant v3 512-byte-sector OLE2 Compound File
// containing Root Entry and a single `Workbook` stream (using MiniFAT-bypass >= 4096 bytes).
func wrapInMinimalOLE2CFB(workbook []byte) []byte {
	const sectorSize = 512
	// Pad workbook >= 4096 bytes so CFB stores it in normal FAT sectors (not MiniFAT).
	if len(workbook) < 4096 {
		pad := make([]byte, 4096-len(workbook))
		workbook = append(append([]byte(nil), workbook...), pad...)
	}
	wbSectors := (len(workbook) + sectorSize - 1) / sectorSize
	// Sector layout:
	// 0: FAT sector
	// 1: Directory sector
	// 2..(2+wbSectors-1): Workbook stream sectors
	totalSectors := 2 + wbSectors
	out := make([]byte, (1+totalSectors)*sectorSize)

	// Header (512 bytes)
	copy(out[0:8], ole2Magic)
	binary.LittleEndian.PutUint16(out[24:26], 0x003E) // minor version
	binary.LittleEndian.PutUint16(out[26:28], 0x0003) // major v3
	binary.LittleEndian.PutUint16(out[28:30], 0xFFFE) // little endian
	binary.LittleEndian.PutUint16(out[30:32], 9)      // sector shift (512)
	binary.LittleEndian.PutUint16(out[32:34], 6)      // mini sector shift (64)
	binary.LittleEndian.PutUint32(out[44:48], 1)      // 1 FAT sector
	binary.LittleEndian.PutUint32(out[48:52], 1)      // first dir sector = 1
	binary.LittleEndian.PutUint32(out[56:60], 4096)   // mini stream cutoff
	binary.LittleEndian.PutUint32(out[60:64], 0xFFFFFFFE)
	binary.LittleEndian.PutUint32(out[68:72], 0xFFFFFFFE)
	for i := 0; i < 109; i++ {
		val := uint32(0xFFFFFFFF)
		if i == 0 {
			val = 0 // FAT lives in sector 0
		}
		binary.LittleEndian.PutUint32(out[76+i*4:80+i*4], val)
	}

	// Sector 0 (offset 512): FAT
	fat := out[sectorSize : 2*sectorSize]
	for i := 0; i < 128; i++ {
		binary.LittleEndian.PutUint32(fat[i*4:i*4+4], 0xFFFFFFFF)
	}
	binary.LittleEndian.PutUint32(fat[0:4], 0xFFFFFFFD) // FATSECT
	binary.LittleEndian.PutUint32(fat[4:8], 0xFFFFFFFE) // Dir ENDOFCHAIN
	for i := 0; i < wbSectors; i++ {
		next := uint32(2 + i + 1)
		if i == wbSectors-1 {
			next = 0xFFFFFFFE
		}
		binary.LittleEndian.PutUint32(fat[(2+i)*4:(2+i)*4+4], next)
	}

	// Sector 1 (offset 1024): Directory entries (128 bytes each)
	dir := out[2*sectorSize : 3*sectorSize]
	writeDirEntry := func(entry []byte, name string, objType byte, startSect uint32, size uint32, childSID uint32) {
		u16 := encodeUTF16LE(name)
		copy(entry[0:len(u16)], u16)
		binary.LittleEndian.PutUint16(entry[64:66], uint16(len(u16)+2))
		entry[66] = objType
		entry[67] = 1 // black
		binary.LittleEndian.PutUint32(entry[68:72], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(entry[72:76], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(entry[76:80], childSID)
		binary.LittleEndian.PutUint32(entry[116:120], startSect)
		binary.LittleEndian.PutUint32(entry[120:124], size)
	}
	// Entry 0: Root Entry -> child 1
	writeDirEntry(dir[0:128], "Root Entry", 5, 0xFFFFFFFE, 0, 1)
	// Entry 1: Workbook -> starts at sector 2
	writeDirEntry(dir[128:256], "Workbook", 2, 2, uint32(len(workbook)), 0xFFFFFFFF)
	// Unused entries 2 & 3
	binary.LittleEndian.PutUint32(dir[256+68:256+72], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[256+72:256+76], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[256+76:256+80], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[384+68:384+72], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[384+72:384+76], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(dir[384+76:384+80], 0xFFFFFFFF)

	// Sectors 2+: Workbook stream
	copy(out[3*sectorSize:], workbook)
	return out
}

func TestParseArtifact_XLSX_EmptyAndMultiSheet(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	_ = f.SetSheetName("Sheet1", "Warehouses")
	_ = f.SetCellValue("Warehouses", "A1", "warehouse_id")
	_ = f.SetCellValue("Warehouses", "B1", "name")
	_ = f.SetCellValue("Warehouses", "A2", "WH-1")
	_ = f.SetCellValue("Warehouses", "B2", "Chilonzor")
	_, _ = f.NewSheet("EmptySheet")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	streams, _, err := ParseArtifact("multi.xlsx", buf.Bytes(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].StreamName != "Warehouses" {
		t.Fatalf("expected only non-empty Warehouses sheet, got %+v", streams)
	}
	_ = strings.TrimSpace("")
}
