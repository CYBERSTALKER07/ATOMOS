#!/bin/bash
sed -i '' '/if e.username == "" || e.password == "" {/,/}/c\
		if e.username == "" || e.password == "" {\
			return ExecutionResult{}, fmt.Errorf("globalpay: missing credentials (username/password)")\
		}
' pegasusX/apps/backend-go/payment/global_pay_executor.go
