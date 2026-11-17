# Golden Migration Fixtures (M11.1)

All files in this directory contain strictly anonymized, synthetic Uzbekistan test data (zero real PII).

1. `suite1_all_in_one_25k.xlsx` — Multi-sheet 25,505-row Excel workbook (`Склады`, `Номенклатура_Цены`, `Остатки`, `Контрагенты`, `Дебиторка_Насия`, `Заказы_История`).
2. `suite2_biff8_legacy.xls` — Legacy OLE2 BIFF8 XLS workbook with Cyrillic `cp1251`/UTF-16LE sheet and headers.
3. `suite2_messy_cp1251_billz.csv` — Semicolon-delimited Windows-1251 Billz/Smartup-style CSV with a 10x price jump anomaly.
4. `suite3_commerceml_1c.xml` — 1C CommerceML 2.10 XML bundle covering goods, offers + warehouse rests, counterparties, completed sales, and open buyer orders.
5. `suite4_composite_coldchain.json` — Single composite flat JSON array combining warehouses, products, cold-chain `StockLots`, formal/informal customers, and open orders for FEFO reservation + cancellation testing.
6. `suite5_financial_and_inn_invariants.json` — Financial (`nasiya` imbalance / currency mismatch), duplicate `OrderId`, INN lookup states, and 72h rollback reference scenarios.

Regenerate cleanly via:
```bash
cd apps/backend-go && go run ./cmd/gen-migration-fixtures -out ../../testdata/migrations
```
