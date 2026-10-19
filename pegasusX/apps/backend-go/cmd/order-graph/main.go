package main

import (
	"fmt"
	"os"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("PEGASUS CANONICAL ORDER STATE MACHINE GRAPH")
	fmt.Println("================================================================================")

	statuses := order.AllStatuses()
	totalTransitions := 0
	for _, s := range statuses {
		next := order.NextStates(s)
		totalTransitions += len(next)
		terminalFlag := ""
		if order.IsTerminal(s) {
			terminalFlag = " [TERMINAL]"
		}
		fmt.Printf("• %-25s -> %2d exits%s\n", s, len(next), terminalFlag)
		for _, target := range next {
			fmt.Printf("    └──> %s\n", target)
		}
	}

	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Printf("SUMMARY: %d Canonical States, %d Directed Transitions\n", len(statuses), totalTransitions)
	fmt.Println("--------------------------------------------------------------------------------")

	// Validate Invariants
	fmt.Println("INVARIANT AUDIT:")
	// 1. ADR-009 Fiscal Hard Gate
	if !order.CanTransition(order.StatusArrived, order.StatusCompleted) &&
		!order.CanTransition(order.StatusAwaitingPayment, order.StatusCompleted) &&
		!order.CanTransition(order.StatusPendingCashCollection, order.StatusCompleted) &&
		!order.CanTransition(order.StatusDeliveredOnCredit, order.StatusCompleted) {
		fmt.Println("  ✓ ADR-009 Fiscal Hard-Gate: VERIFIED (zero direct soft-completes)")
	} else {
		fmt.Println("  ✗ ADR-009 Fiscal Hard-Gate: VIOLATION DETECTED")
		os.Exit(1)
	}

	// 2. §9.1 Credit Terms Invariant
	if order.CanTransition(order.StatusDeliveredOnCredit, order.StatusFiscalizing) &&
		!order.CanTransition(order.StatusDeliveredOnCredit, order.StatusCompleted) {
		fmt.Println("  ✓ Section 9.1 Credit Terms: VERIFIED (requires money settlement before fiscal)")
	} else {
		fmt.Println("  ✗ Section 9.1 Credit Terms: VIOLATION DETECTED")
		os.Exit(1)
	}

	// 3. Anti-Brick Cancel Handshake Exits
	cancelExits := order.NextStates(order.StatusCancelRequested)
	if len(cancelExits) >= 4 &&
		order.CanTransition(order.StatusCancelRequested, order.StatusCancelled) &&
		order.CanTransition(order.StatusCancelRequested, order.StatusLoaded) &&
		order.CanTransition(order.StatusCancelRequested, order.StatusInTransit) &&
		order.CanTransition(order.StatusCancelRequested, order.StatusArrived) {
		fmt.Println("  ✓ Anti-Brick Cancel Handshake: VERIFIED (4 safe operational return exits)")
	} else {
		fmt.Println("  ✗ Anti-Brick Cancel Handshake: VIOLATION DETECTED")
		os.Exit(1)
	}

	fmt.Println("================================================================================")
	fmt.Println("ALL INVARIANTS PASS CANONICAL VALIDATION")
	fmt.Println("================================================================================")
}
