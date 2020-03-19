// +build tools

// Place any tool dependencies as imports in this file.
// Go modules will be forced to download and install them.

package tools

import (
	// Used by make if correct version not on PATH
	_ "github.com/operator-framework/operator-sdk/cmd/operator-sdk"
)
