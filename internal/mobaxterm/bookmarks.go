package mobaxterm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const sshPrefix = "#109#0%"

type Bookmark struct {
	Name       string
	Folder     string
	Host       string
	Port       int
	User       string
	PrivateKey string
	RawFields  []string
}

func ParseFile(path string) ([]Bookmark, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()

	var out []Bookmark
	var section, folder string
	s := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, ";") { continue }
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			folder = ""
			continue
		}
		if !strings.HasPrefix(section, "Bookmarks") { continue }
		if strings.HasPrefix(line, "SubRep=") { folder = strings.TrimPrefix(line, "SubRep="); continue }
		idx := strings.IndexByte(line, '=')
		if idx <= 0 { continue }
		name, raw := line[:idx], line[idx+1:]
		if !strings.HasPrefix(raw, sshPrefix) { continue }
		b, err := parseSSH(name, folder, strings.TrimPrefix(raw, sshPrefix))
		if err != nil { continue } // tolerate unknown/new bookmark layouts
		out = append(out, b)
	}
	if err := s.Err(); err != nil { return nil, err }
	return out, nil
}

func parseSSH(name, folder, payload string) (Bookmark, error) {
	fields := strings.Split(payload, "%")
	if len(fields) < 3 { return Bookmark{}, fmt.Errorf("SSH bookmark %q has too few fields", name) }
	port, err := strconv.Atoi(fields[1])
	if err != nil { return Bookmark{}, fmt.Errorf("SSH bookmark %q has invalid port", name) }
	b := Bookmark{Name: name, Folder: folder, Host: fields[0], Port: port, User: fields[2], RawFields: fields}
	// MobaXterm's bookmark format is internal. Do not trust one hard-coded
	// index for the key; locate a plausible key path among the SSH fields.
	for _, field := range fields {
		v := strings.TrimSpace(field)
		low := strings.ToLower(v)
		if strings.HasSuffix(low, ".ppk") || strings.HasSuffix(low, ".pem") || strings.HasSuffix(low, "id_rsa") || strings.HasSuffix(low, "id_ed25519") {
			b.PrivateKey = v
			break
		}
	}
	return b, nil
}

func Find(bookmarks []Bookmark, host string, port int, user string) (Bookmark, error) {
	var hostPort []Bookmark
	for _, b := range bookmarks {
		if strings.EqualFold(strings.Trim(b.Host, "[]"), strings.Trim(host, "[]")) && b.Port == port {
			hostPort = append(hostPort, b)
			if user != "" && strings.EqualFold(b.User, user) { return b, nil }
		}
	}
	if len(hostPort) == 1 { return hostPort[0], nil }
	if len(hostPort) > 1 { return Bookmark{}, fmt.Errorf("multiple bookmarks match %s:%d but none matches user %q", host, port, user) }
	return Bookmark{}, fmt.Errorf("no SSH bookmark matches %s:%d user %q", host, port, user)
}

func ResolvePortablePath(value, iniPath string) string {
	if value == "" { return "" }
	const marker = "_CurrentDrive_:"
	if strings.HasPrefix(value, marker) {
		drive := filepath.VolumeName(iniPath)
		if drive == "" { drive = filepath.VolumeName(os.Args[0]) }
		rest := strings.TrimPrefix(value, marker)
		return filepath.Clean(drive + string(os.PathSeparator) + strings.TrimLeft(rest, `\\/`))
	}
	return os.ExpandEnv(value)
}
