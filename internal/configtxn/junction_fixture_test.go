package configtxn

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func junctionFixture(target string) []byte {
	sub := append(utf16.Encode([]rune(`\??\`+target)), 0)
	display := append(utf16.Encode([]rune(target)), 0)
	b := make([]byte, 16+(len(sub)+len(display))*2)
	binary.LittleEndian.PutUint32(b, 0xa0000003)
	binary.LittleEndian.PutUint16(b[4:], uint16(len(b)-8))
	binary.LittleEndian.PutUint16(b[10:], uint16((len(sub)-1)*2))
	binary.LittleEndian.PutUint16(b[12:], uint16(len(sub)*2))
	binary.LittleEndian.PutUint16(b[14:], uint16((len(display)-1)*2))
	for i, c := range append(sub, display...) {
		binary.LittleEndian.PutUint16(b[16+i*2:], c)
	}
	return b
}
func TestJunctionFixtureProtocol(t *testing.T) {
	target := `C:\fixture`
	b := junctionFixture(target)
	if binary.LittleEndian.Uint32(b) != 0xa0000003 || int(binary.LittleEndian.Uint16(b[4:])) != len(b)-8 {
		t.Fatal("invalid reparse header")
	}
	name := func(offset, length int) string {
		start := 16 + offset
		if start < 16 || start+length+2 > len(b) {
			t.Fatal("invalid string range")
		}
		chars := make([]uint16, length/2)
		for i := range chars {
			chars[i] = binary.LittleEndian.Uint16(b[start+i*2:])
		}
		if binary.LittleEndian.Uint16(b[start+length:]) != 0 {
			t.Fatal("unterminated reparse string")
		}
		return string(utf16.Decode(chars))
	}
	sub := name(int(binary.LittleEndian.Uint16(b[8:])), int(binary.LittleEndian.Uint16(b[10:])))
	print := name(int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:])))
	if sub != `\??\`+target || print != target {
		t.Fatal("invalid NT substitute or display name")
	}
}
