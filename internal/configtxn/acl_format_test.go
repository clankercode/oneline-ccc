package configtxn

import (
	"encoding/binary"
	"testing"
)

func TestDarwinACLResultRequiresProvenAbsence(t *testing.T) {
	absent := make([]byte, 12)
	binary.LittleEndian.PutUint32(absent, 12)
	if !darwinACLAbsent(absent) {
		t.Fatal("absent native ACL refused")
	}
	noacl := make([]byte, 56)
	binary.LittleEndian.PutUint32(noacl, 56)
	binary.LittleEndian.PutUint32(noacl[4:], 8)
	binary.LittleEndian.PutUint32(noacl[8:], 44)
	binary.LittleEndian.PutUint32(noacl[12:], 0x012cc16d)
	binary.LittleEndian.PutUint32(noacl[48:], 0xffffffff)
	if !darwinACLAbsent(noacl) {
		t.Fatal("native NOACL refused")
	}
	for _, kind := range []string{"empty-acl", "named-acl", "acl-flags", "owner-guid", "bad-offset", "truncated", "bad-magic"} {
		t.Run(kind, func(t *testing.T) {
			b := append([]byte(nil), noacl...)
			switch kind {
			case "empty-acl":
				binary.LittleEndian.PutUint32(b[48:], 0)
			case "named-acl":
				binary.LittleEndian.PutUint32(b[48:], 1)
			case "acl-flags":
				b[52] = 1
			case "owner-guid":
				b[16] = 1
			case "bad-offset":
				binary.LittleEndian.PutUint32(b[4:], 0xffffffff)
			case "truncated":
				b = b[:55]
			case "bad-magic":
				b[12] = 0
			}
			if darwinACLAbsent(b) {
				t.Fatal("ambiguous or meaningful native ACL accepted")
			}
		})
	}
}
