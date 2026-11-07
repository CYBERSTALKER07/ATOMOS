package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

func waitForWSMessage(ctx context.Context, conn *websocket.Conn, want ...string) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return err
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					continue
				}
			}
			return err
		}
		text := string(msg)
		matched := 0
		for _, needle := range want {
			if strings.Contains(text, needle) {
				matched++
			}
		}
		if matched == len(want) {
			return nil
		}
	}
	return fmt.Errorf("ws message not received within window; want %v", want)
}

// runCrossRoleSupplierBroadcastWS verifies supplier-scoped WS fan-out (ops broadcast path).
func runCrossRoleSupplierBroadcastWS(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config, supplierID string) error {
	adminToken, err := issueRoleJWT(cfg, auth.Claims{
		Subject:    "ssmr-supplier-admin",
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	})
	if err != nil {
		return fmt.Errorf("issue supplier admin jwt: %w", err)
	}
	conn, _, err := dialWS(ctx, base, adminToken)
	if err != nil {
		return fmt.Errorf("supplier ws dial: %w", err)
	}
	defer conn.Close()

	probe := fmt.Sprintf("SSMR_WS_PROBE_%d", time.Now().UnixNano())
	broadcastPayload, _ := json.Marshal(map[string]string{
		"title": probe,
		"body":  "cross-role realtime probe",
		"role":  "ALL",
	})
	go func() {
		key := fmt.Sprintf("ssmr-ws-broadcast-probe-%d", time.Now().UnixNano())
		status, body, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/broadcast", broadcastPayload, cookie, key)
		if err != nil || status < 200 || status >= 300 {
			fmt.Printf("PX_E2E_WS_BROADCAST_POST_FAIL status=%d err=%v body=%s\n", status, err, string(body))
		}
	}()
	if err := waitForWSMessage(ctx, conn, "SUPPLIER_BROADCAST", probe); err != nil {
		return fmt.Errorf("supplier broadcast ws: %w", err)
	}
	fmt.Println("PX_E2E_CROSS_ROLE_WS_OK")
	fmt.Println("PX_E2E_DESKTOP_WS_OK")
	return nil
}

// runWarehouseDispatchExecuteWithWS runs dispatch execute while a warehouse admin WS client listens for DISPATCH_COMMITTED.
