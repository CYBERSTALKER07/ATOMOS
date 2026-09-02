package main

import (
	"fmt"
	"cloud.google.com/go/spanner"
)

func main() {
	var j spanner.NullJSON
	fmt.Printf("%T\n", j.Value)
}
