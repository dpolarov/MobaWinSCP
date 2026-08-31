package mobaxterm

import (
	"fmt"
	"os"
	"path/filepath"
)

func LocateINI(explicit string) (string, error) {
	if explicit != "" { return requireFile(explicit) }
	var candidates []string
	if exe, err := os.Executable(); err == nil { candidates = append(candidates, filepath.Join(filepath.Dir(exe), "MobaXterm.ini")) }
	if appdata := os.Getenv("APPDATA"); appdata != "" { candidates = append(candidates, filepath.Join(appdata, "MobaXterm", "MobaXterm.ini")) }
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Documents", "MobaXterm", "MobaXterm.ini"),
			filepath.Join(home, "OneDrive", "Documents", "MobaXterm", "MobaXterm.ini"))
	}
	for _, p := range candidates { if st, err := os.Stat(p); err == nil && !st.IsDir() { return p, nil } }
	return "", fmt.Errorf("MobaXterm.ini not found; pass -ini C:\\path\\MobaXterm.ini")
}

func requireFile(path string) (string, error) {
	p, err := filepath.Abs(os.ExpandEnv(path)); if err != nil { return "", err }
	st, err := os.Stat(p); if err != nil { return "", err }
	if st.IsDir() { return "", fmt.Errorf("%s is a directory", p) }
	return p, nil
}
