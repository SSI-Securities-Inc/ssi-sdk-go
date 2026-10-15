package device

import "testing"

func TestParseRegMachineGUID(t *testing.T) {
	out := "\r\nHKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Cryptography\r\n    MachineGuid    REG_SZ    11111111-2222-3333-4444-555555555555\r\n\r\n"
	if got := parseRegMachineGUID(out); got != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("got %q", got)
	}
	if parseRegMachineGUID("garbage") != "" {
		t.Fatal("unparsable output must give an empty id")
	}
}

func TestIDIsStableAndNotEmpty(t *testing.T) {
	a, b := ID(), ID()
	if a == "" || a != b {
		t.Fatalf("ID must be non-empty and stable: %q vs %q", a, b)
	}
}
