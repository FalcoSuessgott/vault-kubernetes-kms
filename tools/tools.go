//go:build tools

package tools

//go:generate go install gotest.tools/gotestsum
import (
	// gotestsum
	_ "gotest.tools/gotestsum"
)
