package mobaxterm

import "testing"

func TestParseSSH(t *testing.T) {
	b, err := parseSSH("vpn_1", `adp\win1`, `65.21.182.155%2255%root%%-1%-1%%%%%0%0%0%_CurrentDrive_:\ADP\NEW_PRV.ppk%%-1%0%0%0`)
	if err != nil { t.Fatal(err) }
	if b.Host != "65.21.182.155" || b.Port != 2255 || b.User != "root" { t.Fatalf("unexpected bookmark: %#v", b) }
	if b.PrivateKey != `_CurrentDrive_:\ADP\NEW_PRV.ppk` { t.Fatalf("unexpected key %q", b.PrivateKey) }
}

func TestParseSSHWithoutKey(t *testing.T) {
	b, err := parseSSH("Jetson", "Home", `192.168.24.103%22%us1%%-1%-1%%%%%0%0%0%%%-1%0%0%0`)
	if err != nil { t.Fatal(err) }
	if b.PrivateKey != "" { t.Fatalf("unexpected key %q", b.PrivateKey) }
}

func TestFind(t *testing.T) {
	items := []Bookmark{{Name:"a",Host:"10.0.0.1",Port:22,User:"root"},{Name:"b",Host:"10.0.0.1",Port:22,User:"debian"}}
	b, err := Find(items,"10.0.0.1",22,"debian"); if err != nil { t.Fatal(err) }
	if b.Name != "b" { t.Fatalf("got %s", b.Name) }
}
