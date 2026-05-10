
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// audit_outbox scans Go files for violations of the Transactional Outbox pattern.
// Specifically, it flags:
// 1. Direct Kafka writes (e.g., writer.WriteMessages) outside of the outbox relay.
// 2. spanner.ReadWriteTransaction blocks that perform mutations without using outbox.EmitJSON.

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: audit_outbox <path-to-backend-go>")
		os.Exit(1)
	}
	root := os.Args[1]

	fset := token.NewFileSet()
	var violations []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		node, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return fmt.Errorf("failed to parse %s: %v", path, parseErr)
		}

		ast.Inspect(node, func(n ast.Node) bool {
			// Catch direct writer.WriteMessages calls
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "WriteMessages" {
						// Allow telemetry and outbox itself
						if !strings.Contains(path, "/outbox/") && !strings.Contains(path, "/telemetry/") {
							pos := fset.Position(call.Pos())
							violations = append(violations, fmt.Sprintf("FATAL: Direct Kafka WriteMessages at %s:%d. Must use outbox.EmitJSON.", pos.Filename, pos.Line))
						}
					}
				}
			}

			// Catch ReadWriteTransaction lacking outbox
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "ReadWriteTransaction" { // spannerClient.ReadWriteTransaction
						if hasMutation(call) && !hasOutboxEmit(call) {
							pos := fset.Position(call.Pos())
							violations = append(violations, fmt.Sprintf("FATAL: Spanner ReadWriteTransaction with mutation but NO outbox.EmitJSON at %s:%d", pos.Filename, pos.Line))
						}
					}
				}
			}
			return true
		})
		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning files: %v\n", err)
		os.Exit(1)
	}

	if len(violations) > 0 {
		fmt.Printf("Found %d violations of the Transactional Outbox Doctrine:\n", len(violations))
		for _, v := range violations {
			fmt.Println(v)
		}
		os.Exit(1)
	}

	fmt.Println("SUCCESS: Zero Transactional Outbox violations found.")
}

func hasMutation(node ast.Node) (found bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "BufferWrite" || sel.Sel.Name == "InsertOrUpdateMap" || sel.Sel.Name == "Insert" || sel.Sel.Name == "Update" {
					found = true
				}
			}
		}
		return !found
	})
	return
}

func hasOutboxEmit(node ast.Node) (found bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "EmitJSON" || sel.Sel.Name == "Emit" {
					found = true
				}
			}
		}
		return !found
	})
	return
}
