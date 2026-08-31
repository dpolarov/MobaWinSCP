package app

import (
	"fmt"
	"strings"

	"github.com/dpolarov/MobaWinSCP/internal/mobaxterm"
	"github.com/dpolarov/MobaWinSCP/internal/session"
	"github.com/dpolarov/MobaWinSCP/internal/winscp"
)

type Options struct {
	INIPath    string
	WinSCPPath string
	DryRun     bool
}

func Run(o Options) error {
	live, err := session.Detect()
	if err != nil { return err }
	return RunSession(o, live)
}

func RunSession(o Options, live session.Session) error {
	ini, err := mobaxterm.LocateINI(o.INIPath)
	if err != nil { return err }
	bookmarks, err := mobaxterm.ParseFile(ini)
	if err != nil { return fmt.Errorf("parse %s: %w", ini, err) }
	bookmark, err := mobaxterm.Find(bookmarks, live.Host, live.Port, live.User)
	if err != nil { return err }
	exe, err := winscp.Locate(o.WinSCPPath)
	if err != nil { return err }
	key := mobaxterm.ResolvePortablePath(bookmark.PrivateKey, ini)
	launch := winscp.LaunchOptions{Executable: exe, Host: bookmark.Host, Port: bookmark.Port, User: bookmark.User, RemotePath: live.RemotePath, PrivateKey: key}
	args := winscp.Args(launch)
	if o.DryRun {
		fmt.Printf("Session : %s/%s -> %s@%s:%d\n", bookmark.Folder, bookmark.Name, bookmark.User, bookmark.Host, bookmark.Port)
		fmt.Printf("Remote  : %s\n", live.RemotePath)
		fmt.Printf("INI     : %s\n", ini)
		fmt.Printf("Key     : %s\n", printableKey(key))
		fmt.Printf("WinSCP  : %s %s\n", exe, quoteArgs(args))
		return nil
	}
	if err := winscp.Launch(launch); err != nil { return fmt.Errorf("launch WinSCP: %w", err) }
	return nil
}

func printableKey(key string) string { if key == "" { return "<none>" }; return key }
func quoteArgs(args []string) string {
	out := make([]string, len(args)); for i, a := range args { if strings.ContainsAny(a, " \t") { out[i] = `"` + a + `"` } else { out[i] = a } }; return strings.Join(out, " ")
}
