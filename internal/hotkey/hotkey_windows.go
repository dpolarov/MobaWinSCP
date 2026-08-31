//go:build windows

package hotkey

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/dpolarov/MobaWinSCP/internal/session"
)

const (
	wmHotkey        = 0x0312
	modAlt          = 0x0001
	modControl      = 0x0002
	modShift        = 0x0004
	modWin          = 0x0008
	modNoRepeat     = 0x4000
	inputKeyboard   = 1
	keyeventfKeyUp  = 0x0002
	keyeventfUnicode = 0x0004
	marker          = "MOBWINSCP|"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procGetForeground    = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW   = user32.NewProc("GetWindowTextW")
	procGetWindowTextLen = user32.NewProc("GetWindowTextLengthW")
	procGetClassNameW    = user32.NewProc("GetClassNameW")
	procEnumChildWindows = user32.NewProc("EnumChildWindows")
	procSendInput        = user32.NewProc("SendInput")
)

type Options struct {
	Hotkey  string
	Verbose bool
	OnSession func(session.Session) error
}

type point struct { X, Y int32 }
type msg struct {
	Hwnd uintptr
	Message uint32
	_ uint32
	WParam uintptr
	LParam uintptr
	Time uint32
	Pt point
	LPrivate uint32
}

type keybdInput struct {
	VK uint16
	Scan uint16
	Flags uint32
	Time uint32
	ExtraInfo uintptr
}

type input struct {
	Type uint32
	_ uint32
	Ki keybdInput
	_ [8]byte
}

func Run(o Options) error {
	if o.OnSession == nil { return fmt.Errorf("hotkey callback is nil") }
	mods, vk, err := parseHotkey(o.Hotkey)
	if err != nil { return err }
	const id = 0x4D57
	r, _, callErr := procRegisterHotKey.Call(0, id, uintptr(mods|modNoRepeat), uintptr(vk))
	if r == 0 { return fmt.Errorf("cannot register hotkey %s: %v", o.Hotkey, callErr) }
	defer procUnregisterHotKey.Call(0, id)

	fmt.Printf("MobaWinSCP listener active. Press %s in an SSH terminal.\n", o.Hotkey)
	for {
		var m msg
		r, _, callErr := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) == -1 { return fmt.Errorf("GetMessageW failed: %v", callErr) }
		if r == 0 { return nil }
		if m.Message != wmHotkey || m.WParam != id { continue }
		if err := handle(o); err != nil { fmt.Printf("MobaWinSCP hotkey: %v\n", err) }
	}
}

func handle(o Options) error {
	hwnd, _, _ := procGetForeground.Call()
	if hwnd == 0 { return fmt.Errorf("no foreground window") }
	title := windowText(hwnd)
	class := className(hwnd)
	if o.Verbose { fmt.Printf("Foreground: class=%q title=%q\n", class, title) }
	if !strings.Contains(strings.ToLower(title+" "+class), "moba") {
		return fmt.Errorf("foreground window does not look like MobaXterm")
	}

	// Keep the marker in the terminal title briefly so the local helper can
	// read it from the MobaXterm window hierarchy. No remote helper is needed.
	cmd := `printf '\033]0;MOBWINSCP|%s|%s|%s\007' "$USER" "$SSH_CONNECTION" "$PWD"; sleep 0.5`
	if err := sendText(cmd); err != nil { return err }
	if err := sendEnter(); err != nil { return err }

	var payload string
	deadline := time.Now().Add(450 * time.Millisecond)
	for time.Now().Before(deadline) {
		payload = findMarker(hwnd)
		if payload != "" { break }
		time.Sleep(10 * time.Millisecond)
	}
	if payload == "" {
		return fmt.Errorf("session marker was not visible; in the SSH bookmark disable 'Lock terminal title' and retry")
	}
	if o.Verbose { fmt.Printf("Detected: %s\n", payload) }
	live, err := parsePayload(payload)
	if err != nil { return err }
	return o.OnSession(live)
}

func parsePayload(v string) (session.Session, error) {
	i := strings.Index(v, marker)
	if i < 0 { return session.Session{}, fmt.Errorf("invalid session marker") }
	v = v[i+len(marker):]
	parts := strings.SplitN(v, "|", 3)
	if len(parts) != 3 { return session.Session{}, fmt.Errorf("invalid session marker fields") }
	user := strings.TrimSpace(parts[0])
	conn := strings.Fields(parts[1])
	pwd := strings.TrimSpace(parts[2])
	if len(conn) != 4 { return session.Session{}, fmt.Errorf("invalid SSH_CONNECTION %q", parts[1]) }
	port, err := strconv.Atoi(conn[3])
	if err != nil || port < 1 || port > 65535 { return session.Session{}, fmt.Errorf("invalid SSH port %q", conn[3]) }
	if user == "" || conn[2] == "" { return session.Session{}, fmt.Errorf("missing SSH user/host") }
	if pwd == "" { pwd = "/" }
	return session.Session{User: user, Host: conn[2], Port: port, RemotePath: pwd}, nil
}

func findMarker(parent uintptr) string {
	if t := windowText(parent); strings.Contains(t, marker) { return t }
	var found string
	cb := syscall.NewCallback(func(hwnd, lParam uintptr) uintptr {
		if found != "" { return 0 }
		if t := windowText(hwnd); strings.Contains(t, marker) { found = t; return 0 }
		return 1
	})
	procEnumChildWindows.Call(parent, cb, 0)
	return found
}

func windowText(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLen.Call(hwnd)
	if n == 0 { return "" }
	buf := make([]uint16, int(n)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func className(hwnd uintptr) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 { return "" }
	return syscall.UTF16ToString(buf[:n])
}

func sendText(s string) error {
	for _, u := range utf16.Encode([]rune(s)) {
		if err := sendKey(0, u, keyeventfUnicode); err != nil { return err }
		if err := sendKey(0, u, keyeventfUnicode|keyeventfKeyUp); err != nil { return err }
	}
	return nil
}

func sendEnter() error {
	if err := sendKey(0x0D, 0, 0); err != nil { return err }
	return sendKey(0x0D, 0, keyeventfKeyUp)
}

func sendKey(vk, scan uint16, flags uint32) error {
	in := input{Type: inputKeyboard, Ki: keybdInput{VK: vk, Scan: scan, Flags: flags}}
	r, _, callErr := procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	if r != 1 { return fmt.Errorf("SendInput failed: %v", callErr) }
	return nil
}

func parseHotkey(v string) (uint32, uint32, error) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(v)), "+")
	if len(parts) < 2 { return 0, 0, fmt.Errorf("invalid hotkey %q", v) }
	var mods uint32
	var vk uint32
	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "CTRL", "CONTROL": mods |= modControl
		case "ALT": mods |= modAlt
		case "SHIFT": mods |= modShift
		case "WIN", "WINDOWS": mods |= modWin
		default:
			if vk != 0 { return 0, 0, fmt.Errorf("multiple keys in hotkey %q", v) }
			if len(p) == 1 && ((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= '0' && p[0] <= '9')) { vk = uint32(p[0]); continue }
			if strings.HasPrefix(p, "F") {
				n, err := strconv.Atoi(strings.TrimPrefix(p, "F")); if err == nil && n >= 1 && n <= 24 { vk = uint32(0x70 + n - 1); continue }
			}
			return 0, 0, fmt.Errorf("unsupported key %q", p)
		}
	}
	if vk == 0 { return 0, 0, fmt.Errorf("hotkey %q has no key", v) }
	return mods, vk, nil
}
