package main

import (
	"context"
	"log"
	
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
)

func main() {
	ctx := context.Background()
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil { log.Fatal(err) }
	
	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database: "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db",
		Statements: []string{
			"ALTER TABLE Orders ADD COLUMN VolumeVU FLOAT64",
		},
	})
	if err != nil { log.Fatal(err) }
	if err := op.Wait(ctx); err != nil { log.Fatal(err) }
	log.Println("Migration successful")
}
