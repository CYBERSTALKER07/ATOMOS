package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrWrongCell is returned when a validly signed JWT belongs to another home cell.
// SessionAuth / ParseBearerClaims treat this as unauthenticated (401).
var ErrWrongCell = errors.New("wrong cell")

func cellJWTEnforce() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CELL_JWT_ENFORCE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "production")
}

// rejectForeignCell is GS-C4: when enforcement is on and HOME_CELL is set
// (EU overlay sets cell-eu), a UZ token is 401 even if the JWT secret was copied.
func rejectForeignCell(c Claims) error {
	if !cellJWTEnforce() {
		return nil
	}
	want := strings.ToLower(strings.TrimSpace(os.Getenv("HOME_CELL")))
	if want == "" {
		return nil
	}
	got := strings.ToLower(strings.TrimSpace(c.HomeCell))
	if got == "" || got == want {
		return nil
	}
	return fmt.Errorf("jwt: %w (%s != %s)", ErrWrongCell, got, want)
}
