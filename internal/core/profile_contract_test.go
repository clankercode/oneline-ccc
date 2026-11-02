package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/clankercode/oneline-ccc/internal/core"
	"github.com/clankercode/oneline-ccc/internal/credentials"
)

func attestedProfile() core.Profile {
	p := core.Defaults()
	p.BaseURL = "https://gateway.example/v1"
	p.CapacityConfirmed = true
	return p
}

func TestProfileRejectsMalformedUTF8ModelIDs(t *testing.T) {
	for _, model := range []string{"Exact-\xff", "Exact-\xc0\xaf", "Exact-\xed\xa0\x80", "Exact-\xf0\x9f"} {
		p := attestedProfile()
		p.Model = model
		if core.ValidateProfile(p) == nil {
			t.Error("malformed UTF-8 model ID accepted")
		}
	}
}

func TestProfileValidUnicodeModelIDsRoundTripExactly(t *testing.T) {
	for _, model := range []string{"Exact/Model", "Modèle/模型", "Exact-🚀", "Exact-\ufffd", `Exact-\ud800`} {
		p := attestedProfile()
		p.Model = model
		if err := core.ValidateProfile(p); err != nil {
			t.Fatalf("valid model rejected: %v", err)
		}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var decoded core.Profile
		if err := json.Unmarshal(data, &decoded); err != nil || decoded.Model != model {
			t.Fatal("exact model changed during JSON round trip")
		}
	}
}

func TestProfileCredentialReferencesMatchChildEnv(t *testing.T) {
	// Keep the core-only validator aligned with the launcher's private policy.
	// This contract test crosses packages without a production dependency.
	reserved := strings.Fields(`PATH HOME USER LOGNAME USERNAME USERPROFILE HOMEDRIVE HOMEPATH
SYSTEMROOT WINDIR COMSPEC PATHEXT TEMP TMP TMPDIR LANG LANGUAGE LC_ALL LC_CTYPE LC_MESSAGES
LC_COLLATE LC_NUMERIC LC_TIME LC_MONETARY TERM COLORTERM TERMINFO TERMINFO_DIRS TZ SHELL XDG_RUNTIME_DIR
CODEX_PROFILE CODEX_HOME LD_PRELOAD LD_LIBRARY_PATH DYLD_INSERT_LIBRARIES DYLD_LIBRARY_PATH
NODE_OPTIONS NODE_PATH BASH_ENV ENV PYTHONPATH PYTHONHOME GIT_CONFIG GIT_CONFIG_COUNT`)
	for _, name := range reserved {
		for _, spelling := range []string{name, strings.ToLower(name), name[:1] + strings.ToLower(name[1:])} {
			t.Run(spelling, func(t *testing.T) {
				p := attestedProfile()
				p.EnvKey = spelling
				if core.ValidateProfile(p) == nil || credentials.ChildEnv(spelling, "fixture-secret") != nil {
					t.Fatal("reserved runtime variable accepted as a credential reference")
				}
			})
		}
	}
	for _, name := range []string{"", "BAD=NAME", "9KEY", "bad-name", "bad key", "KEY\x00", "KEY\xff"} {
		p := attestedProfile()
		p.EnvKey = name
		if core.ValidateProfile(p) == nil || credentials.ChildEnv(name, "fixture-secret") != nil {
			t.Error("invalid credential reference accepted")
		}
	}
	for _, name := range []string{"LLMP_API_KEY", "OPENAI_API_KEY", "_SELECTED_KEY", "selected_key", "A"} {
		p := attestedProfile()
		p.EnvKey = name
		if core.ValidateProfile(p) != nil || credentials.ChildEnv(name, "fixture-secret") == nil {
			t.Error("valid credential reference refused")
		}
	}
}
