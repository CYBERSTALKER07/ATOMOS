-- L3.3: expose sell-through merge metadata on reorder suggestions.
ALTER TABLE ReorderSuggestions ADD COLUMN SourcesJson STRING(MAX);
ALTER TABLE ReorderSuggestions ADD COLUMN SellThroughVel FLOAT64;
ALTER TABLE ReorderSuggestions ADD COLUMN BaseDemand FLOAT64;
