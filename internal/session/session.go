package session

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Session struct {
	User       string
	Host       string
	Port       int
	RemotePath string
}

// Detect reads the live OpenSSH environment exported by the remote shell.
// SSH_CONNECTION is: client_ip client_port server_ip server_port.
// MOBWINSCP_* overrides are useful for testing and unusual SSH setups.
func Detect() (Session, error) {
	s := Session{
		User:       firstNonEmpty(os.Getenv("MOBWINSCP_USER"), os.Getenv("USER"), os.Getenv("LOGNAME")),
		RemotePath: firstNonEmpty(os.Getenv("MOBWINSCP_PWD"), os.Getenv("PWD")),
	}

	if host := os.Getenv("MOBWINSCP_HOST"); host != "" {
		s.Host = host
		p, err := parsePort(firstNonEmpty(os.Getenv("MOBWINSCP_PORT"), "22"))
		if err != nil { return Session{}, err }
		s.Port = p
	} else {
		fields := strings.Fields(os.Getenv("SSH_CONNECTION"))
		if len(fields) != 4 {
			return Session{}, fmt.Errorf("SSH_CONNECTION is unavailable; run MobaWinSCP from an SSH shell or set MOBWINSCP_HOST/PORT")
		}
		s.Host = fields[2]
		p, err := parsePort(fields[3])
		if err != nil { return Session{}, fmt.Errorf("invalid SSH_CONNECTION server port: %w", err) }
		s.Port = p
	}

	if s.User == "" { return Session{}, fmt.Errorf("cannot determine remote username") }
	if s.RemotePath == "" { s.RemotePath = "/" }
	return s, nil
}

func parsePort(v string) (int, error) {
	p, err := strconv.Atoi(v)
	if err != nil || p < 1 || p > 65535 { return 0, fmt.Errorf("invalid port %q", v) }
	return p, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values { if v != "" { return v } }
	return ""
}

func SameHost(a, b string) bool {
	if strings.EqualFold(a, b) { return true }
	aIP, aErr := net.ParseIP(strings.Trim(a, "[]")), error(nil)
	bIP, bErr := net.ParseIP(strings.Trim(b, "[]")), error(nil)
	_ = aErr; _ = bErr
	return aIP != nil && bIP != nil && aIP.Equal(bIP)
}
