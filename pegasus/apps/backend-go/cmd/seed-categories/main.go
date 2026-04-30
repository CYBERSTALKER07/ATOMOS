package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func main() {
	os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	db := "projects/the-lab-industries/instances/dev-instance/databases/the-lab-db"
	client, err := spanner.NewClient(context.Background(), db)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	cats := []string{
		"Beverages",
		"Dairy & Eggs",
		"Snacks & Confectionery",
		"Bakery & Bread",
		"Frozen Foods",
		"Meat & Poultry",
		"Fresh Produce",
		"Grocery & Staples",
		"Household & Cleaning",
		"Personal Care",
		"Tobacco & Accessories",
		"Water & Juices",
	}

	var ms []*spanner.Mutation
	for i, c := range cats {
		ms = append(ms, spanner.InsertMap("PlatformCategories", map[string]interface{}{
			"CategoryId":   uuid.New().String(),
			"DisplayName":  c,
			"DisplayOrder": int64(i + 1),
		}))
	}

	if _, err := client.Apply(context.Background(), ms); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Seeded %d platform categories.\n", len(cats))
}
