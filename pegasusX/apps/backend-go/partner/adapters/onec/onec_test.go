package onec

import "testing"

func TestParseCommerceMLCatalog(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Каталог>
  <Товары>
    <Товар>
      <Ид>SKU-1</Ид>
      <Наименование>Tea</Наименование>
      <Штрихкод>4600000000001</Штрихкод>
      <Цена>15000</Цена>
      <Валюта>UZS</Валюта>
    </Товар>
  </Товары>
</Каталог>`)
	batch, err := ParseCommerceMLCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Products) != 1 || batch.Products[0].ExternalID != "SKU-1" {
		t.Fatalf("%+v", batch.Products)
	}
	if batch.Products[0].Barcode != "4600000000001" {
		t.Fatalf("barcode=%s", batch.Products[0].Barcode)
	}
}
