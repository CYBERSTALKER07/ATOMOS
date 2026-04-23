package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func InjectAIAndOrders(ctx context.Context, client *spanner.Client, retailers []SimRetailer, supplierId string, rng *rand.Rand) {
	// We'll use 2 specific retailers from our generated list to demonstrate AI cases
	if len(retailers) < 3 {
		return
	}

	ret1 := retailers[0] // Case 1: History, Auto-Order OFF
	ret2 := retailers[1] // Case 2: Active Preorders, Auto-Order ON

	now := time.Now()

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{}

		// ---------------------------------------------------------
		// RETAILER 1: History (Auto-Order OFF)
		// We insert 10 COMPLETED orders in the past 30 days
		// ---------------------------------------------------------
		for i := 0; i < 10; i++ {
			orderId := fmt.Sprintf("ORD-SIM-HIST-%03d", i)
			daysAgo := rng.Intn(30) + 1
			createdAt := now.AddDate(0, 0, -daysAgo)

			amount := int64(100000 + rng.Intn(400000))

			mutations = append(mutations, spanner.Insert("Orders",
				[]string{"OrderId", "RetailerId", "SupplierId", "DriverId", "State",
					"Amount", "PaymentStatus", "RouteId", "OrderSource", "DeliveryToken", "CreatedAt"},
				[]interface{}{orderId, ret1.ID, supplierId, nilStr(""), "COMPLETED",
					amount, "PAID", nilStr(""), "RETAILER_APP", fmt.Sprintf("VOID-ORD-%s", uuid.New().String()), createdAt}))

			mutations = append(mutations, spanner.Insert("OrderLineItems",
				[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Status"},
				[]interface{}{fmt.Sprintf("LI-%s-1", orderId), orderId, "SKU-SIM-001", int64(2), int64(5000), "COMPLETED"}))
		}

		// Ensure Retailer 1 Global Settings is OFF
		mutations = append(mutations, spanner.Update("RetailerGlobalSettings",
			[]string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"},
			[]interface{}{ret1.ID, false, spanner.CommitTimestamp}))

		// ---------------------------------------------------------
		// RETAILER 2: Active Preorders (Auto-Order ON)
		// ---------------------------------------------------------
		mutations = append(mutations, spanner.Update("RetailerGlobalSettings",
			[]string{"RetailerId", "GlobalAutoOrderEnabled", "UpdatedAt"},
			[]interface{}{ret2.ID, true, spanner.CommitTimestamp}))

		// Inject an AI Prediction that is WAITING
		predId := "PRED-SIM-001"
		triggerDate := now.Add(24 * time.Hour) // Trigger tomorrow
		mutations = append(mutations, spanner.Insert("AIPredictions",
			[]string{"PredictionId", "RetailerId", "Status", "TriggerDate", "CreatedAt", "UpdatedAt"},
			[]interface{}{predId, ret2.ID, "WAITING", triggerDate, spanner.CommitTimestamp, spanner.CommitTimestamp}))

		mutations = append(mutations, spanner.Insert("AIPredictionItems",
			[]string{"PredictionItemId", "PredictionId", "SkuId", "ExpectedQuantity", "UnitPrice", "ConfidenceScore"},
			[]interface{}{"PITEM-SIM-001", predId, "SKU-SIM-001", int64(5), int64(5000), 0.92}))

		// ---------------------------------------------------------
		// ALL RETAILERS: Inject 1 PENDING order for today's routing
		// ---------------------------------------------------------
		for i, r := range retailers {
			// Skip injecting pending order for every single one if we want some to be quiet,
			// but prompt asks to simulate 100 orders from 100 retailers.
			orderId := fmt.Sprintf("ORD-SIM-TODAY-%03d", i)
			amount := int64(50000 + rng.Intn(200000))

			// Deterministic delivery token for scanning
			deliveryToken := fmt.Sprintf("VOID-ORD-%s", r.ID)

			mutations = append(mutations, spanner.Insert("Orders",
				[]string{"OrderId", "RetailerId", "SupplierId", "DriverId", "State",
					"Amount", "PaymentStatus", "RouteId", "OrderSource", "DeliveryToken", "CreatedAt"},
				[]interface{}{orderId, r.ID, supplierId, nilStr(""), "PENDING",
					amount, "PENDING", nilStr(""), "RETAILER_APP", deliveryToken, spanner.CommitTimestamp}))

			mutations = append(mutations, spanner.Insert("OrderLineItems",
				[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Status"},
				[]interface{}{fmt.Sprintf("LI-%s", orderId), orderId, "SKU-SIM-001", int64(1), int64(5000), "PENDING"}))
		}

		return txn.BufferWrite(mutations)
	})

	if err != nil {
		log.Fatalf("Failed injecting AI states and Orders: %v", err)
	}

	log.Printf("[SIMULATE] Simulated Historical AI data for Retailer %s and WAITING prediction for %s.\n", ret1.ID, ret2.ID)
	log.Printf("[SIMULATE] Created 1 PENDING order for each generated Retailer.\n")
}
