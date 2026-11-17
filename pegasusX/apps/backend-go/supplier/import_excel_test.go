package supplier

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type recordingInventoryRepo struct {
	countingSupplierRepo
	topology SupplierTopology
}

func (r *recordingInventoryRepo) GetTopology(context.Context, string) (SupplierTopology, error) {
	return r.topology, nil
}

type recordingInventoryServicer struct {
	mu      sync.Mutex
	upserts []InventoryLevelUpsert
}

func (s *recordingInventoryServicer) ListBySupplier(context.Context, string) ([]InventoryLevelView, error) {
	return nil, nil
}

func (s *recordingInventoryServicer) AdjustStock(context.Context, string, int64, int64) error {
	return nil
}

func (s *recordingInventoryServicer) FindByWarehouseProduct(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

func (s *recordingInventoryServicer) UpsertLevel(_ context.Context, level InventoryLevelUpsert) error {
	s.mu.Lock()
	s.upserts = append(s.upserts, level)
	s.mu.Unlock()
	return nil
}

func buildMultiSheetXLSX10500(t *testing.T, whID string) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet1 := "WarehouseA"
	_ = f.SetSheetName("Sheet1", sheet1)
	sw1, err := f.NewStreamWriter(sheet1)
	if err != nil {
		t.Fatalf("NewStreamWriter sheet1: %v", err)
	}
	if err := sw1.SetRow("A1", []any{"warehouse_id", "product_id", "quantity_on_hand", "reorder_threshold", "product_name"}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 6000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := sw1.SetRow(cell, []any{whID, fmt.Sprintf("SKU-XLSX-%05d", i), (i % 200) + 1, 5, fmt.Sprintf("Item %d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sw1.Flush(); err != nil {
		t.Fatal(err)
	}

	sheet2 := "WarehouseB"
	if _, err := f.NewSheet(sheet2); err != nil {
		t.Fatal(err)
	}
	sw2, err := f.NewStreamWriter(sheet2)
	if err != nil {
		t.Fatalf("NewStreamWriter sheet2: %v", err)
	}
	// Sheet 2 uses alias headers to prove cross-sheet header mapping works seamlessly.
	if err := sw2.SetRow("A1", []any{"warehouse", "sku", "qty", "reorder"}); err != nil {
		t.Fatal(err)
	}
	for i := 6001; i <= 10500; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i-6000+1)
		if err := sw2.SetRow(cell, []any{whID, fmt.Sprintf("SKU-XLSX-%05d", i), (i % 150) + 2, 3}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sw2.Flush(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func buildSupplierBIFF8XLS(withFilePass bool) []byte {
	var globals bytes.Buffer
	bofGlobals := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofGlobals[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofGlobals[2:4], 0x0005)
	appendTestBIFFRec(&globals, 0x0809, bofGlobals)

	if withFilePass {
		appendTestBIFFRec(&globals, 0x002F, []byte{0x01, 0x00, 0x01, 0x00, 0x00, 0x00})
	}

	var cp [2]byte
	binary.LittleEndian.PutUint16(cp[:], 1251)
	appendTestBIFFRec(&globals, 0x0042, cp[:])

	strs := []string{"warehouse_id", "product_id", "quantity_on_hand", "WH-XLS-1", "SKU-XLS-1", "SKU-XLS-2"}
	var sst bytes.Buffer
	_ = binary.Write(&sst, binary.LittleEndian, uint32(len(strs)))
	_ = binary.Write(&sst, binary.LittleEndian, uint32(len(strs)))
	for _, s := range strs {
		_ = binary.Write(&sst, binary.LittleEndian, uint16(len(s)))
		sst.WriteByte(0x00)
		sst.WriteString(s)
	}
	appendTestBIFFRec(&globals, 0x00FC, sst.Bytes())

	sheetName := "Stock"
	var sheetUTF16 bytes.Buffer
	for _, r := range sheetName {
		var u [2]byte
		binary.LittleEndian.PutUint16(u[:], uint16(r))
		sheetUTF16.Write(u[:])
	}
	bsPos := globals.Len()
	bsData := make([]byte, 8+sheetUTF16.Len())
	bsData[6] = byte(len(sheetName))
	bsData[7] = 1
	copy(bsData[8:], sheetUTF16.Bytes())
	appendTestBIFFRec(&globals, 0x0085, bsData)
	appendTestBIFFRec(&globals, 0x000A, nil)

	binary.LittleEndian.PutUint32(globals.Bytes()[bsPos+4:bsPos+8], uint32(globals.Len()))

	var ws bytes.Buffer
	bofSheet := make([]byte, 16)
	binary.LittleEndian.PutUint16(bofSheet[0:2], 0x0600)
	binary.LittleEndian.PutUint16(bofSheet[2:4], 0x0010)
	appendTestBIFFRec(&ws, 0x0809, bofSheet)

	writeLabelSST := func(r, c uint16, idx uint32) {
		b := make([]byte, 10)
		binary.LittleEndian.PutUint16(b[0:2], r)
		binary.LittleEndian.PutUint16(b[2:4], c)
		binary.LittleEndian.PutUint32(b[6:10], idx)
		appendTestBIFFRec(&ws, 0x00FD, b)
	}
	writeRK := func(r, c uint16, val int32) {
		b := make([]byte, 10)
		binary.LittleEndian.PutUint16(b[0:2], r)
		binary.LittleEndian.PutUint16(b[2:4], c)
		binary.LittleEndian.PutUint32(b[6:10], uint32((val<<2)|0x02))
		appendTestBIFFRec(&ws, 0x027E, b)
	}

	writeLabelSST(0, 0, 0)
	writeLabelSST(0, 1, 1)
	writeLabelSST(0, 2, 2)
	writeLabelSST(1, 0, 3)
	writeLabelSST(1, 1, 4)
	writeRK(1, 2, 25)
	writeLabelSST(2, 0, 3)
	writeLabelSST(2, 1, 5)
	writeRK(2, 2, 40)
	appendTestBIFFRec(&ws, 0x000A, nil)

	globals.Write(ws.Bytes())
	return wrapTestOLE2CFB(globals.Bytes())
}

func appendTestBIFFRec(buf *bytes.Buffer, opcode uint16, data []byte) {
	var hdr [4]byte
	binary.LittleEndian.PutUint16(hdr[0:2], opcode)
	binary.LittleEndian.PutUint16(hdr[2:4], uint16(len(data)))
	buf.Write(hdr[:])
	buf.Write(data)
}

func wrapTestOLE2CFB(workbook []byte) []byte {
	const sectorSize = 512
	if len(workbook) < 4096 {
		pad := make([]byte, 4096-len(workbook))
		workbook = append(append([]byte(nil), workbook...), pad...)
	}
	wbSectors := (len(workbook) + sectorSize - 1) / sectorSize
	totalSectors := 2 + wbSectors
	out := make([]byte, (1+totalSectors)*sectorSize)
	copy(out[0:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
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
		var u16 bytes.Buffer
		for _, r := range name {
			var u [2]byte
			binary.LittleEndian.PutUint16(u[:], uint16(r))
			u16.Write(u[:])
		}
		copy(entry[0:u16.Len()], u16.Bytes())
		binary.LittleEndian.PutUint16(entry[64:66], uint16(u16.Len()+2))
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

func buildEncryptedOLE2OOXMLContainer() []byte {
	encPkgUTF16 := []byte{'E', 0, 'n', 0, 'c', 0, 'r', 0, 'y', 0, 'p', 0, 't', 0, 'e', 0, 'd', 0, 'P', 0, 'a', 0, 'c', 0, 'k', 0, 'a', 0, 'g', 0, 'e', 0}
	out := make([]byte, 1024)
	copy(out[0:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	copy(out[512:], encPkgUTF16)
	return out
}

func TestDiscoverImportArtifact_XLSX_10500RowsAcrossSheets(t *testing.T) {
	xlsxBytes := buildMultiSheetXLSX10500(t, "WH-EXCEL-1")
	outcome, headers, rows, err := discoverImportArtifact("multi_sheet_10500.xlsx", xlsxBytes, 0)
	if err != nil {
		t.Fatalf("discoverImportArtifact: %v", err)
	}
	if len(rows) != 10500 {
		t.Fatalf("rows=%d want=10500", len(rows))
	}
	if len(headers) < 4 {
		t.Fatalf("expected merged headers across sheets, got %v", headers)
	}
	status, lowConf := resolveImportDiscoveryStatus(outcome.Mappings)
	if status != "DISCOVERED" || lowConf {
		t.Fatalf("status=%s lowConf=%v mappings=%+v", status, lowConf, outcome.Mappings)
	}

	staged, summary := buildImportStagedRows("sup-1", rows, outcome.Mappings, map[string]struct{}{"WH-EXCEL-1": {}}, func(string) bool { return true })
	if len(staged) != 10500 {
		t.Fatalf("staged=%d want=10500", len(staged))
	}
	if summary["valid_rows"] != 10500 || summary["invalid_rows"] != 0 {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestHandleInventoryImport_XLSX_10500Rows_And_RealXLS_And_Encrypted422(t *testing.T) {
	invSvc := &recordingInventoryServicer{}
	repo := &recordingInventoryRepo{
		topology: SupplierTopology{
			Warehouses: []WarehouseNode{
				{WarehouseID: "WH-EXCEL-1"},
				{WarehouseID: "WH-XLS-1"},
			},
		},
	}
	svc := NewService(ServiceConfig{
		Repo:             repo,
		InventoryService: invSvc,
		SupplierID:       "sup-excel-1",
	})

	// 1. Real 10,500-row multi-sheet .xlsx workbook -> HTTP 200 and 10,500 applied rows.
	xlsxBytes := buildMultiSheetXLSX10500(t, "WH-EXCEL-1")
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import", bytes.NewReader(xlsxBytes))
	req.Header.Set("X-File-Name", "stock_10500.xlsx")
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{SupplierID: "sup-excel-1", Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	svc.HandleInventoryImport(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("xlsx 10500 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var res inventoryImportResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode xlsx result: %v", err)
	}
	if res.Applied != 10500 || res.Skipped != 0 || len(invSvc.upserts) != 10500 {
		t.Fatalf("xlsx applied=%d skipped=%d upserts=%d", res.Applied, res.Skipped, len(invSvc.upserts))
	}

	// 2. Real BIFF8 .xls workbook -> HTTP 200 and 2 applied rows.
	invSvc.upserts = nil
	xlsBytes := buildSupplierBIFF8XLS(false)
	reqXLS := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import", bytes.NewReader(xlsBytes))
	reqXLS.Header.Set("X-File-Name", "legacy.xls")
	reqXLS = reqXLS.WithContext(auth.WithClaims(reqXLS.Context(), auth.Claims{SupplierID: "sup-excel-1", Role: auth.RoleAdmin}))
	recXLS := httptest.NewRecorder()
	svc.HandleInventoryImport(recXLS, reqXLS)
	if recXLS.Code != http.StatusOK {
		t.Fatalf("xls status=%d body=%s", recXLS.Code, recXLS.Body.String())
	}
	var resXLS inventoryImportResult
	_ = json.Unmarshal(recXLS.Body.Bytes(), &resXLS)
	if resXLS.Applied != 2 || len(invSvc.upserts) != 2 {
		t.Fatalf("xls applied=%d upserts=%d", resXLS.Applied, len(invSvc.upserts))
	}

	// 3. Password-protected BIFF8 .xls -> HTTP 422.
	encXLS := buildSupplierBIFF8XLS(true)
	reqEncXLS := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import", bytes.NewReader(encXLS))
	reqEncXLS.Header.Set("X-File-Name", "locked.xls")
	reqEncXLS = reqEncXLS.WithContext(auth.WithClaims(reqEncXLS.Context(), auth.Claims{SupplierID: "sup-excel-1", Role: auth.RoleAdmin}))
	recEncXLS := httptest.NewRecorder()
	svc.HandleInventoryImport(recEncXLS, reqEncXLS)
	if recEncXLS.Code != http.StatusUnprocessableEntity {
		t.Fatalf("encrypted xls status=%d want=422 body=%s", recEncXLS.Code, recEncXLS.Body.String())
	}

	// 4. Password-protected OOXML .xlsx (EncryptedPackage) -> HTTP 422.
	encXLSX := buildEncryptedOLE2OOXMLContainer()
	reqEncXLSX := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import", bytes.NewReader(encXLSX))
	reqEncXLSX.Header.Set("X-File-Name", "locked.xlsx")
	reqEncXLSX = reqEncXLSX.WithContext(auth.WithClaims(reqEncXLSX.Context(), auth.Claims{SupplierID: "sup-excel-1", Role: auth.RoleAdmin}))
	recEncXLSX := httptest.NewRecorder()
	svc.HandleInventoryImport(recEncXLSX, reqEncXLSX)
	if recEncXLSX.Code != http.StatusUnprocessableEntity {
		t.Fatalf("encrypted xlsx status=%d want=422 body=%s", recEncXLSX.Code, recEncXLSX.Body.String())
	}

	// 5. Spoofed PDF bytes masquerading as .xlsx -> HTTP 400.
	reqPDF := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/import", strings.NewReader("%PDF-1.7\n1 0 obj\n"))
	reqPDF.Header.Set("X-File-Name", "spoofed.xlsx")
	reqPDF = reqPDF.WithContext(auth.WithClaims(reqPDF.Context(), auth.Claims{SupplierID: "sup-excel-1", Role: auth.RoleAdmin}))
	recPDF := httptest.NewRecorder()
	svc.HandleInventoryImport(recPDF, reqPDF)
	if recPDF.Code != http.StatusBadRequest {
		t.Fatalf("spoofed pdf status=%d want=400 body=%s", recPDF.Code, recPDF.Body.String())
	}

	// 6. Verify discoverImportArtifact also surfaces ErrPasswordProtectedWorkbook directly.
	if _, _, _, err := discoverImportArtifact("locked.xlsx", encXLSX, 0); !errors.Is(err, parser.ErrPasswordProtectedWorkbook) {
		t.Fatalf("expected ErrPasswordProtectedWorkbook, got %v", err)
	}
}

func TestLiveSpanner_IngestImportArtifact_XLSX_10500Rows_And_Encrypted422(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner Excel import test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := spanner.NewClient(ctx, "projects/pegasusx-local/instances/pegasusx-instance/databases/pegasusx-db",
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("open spanner client: %v", err)
	}
	defer client.Close()

	if _, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "SupplierImportSessions", spanner.Key{"__probe__", "__probe__"}, []string{"status"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		t.Fatalf("SupplierImportSessions unavailable: %v", probeErr)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	supID := "sup-xlsx-live-" + suffix
	whID := "wh-xlsx-live-" + suffix
	sessEncID := "sess-enc-" + suffix
	sessBigID := "sess-big-" + suffix

	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId": whID, "SupplierId": supID, "Name": "Excel Live WH",
			"CoverageRadiusKm": float64(25), "IsActive": true, "IsOnShift": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed warehouse: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("SupplierImportStagedRows", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierImportMapping", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierImportSessions", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Warehouses", spanner.Key{whID}),
		})
	})

	repo := NewImportRepository(client)
	if _, err := repo.CreateImportSession(ctx, supID, sessEncID, "locked.xlsx", "INITIALIZED"); err != nil {
		t.Fatalf("CreateImportSession enc: %v", err)
	}
	if _, err := repo.CreateImportSession(ctx, supID, sessBigID, "bulk_10500.xlsx", "INITIALIZED"); err != nil {
		t.Fatalf("CreateImportSession big: %v", err)
	}

	svc := NewService(ServiceConfig{
		Repo: &recordingInventoryRepo{
			topology: SupplierTopology{Warehouses: []WarehouseNode{{WarehouseID: whID}}},
		},
		SupplierID: supID,
	})

	ingestHandler := handlePostImportIngest(repo, svc)

	// 1. Encrypted workbook returns HTTP 422.
	reqEnc := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/"+sessEncID+"/ingest", bytes.NewReader(buildEncryptedOLE2OOXMLContainer()))
	reqEnc = reqEnc.WithContext(auth.WithClaims(reqEnc.Context(), auth.Claims{SupplierID: supID, Role: auth.RoleAdmin}))
	rctxEnc := chi.NewRouteContext()
	rctxEnc.URLParams.Add("id", sessEncID)
	reqEnc = reqEnc.WithContext(context.WithValue(reqEnc.Context(), chi.RouteCtxKey, rctxEnc))
	recEnc := httptest.NewRecorder()
	ingestHandler(recEnc, reqEnc)
	if recEnc.Code != http.StatusUnprocessableEntity {
		t.Fatalf("live encrypted ingest status=%d want=422 body=%s", recEnc.Code, recEnc.Body.String())
	}

	// 2. Real 10,500-row multi-sheet .xlsx workbook stages all 10,500 rows into Spanner.
	xlsx10500 := buildMultiSheetXLSX10500(t, whID)
	reqBig := httptest.NewRequest(http.MethodPost, "/v1/supplier/inventory/imports/"+sessBigID+"/ingest", bytes.NewReader(xlsx10500))
	reqBig = reqBig.WithContext(auth.WithClaims(reqBig.Context(), auth.Claims{SupplierID: supID, Role: auth.RoleAdmin}))
	rctxBig := chi.NewRouteContext()
	rctxBig.URLParams.Add("id", sessBigID)
	reqBig = reqBig.WithContext(context.WithValue(reqBig.Context(), chi.RouteCtxKey, rctxBig))
	recBig := httptest.NewRecorder()
	ingestHandler(recBig, reqBig)
	if recBig.Code != http.StatusOK {
		t.Fatalf("live 10500 ingest status=%d want=200 body=%s", recBig.Code, recBig.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(recBig.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode live ingest response: %v", err)
	}
	if int(payload["rows_staged"].(float64)) != 10500 || int(payload["valid_rows"].(float64)) != 10500 {
		t.Fatalf("unexpected staged summary: %+v", payload)
	}
	sess, err := repo.GetSession(ctx, supID, sessBigID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.Status != "DISCOVERED" {
		t.Fatalf("session status=%q want=DISCOVERED", sess.Status)
	}
}

func TestInventoryImport_RejectsAuditorAndAgent403_And_ExpiredAuditor401(t *testing.T) {
	svc := NewService(ServiceConfig{
		Repo:       &recordingInventoryRepo{},
		SupplierID: "sup-rbac-1",
	})
	repo := &ImportRepository{}

	auditorClaims := auth.Claims{
		Subject:       "aud-1",
		Role:          auth.RoleAdmin,
		SupplierID:    "sup-rbac-1",
		SupplierRole:  auth.Role(auth.SupplierRoleAuditor),
		ActorType:     auth.ActorTypeHuman,
		SupplierPerms: []string{auth.PermAuditRead},
	}
	agentClaims := auth.Claims{
		Subject:      "agent-1",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-rbac-1",
		SupplierRole: auth.Role(auth.SupplierRoleAgent),
		ActorType:    auth.ActorTypeAgent,
	}
	past := time.Now().UTC().Add(-2 * time.Minute)
	expiredAudClaims := auth.Claims{
		Subject:          "aud-exp",
		Role:             auth.RoleAdmin,
		SupplierID:       "sup-rbac-1",
		SupplierRole:     auth.Role(auth.SupplierRoleAuditor),
		ActorType:        auth.ActorTypeHuman,
		SupplierPerms:    []string{auth.PermAuditRead},
		AuditorExpiresAt: &past,
	}

	mutHandlers := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{name: "legacy_inventory_import", path: "/v1/supplier/inventory/import", handler: svc.HandleInventoryImport},
		{name: "create_import_session", path: "/v1/supplier/inventory/imports", handler: handleCreateImportSession(repo, svc)},
		{name: "post_import_uploaded", path: "/v1/supplier/inventory/imports/s1/uploaded", handler: handlePostImportUploaded(repo, svc)},
		{name: "post_import_ingest", path: "/v1/supplier/inventory/imports/s1/ingest", handler: handlePostImportIngest(repo, svc)},
		{name: "post_import_mapping", path: "/v1/supplier/inventory/imports/s1/mapping", handler: handlePostImportMapping(repo)},
		{name: "post_import_approve", path: "/v1/supplier/inventory/imports/s1/approve", handler: handlePostImportApprove(repo, svc)},
		{name: "post_import_apply", path: "/v1/supplier/inventory/imports/s1/apply", handler: handlePostImportApply(repo, svc, nil, nil)},
	}

	for _, tc := range mutHandlers {
		for _, actor := range []struct {
			label  string
			claims auth.Claims
			want   int
		}{
			{label: "AUDITOR", claims: auditorClaims, want: http.StatusForbidden},
			{label: "AGENT", claims: agentClaims, want: http.StatusForbidden},
			{label: "EXPIRED_AUDITOR", claims: expiredAudClaims, want: http.StatusUnauthorized},
		} {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{}`))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", "s1")
			req = req.WithContext(context.WithValue(auth.WithClaims(req.Context(), actor.claims), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			tc.handler(rec, req)
			if rec.Code != actor.want {
				t.Fatalf("%s %s status=%d want=%d body=%s", actor.label, tc.name, rec.Code, actor.want, rec.Body.String())
			}
		}
	}

	for _, rtc := range []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{name: "get_import_session", path: "/v1/supplier/inventory/imports/s1", handler: handleGetImportSession(repo)},
		{name: "get_import_rows", path: "/v1/supplier/inventory/imports/s1/rows", handler: handleGetImportRows(repo)},
		{name: "get_import_mapping", path: "/v1/supplier/inventory/imports/s1/mapping", handler: handleGetImportMapping(repo)},
	} {
		req := httptest.NewRequest(http.MethodGet, rtc.path, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "s1")
		req = req.WithContext(context.WithValue(auth.WithClaims(req.Context(), expiredAudClaims), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		rtc.handler(rec, req)
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "auditor_access_expired") {
			t.Fatalf("EXPIRED_AUDITOR %s status=%d want=401 auditor_access_expired body=%s", rtc.name, rec.Code, rec.Body.String())
		}
	}
}
