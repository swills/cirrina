//go:build tools
// +build tools

package tools

import (
	_ "go.uber.org/mock/mockgen"
	_ "github.com/a-h/templ/cmd/templ"
)
