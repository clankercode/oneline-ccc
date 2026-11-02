//go:build windows

package configtxn

import (
	"encoding/binary"
	"os"
	"runtime"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const fullFileAccess = windows.ACCESS_MASK(0x1f01ff)

func currentSID() (*windows.SID, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, ErrUnsafe
	}
	return user.User.Sid, nil
}

func currentGroupSID() (*windows.SID, error) {
	group, err := windows.GetCurrentProcessToken().GetTokenPrimaryGroup()
	if err != nil {
		return nil, ErrUnsafe
	}
	return group.PrimaryGroup, nil
}

func privateDescriptor(dir bool) (*windows.SECURITY_DESCRIPTOR, error) {
	sid, err := currentSID()
	if err != nil {
		return nil, err
	}
	flags := ""
	if dir {
		flags = "OICI"
	}
	group, err := currentGroupSID()
	if err != nil {
		return nil, err
	}
	text := "O:" + sid.String() + "G:" + group.String() + "D:P(A;" + flags + ";FA;;;" + sid.String() + ")(A;" + flags + ";FA;;;SY)(A;" + flags + ";FA;;;BA)"
	sd, err := windows.SecurityDescriptorFromString(text)
	if err != nil {
		return nil, ErrUnsafe
	}
	return sd, nil
}

// Accept only the exact supported owner/protected-DACL policy. Unknown deny,
// object, inherited, partial-access and extra-principal ACEs are refused rather
// than silently lost when the stage replaces an existing file.
func verifySecurity(h windows.Handle, dir bool) error {
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return ErrUnsafe
	}
	return verifyDescriptor(sd, dir)
}

func verifyDescriptor(sd *windows.SECURITY_DESCRIPTOR, dir bool) error {
	sid, err := currentSID()
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil || !windows.EqualSid(owner, sid) {
		return ErrUnsafe
	}
	group, _, err := sd.Group()
	if err != nil || group == nil {
		return ErrUnsafe
	}
	expectedGroup, err := currentGroupSID()
	if err != nil || !windows.EqualSid(group, expectedGroup) {
		return ErrUnsafe
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return ErrUnsafe
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil || acl.AceCount != 3 {
		return ErrUnsafe
	}
	expected := map[string]int{sid.String(): 1}
	expected["S-1-5-18"]++
	expected["S-1-5-32-544"]++
	flags := uint8(0)
	if dir {
		flags = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if windows.GetAce(acl, i, &ace) != nil || ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != flags || ace.Mask != fullFileAccess || ace.Header.AceSize < 20 {
			return ErrUnsafe
		}
		principal := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !principal.IsValid() || expected[principal.String()] == 0 {
			return ErrUnsafe
		}
		expected[principal.String()]--
	}
	for _, n := range expected {
		if n != 0 {
			return ErrUnsafe
		}
	}
	runtime.KeepAlive(sd)
	return nil
}

func nativeOpen(path string, access uint32) (windows.Handle, error) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, ErrUnsafe
	}
	return windows.CreateFile(ptr, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
}

func checkNative(h windows.Handle, dir, private bool) error {
	kind, err := windows.GetFileType(h)
	if err != nil || kind != windows.FILE_TYPE_DISK {
		return ErrUnsafe
	}
	var info windows.ByHandleFileInformation
	if windows.GetFileInformationByHandle(h, &info) != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return ErrUnsafe
	}
	if (info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0) != dir {
		return ErrUnsafe
	}
	if private {
		// Refuse metadata not represented by the frozen Write API. New ordinary
		// files/directories have NORMAL/ARCHIVE/DIRECTORY attributes only.
		if info.FileAttributes & ^uint32(windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_ATTRIBUTE_ARCHIVE|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 {
			return ErrUnsafe
		}
		if !dir && info.NumberOfLinks != 1 {
			return ErrUnsafe
		}
		if err := verifySecurity(h, dir); err != nil {
			return err
		}
		if !dir {
			return noNamedStreams(h)
		}
	}
	return nil
}

// NtQueryInformationFile is handle based, so a path swap cannot redirect the
// stream check to a different file. Unknown/unsupported stream queries fail shut.
func noNamedStreams(h windows.Handle) error {
	b := make([]byte, 64*1024)
	var status windows.IO_STATUS_BLOCK
	const fileStreamInformation = 22
	if windows.NtQueryInformationFile(h, &status, &b[0], uint32(len(b)), fileStreamInformation) != nil {
		return ErrUnsafe
	}
	used := int(status.Information)
	if used < 0 || used > len(b) {
		return ErrUnsafe
	}
	for offset := 0; offset < used; {
		if used-offset < 24 {
			return ErrUnsafe
		}
		n := int(binary.LittleEndian.Uint32(b[offset+4:]))
		next := int(binary.LittleEndian.Uint32(b[offset:]))
		if n%2 != 0 || n < 0 || n > used-offset-24 {
			return ErrUnsafe
		}
		chars := make([]uint16, n/2)
		for i := range chars {
			chars[i] = binary.LittleEndian.Uint16(b[offset+24+i*2:])
		}
		if string(utf16.Decode(chars)) != "::$DATA" {
			return ErrUnsafe
		}
		if next == 0 {
			break
		}
		if next < 24+n || next > used-offset {
			return ErrUnsafe
		}
		offset += next
	}
	return nil
}

func validateHandle(f *os.File) error              { return checkNative(windows.Handle(f.Fd()), false, true) }
func ownedPath(path string, info os.FileInfo) bool { return privateMetadata(path, info, info.IsDir()) }
func privateMetadata(path string, info os.FileInfo, dir bool) bool {
	h, err := nativeOpen(path, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	return checkNative(h, dir, true) == nil
}
