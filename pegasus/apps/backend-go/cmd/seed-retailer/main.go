package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/spanner"
)

func main() {
	os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	dbString := "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db"
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, dbString)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdate("Retailers",
			[]string{"RetailerId", "Name", "TaxIdentificationNumber", "Status", "CreatedAt"},
			[]interface{}{"RET-9021", "Test Shop", "123456789", "VERIFIED", spanner.CommitTimestamp},
		),
	})
	if err != nil {
		log.Fatal("Failed to seed: ", err)
	}
	fmt.Println("Seeded RET-9021 with VERIFIED status")
}
