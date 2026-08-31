package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dpolarov/MobaWinSCP/internal/app"
	"github.com/dpolarov/MobaWinSCP/internal/hotkey"
)

var version = "dev"

func main() {
	var iniPath string
	var winscpPath string
	var dryRun bool
	var showVersion bool
	var listen bool
	var hotkeyName string
	var verbose bool

	flag.StringVar(&iniPath, "ini", "", "path to MobaXterm.ini (auto-detected when empty)")
	flag.StringVar(&winscpPath, "winscp", "", "path to WinSCP.exe (auto-detected when empty)")
	flag.BoolVar(&dryRun, "dry-run", false, "print detected session and WinSCP command without launching")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&listen, "listen", false, "stay resident and open WinSCP from the global hotkey")
	flag.StringVar(&hotkeyName, "hotkey", "Ctrl+Alt+W", "global hotkey used with -listen")
	flag.BoolVar(&verbose, "verbose", false, "print hotkey detection diagnostics")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	opts := app.Options{INIPath: iniPath, WinSCPPath: winscpPath, DryRun: dryRun}
	if listen {
		err := hotkey.Run(hotkey.Options{
			Hotkey: hotkeyName,
			Verbose: verbose,
			OnSession: func(s session.Session) error {
				return app.RunSession(opts, s)
			},
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "MobaWinSCP:", err)
			os.Exit(1)
		}
		return
	}

	if err := app.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "MobaWinSCP:", err)
		os.Exit(1)
	}
}
