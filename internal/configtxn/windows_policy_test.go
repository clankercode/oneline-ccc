//go:build windows

package configtxn

import (
	"errors"
	"golang.org/x/sys/windows"
	"testing"
)

func TestWindowsDescriptorPolicy(t *testing.T) {
	for _, dir := range []bool{false, true} {
		sd, err := privateDescriptor(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyDescriptor(sd, dir); err != nil {
			t.Fatal("supported descriptor refused", err)
		}
		if err := verifyDescriptor(sd, !dir); !errors.Is(err, ErrUnsafe) {
			t.Fatal("wrong inheritance accepted", err)
		}
	}
	sid, err := currentSID()
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"O:" + sid.String() + "D:P(A;;FA;;;WD)",
		"O:" + sid.String() + "D:(A;;FA;;;" + sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)",
		"O:SYD:P(A;;FA;;;" + sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)",
		"O:" + sid.String() + "D:P(A;;FR;;;" + sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)",
	} {
		sd, err := windows.SecurityDescriptorFromString(text)
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyDescriptor(sd, false); !errors.Is(err, ErrUnsafe) {
			t.Fatal("unsupported ACL accepted", err)
		}
	}
}

func TestWindowsPathPolicy(t *testing.T) {
	for _, path := range []string{`C:\safe\config`, `D:\space allowed\file.toml`, `C:\`} {
		if err := platformPath(path); err != nil {
			t.Fatal("valid path refused", err)
		}
	}
	for _, path := range []string{`\\server\share\config`, `\\?\C:\config`, `C:\config:stream`, `C:\config.`, `C:\parent \config`, `C:\NUL`, `C:\COM1.txt`, `C:\LPT1`, `C:\CONOUT$`, `C:\bad?name`} {
		if err := platformPath(path); !errors.Is(err, ErrUnsafe) {
			t.Fatal("unsafe path accepted", err)
		}
	}
	if !reserved(`C:\private\state`, `c:\PRIVATE\STATE\config`) || !reserved(`C:\state`, `C:\.CCC-CONFIGTXN.LOCK`) {
		t.Fatal("case alias bypassed reserved paths")
	}
}
