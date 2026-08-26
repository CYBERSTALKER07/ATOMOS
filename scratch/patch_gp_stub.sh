#!/bin/bash
sed -i '' '/if e.username == "" || e.password == "" {/,/}, nil/c\
		if e.username == "" || e.password == "" {\
			return ExecutionResult{}, errGlobalPayUnkeyed()\
		}
' pegasusX/apps/backend-go/payment/global_pay_executor.go
