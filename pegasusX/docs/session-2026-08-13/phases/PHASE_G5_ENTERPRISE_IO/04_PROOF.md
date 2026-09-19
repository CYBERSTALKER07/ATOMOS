# G5 Proof — Enterprise I/O

## Tests

```text
go test ./partner/ ./partner/edi/ ./partner/adapters/onec/ -count=1
# ok
```

## Evidence

| ID | Evidence |
|----|----------|
| G5-A1 | `edi_profile.go`, profile gate inbound/outbound, GET/PUT `/edi/profile`, migration |
| G5-B1 | `adapters/onec` + `HandleOneCImport`; SAP residual README |
| G5-C1 | `masterdata_g5.go` parties/plants/DLQ |
| G5-D1 | `wms_asn.go` POST `/wms/asn` idempotent |

## Honesty

EDI-lite + AS2 not Drummond-certified. 1C subset not vendor-certified. SAP deferred.
