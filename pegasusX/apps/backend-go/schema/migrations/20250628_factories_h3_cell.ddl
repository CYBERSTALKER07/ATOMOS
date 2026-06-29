ALTER TABLE Factories ADD COLUMN H3Cell STRING(15);
CREATE INDEX Idx_Factories_ByH3Cell ON Factories(SupplierId, H3Cell);
