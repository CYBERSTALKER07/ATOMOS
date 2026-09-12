-- Gate 5 / §8.10 Phase 3: GlobalProducts master + offers + match queue + UoM hierarchy.
-- Operational SKUs remain on Products; this is a link layer.

CREATE TABLE UnitsOfMeasure (
  UomId          STRING(36)   NOT NULL,
  Code           STRING(16)   NOT NULL,  -- EACH | INNER | CASE | PALLET
  Name           STRING(64)   NOT NULL,
  FactorToBase   INT64        NOT NULL,  -- units of EACH per this UoM
  ParentUomId    STRING(36),
  CreatedAt      TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (UomId);

CREATE UNIQUE INDEX Idx_UnitsOfMeasure_ByCode ON UnitsOfMeasure(Code);

CREATE TABLE GlobalProducts (
  GlobalProductId  STRING(36)   NOT NULL,
  Gtin             STRING(14),             -- normalized GTIN when known
  Brand            STRING(128),
  Manufacturer     STRING(128),
  Name             STRING(255)  NOT NULL,
  PackQty          INT64        NOT NULL DEFAULT (1),
  BaseUomId        STRING(36)   NOT NULL,
  NormalizedKey    STRING(512),            -- brand|name|pack|uom for fuzzy
  Version          INT64        NOT NULL DEFAULT (1),
  CreatedAt        TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (GlobalProductId);

CREATE UNIQUE NULL_FILTERED INDEX Idx_GlobalProducts_ByGtin ON GlobalProducts(Gtin);
CREATE INDEX Idx_GlobalProducts_ByNormalizedKey ON GlobalProducts(NormalizedKey);

CREATE TABLE SupplierProductOffers (
  SupplierId       STRING(36)   NOT NULL,
  ProductId        STRING(36)   NOT NULL,
  GlobalProductId  STRING(36)   NOT NULL,
  PriceMinor       INT64        NOT NULL,
  Currency         STRING(3)    NOT NULL,
  Moq              INT64        NOT NULL DEFAULT (1),
  LeadTimeDays     INT64        NOT NULL DEFAULT (0),
  Status           STRING(16)   NOT NULL,  -- LINKED | PENDING | REJECTED
  Version          INT64        NOT NULL DEFAULT (1),
  CreatedAt        TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt        TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId, ProductId);

CREATE INDEX Idx_Offers_ByGlobalProduct ON SupplierProductOffers(GlobalProductId, Status);
CREATE INDEX Idx_Offers_ByProduct ON SupplierProductOffers(ProductId);

CREATE TABLE ProductMatchQueue (
  QueueId                   STRING(36)   NOT NULL,
  SupplierId                STRING(36)   NOT NULL,
  ProductId                 STRING(36)   NOT NULL,
  CandidateGlobalProductId  STRING(36),
  MatchMethod               STRING(16)   NOT NULL,  -- EXACT_GTIN | FUZZY | MANUAL
  Score                     FLOAT64      NOT NULL DEFAULT (0),
  Status                    STRING(16)   NOT NULL,  -- PENDING | ACCEPTED | REJECTED
  Reason                    STRING(512),
  Version                   INT64        NOT NULL DEFAULT (1),
  CreatedAt                 TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
  UpdatedAt                 TIMESTAMP    NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (QueueId);

CREATE INDEX Idx_MatchQueue_ByStatusCreated ON ProductMatchQueue(Status, CreatedAt);
CREATE INDEX Idx_MatchQueue_BySupplierProduct ON ProductMatchQueue(SupplierId, ProductId, Status);

-- Seed standard pack hierarchy (idempotent via apply scripts / setup InsertOrUpdate).
-- uom-each / uom-inner / uom-case / uom-pallet with FactorToBase 1 / 6 / 24 / 96.
