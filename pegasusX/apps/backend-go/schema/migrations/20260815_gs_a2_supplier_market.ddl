-- GS-A2: persist market pack + home cell on the tenant row.
-- Nullable on purpose: empty ≠ silent “user chose UZ”. Session source is claim|profile|env.

ALTER TABLE Suppliers ADD COLUMN MarketCode STRING(8);
ALTER TABLE Suppliers ADD COLUMN HomeCell STRING(32);
