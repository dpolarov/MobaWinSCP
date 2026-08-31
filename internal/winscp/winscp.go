package winscp

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const DefaultPortablePath = `I:\utils\WinSCP\WinSCP.exe`

type LaunchOptions struct {
	Executable string
	Host       string
	Port       int
	User       string
	RemotePath string
	PrivateKey string
}

func Locate(explicit string) (string, error) {
	// Explicit command-line/config value always wins.
	if explicit != "" { return existing(explicit) }
	if p, err := exec.LookPath("WinSCP.exe"); err == nil { return p, nil }
	for _, root := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
		if root == "" { continue }
		for _, rel := range []string{filepath.Join("WinSCP", "WinSCP.exe"), filepath.Join("Programs", "WinSCP", "WinSCP.exe")} {
			p := filepath.Join(root, rel); if st, err := os.Stat(p); err == nil && !st.IsDir() { return p, nil }
		}
	}
	// User's portable layout. Keep this as a final fallback so normal
	// installations and explicit overrides remain portable for other users.
	if p, err := existing(DefaultPortablePath); err == nil { return p, nil }
	return "", fmt.Errorf("WinSCP.exe not found; pass -winscp C:\\path\\WinSCP.exe (default fallback: %s)", DefaultPortablePath)
}

func Args(o LaunchOptions) []string {
	u := &url.URL{Scheme: "sftp", Host: fmt.Sprintf("%s:%d", bracketHost(o.Host), o.Port), User: url.User(o.User)}
	if o.RemotePath != "" { u.Path = ensureSlash(o.RemotePath) }
	args := []string{u.String()}
	if o.PrivateKey != "" { args = append(args, "/privatekey="+o.PrivateKey) }
	return args
}

func Launch(o LaunchOptions) error {
	cmd := exec.Command(o.Executable, Args(o)...)
	return cmd.Start()
}

func existing(path string) (string, error) {
	p, err := filepath.Abs(os.ExpandEnv(path)); if err != nil { return "", err }
	if st, err := os.Stat(p); err != nil || st.IsDir() { return "", fmt.Errorf("WinSCP executable not found at %s", p) }
	return p, nil
}

func ensureSlash(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") { p = "/" + p }
	if !strings.HasSuffix(p, "/") { p += "/" }
	return p
}

func bracketHost(h string) string {
	h = strings.Trim(h, "[]")
	if strings.Contains(h, ":") { return "[" + h + "]" }
	return h
}
