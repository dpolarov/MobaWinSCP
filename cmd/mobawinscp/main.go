package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dpolarov/MobaWinSCP/internal/app"
)

var version = "dev"

func main() {
	var iniPath string
	var winscpPath string
	var dryRun bool
	var showVersion bool

	flag.StringVar(&iniPath, "ini", "", "path to MobaXterm.ini (auto-detected when empty)")
	flag.StringVar(&winscpPath, "winscp", "", "path to WinSCP.exe (auto-detected when empty)")
	flag.BoolVar(&dryRun, "dry-run", false, "print detected session and WinSCP command without launching")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	if err := app.Run(app.Options{INIPath: iniPath, WinSCPPath: winscpPath, DryRun: dryRun}); err != nil {
		fmt.Fprintln(os.Stderr, "MobaWinSCP:", err)
		os.Exit(1)
	}
}
