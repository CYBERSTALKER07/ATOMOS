package main

import (
	"context"
	"log"

	"cloud.google.com/go/spanner"
)

func forceUpdateTestRetailer() {
	ctx := context.Background()
	client, err := spanner.NewClient(ctx, "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db")
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: "UPDATE Retailers SET Phone = '+998901234567', PasswordHash = '$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q' WHERE RetailerId = 'retailer-123'",
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		log.Fatalf("failed: %v", err)
	}
	log.Println("Force updated test retailer!")
}
