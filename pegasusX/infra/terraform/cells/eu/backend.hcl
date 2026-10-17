# GS-C2 — EU cell state prefix.
# MUST NOT be "pegasusx/ssmr" — that prefix is the live UZ/SSMR state.
# C3 creates project pegasusx-cell-eu; do not terraform init this backend
# against the live bucket from the catalog (would write a new state object).
prefix = "pegasusx/cell-eu"
