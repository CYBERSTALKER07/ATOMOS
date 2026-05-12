import os

api_path = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/data/api/PegasusApi.kt"
with open(api_path, "r") as f:
    api_text = f.read()

# Make sure we have the proper CartSyncRequest and response imports if needed, but we'll use JsonElement or create small data classes
# Let's insert the missing WebSocket events

ws_path = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/data/api/RetailerWebSocket.kt"
with open(ws_path, "r") as f:
    ws_text = f.read()

if "SHOP_CLOSED_NOTIFICATION" not in ws_text:
    old_ws = """                "PRE_ORDER_EDITED" -> {
                    val order = json.decodeFromJsonElement<Order>(data)
                    _events.emit(WsEvent.PreOrderEdited(order))
                }
                else -> {
                    Log.w("RetailerWebSocket", "Unhandled event type: $type")
                }"""
    new_ws = """                "PRE_ORDER_EDITED" -> {
                    val order = json.decodeFromJsonElement<Order>(data)
                    _events.emit(WsEvent.PreOrderEdited(order))
                }
                "SHOP_CLOSED_NOTIFICATION" -> {
                    // Alert the retailer that the shop is closed
                    val orderId = data.jsonObject["order_id"]?.jsonPrimitive?.content ?: ""
                    _events.emit(WsEvent.ShopClosed(orderId))
                }
                "ORDER_REJECTED_BY_SUPPLIER" -> {
                    val orderId = data.jsonObject["order_id"]?.jsonPrimitive?.content ?: ""
                    val reason = data.jsonObject["reason"]?.jsonPrimitive?.content ?: ""
                    _events.emit(WsEvent.OrderRejected(orderId, reason))
                }
                else -> {
                    Log.w("RetailerWebSocket", "Unhandled event type: $type")
                }"""
    ws_text = ws_text.replace(old_ws, new_ws)
    
    # Also update the WsEvent sealed class
    old_wsevent = """    data class PreOrderConfirmed(val order: Order) : WsEvent()
    data class PreOrderEdited(val order: Order) : WsEvent()
}"""
    new_wsevent = """    data class PreOrderConfirmed(val order: Order) : WsEvent()
    data class PreOrderEdited(val order: Order) : WsEvent()
    data class ShopClosed(val orderId: String) : WsEvent()
    data class OrderRejected(val orderId: String, val reason: String) : WsEvent()
}"""
    ws_text = ws_text.replace(old_wsevent, new_wsevent)

with open(ws_path, "w") as f:
    f.write(ws_text)

print("Patched WebSocket.")
