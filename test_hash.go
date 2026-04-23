package main
import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)
func main() {
	hash := "$2a$10$GZaXLJ15MwgE7QhH6b5SguoD0oxqmn/lLytHULabJOaxGvOt//H9q"
	passwords := []string{"password", "password123", "123456", "12345678", "admin"}
	for _, p := range passwords {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p))
		if err == nil {
			fmt.Println("MATCH FOUND:", p)
			return
		}
	}
	fmt.Println("No match found")
}
