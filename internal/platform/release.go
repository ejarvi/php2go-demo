// Package platform holds what every page shares: configuration, security headers, the
// CloudFront origin check and the release ID.
package platform

import "os"

// ReleaseID is set at build time by the deploy with -ldflags "-X …ReleaseID=r12+abc1234" (D-016).
var ReleaseID = ""

// Release is the running release's ID: the one built in, else the RELEASE environment
// variable (used by the tests), else "dev".
func Release() string {
	if ReleaseID != "" {
		return ReleaseID
	}
	if r := os.Getenv("RELEASE"); r != "" {
		return r
	}
	return "dev"
}
