//go:build !windows

package hotkey

import (
	"fmt"

	"github.com/dpolarov/MobaWinSCP/internal/session"
)

type Options struct {
	Hotkey    string
	Verbose   bool
	OnSession func(session.Session) error
}

func Run(o Options) error {
	return fmt.Errorf("global hotkey mode is supported on Windows only")
}
