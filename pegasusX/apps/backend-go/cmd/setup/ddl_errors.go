package main

import "github.com/pegasusx/pegasusx/apps/backend-go/schemadrift"

func isBenignDDLConflict(err error) bool {
	return schemadrift.IsBenignDDLConflict(err)
}
