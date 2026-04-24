#\!/bin/bash
cd the-lab-monorepo/apps/backend-go

# In webhookroutes/routes.go: remove Click/Payme completely
sed -i '' '/click/d' webhookroutes/routes.go
sed -i '' '/payme/d' webhookroutes/routes.go

# In payment/gateway_client.go: remove CLICK and PAYME cases
sed -i '' '/case "PAYME":/,/case "GLOBAL_PAY":/{ /case "GLOBAL_PAY":/\!d; }' payment/gateway_client.go
sed -i '' '/case "CLICK":/,/case "GLOBAL_PAY":/{ /case "GLOBAL_PAY":/\!d; }' payment/gateway_client.go

# Now, wait, the simplest way is to manually remove the `HandleClickWebhook` and `HandlePaymeWebhook` 
# from webhooks.go by matching line numbers inside python or `sed`\!

