package configtxn

import "encoding/binary"

// Parse the single ATTR_CMN_EXTENDED_SECURITY result, including its native
// attrreference and kauth_filesec layout. Empty ACLs are meaningful (deny-all),
// so only absent security data or KAUTH_FILESEC_NOACL with no flags is accepted.
func darwinACLAbsent(b []byte) bool {
	if len(b) < 12 {
		return false
	}
	total := int(binary.LittleEndian.Uint32(b))
	if total < 12 || total > len(b) {
		return false
	}
	offset := int(int32(binary.LittleEndian.Uint32(b[4:])))
	n := int(binary.LittleEndian.Uint32(b[8:]))
	if n == 0 {
		return total == 12
	}
	start := 4 + offset
	if start < 12 || n != 44 || start > total-n {
		return false
	}
	security := b[start : start+n]
	if binary.LittleEndian.Uint32(security) != 0x012cc16d {
		return false
	}
	for _, v := range security[4:36] {
		if v != 0 {
			return false
		}
	}
	return binary.LittleEndian.Uint32(security[36:]) == 0xffffffff && binary.LittleEndian.Uint32(security[40:]) == 0
}
