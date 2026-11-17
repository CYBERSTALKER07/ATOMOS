package onec

import (
	"fmt"
	"testing"
)

func TestExtractCommerceMLBundle_MultiEntity(t *testing.T) {
	xmlDoc := `<?xml version="1.0" encoding="UTF-8"?>
<КоммерческаяИнформация ВерсияСхемы="2.10">
  <Каталог><Товары><Товар><Ид>p1</Ид><Артикул>SKU-10</Артикул><Наименование>Tea 500g</Наименование><Штрихкод>478001</Штрихкод><БазоваяЕдиница>PCS</БазоваяЕдиница></Товар></Товары></Каталог>
  <ПакетПредложений><Предложения><Предложение><Ид>p1</Ид><Артикул>SKU-10</Артикул><Цены><Цена><ЦенаЗаЕдиницу>24500</ЦенаЗаЕдиницу><Валюта>UZS</Валюта></Цена></Цены><Склад ИдСклада="WH-1" КоличествоНаСкладе="80"/></Предложение></Предложения></ПакетПредложений>
  <Контрагенты><Контрагент><Наименование>LLC Baraka</Наименование><ИНН>301234567</ИНН></Контрагент></Контрагенты>
  <Документ><Номер>DOC-42</Номер><Сумма>1250000</Сумма></Документ>
</КоммерческаяИнформация>`

	bundle, err := ExtractCommerceMLBundle([]byte(xmlDoc))
	if err != nil {
		t.Fatalf("ExtractCommerceMLBundle: %v", err)
	}
	if len(bundle.Streams) != 4 || len(bundle.Batch.Products) != 1 {
		t.Fatalf("unexpected bundle counts: streams=%d products=%d", len(bundle.Streams), len(bundle.Batch.Products))
	}
	p := bundle.Batch.Products[0]
	if p.ExternalID != "p1" || p.PriceMinor != 2450000 || p.Currency != "UZS" || bundle.StockByExternalID["p1"] != 80 {
		t.Fatalf("unexpected product merge: %+v stock=%v", p, bundle.StockByExternalID["p1"])
	}
	if len(bundle.Customers) != 1 || fmt.Sprint(bundle.Customers[0]["inn"]) != "301234567" {
		t.Fatalf("expected customer INN, got %+v", bundle.Customers)
	}
	if len(bundle.Documents) != 1 || fmt.Sprint(bundle.Documents[0]["external_order_no"]) != "DOC-42" {
		t.Fatalf("expected document row, got %+v", bundle.Documents)
	}
}
