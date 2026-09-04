-- Theatre #8: persist seasonal override multipliers (kill hardcoded 1.2 at read time).

ALTER TABLE SeasonalTemplateOverrides ADD COLUMN Multiplier FLOAT64;
