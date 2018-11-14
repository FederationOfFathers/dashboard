package environment

import "os"

const prodServicePath = "/var/lib/dashboard/prod"
const devServicePath = "/var/lib/dashboard/dev"

// IsProd true if production environment service path exists
var IsProd bool

// IsDev true if development environment service path exists
var IsDev bool

// IsLocal true if neither prod or dev service paths exist
var IsLocal bool

//
func init() {
	// check prod status
	_, prodStatErr := os.Stat(prodServicePath)
	IsProd = !os.IsNotExist(prodStatErr)

	// check dev status
	_, devStatErr := os.Stat(devServicePath)
	IsDev = !os.IsNotExist(devStatErr)

	// if neither, then localhost
	IsLocal = !IsProd && !IsDev
}
