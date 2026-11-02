package core

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var credentialReferenceName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ValidateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("gateway URL must have a host and no embedded credentials, query or fragment")
	}
	ip := net.ParseIP(u.Hostname())
	loopback := strings.EqualFold(u.Hostname(), "localhost") || ip != nil && ip.IsLoopback()
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback) {
		return errors.New("use HTTPS; HTTP is allowed only on loopback")
	}
	if strings.ContainsAny(raw, "\r\n\x00") {
		return errors.New("gateway URL contains control characters")
	}
	return nil
}
func ValidateProfile(p Profile) error {
	if err := ValidateEndpoint(p.BaseURL); err != nil {
		return err
	}
	if p.Model == "" || len(p.Model) > 256 || !utf8.ValidString(p.Model) || strings.IndexFunc(p.Model, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return errors.New("choose an exact UTF-8 model ID without spaces or control characters")
	}
	if p.Budget <= 0 || p.Compact <= 0 || p.Compact >= p.Budget {
		return errors.New("budget must be positive; compaction must be positive and below the budget")
	}
	if !p.CapacityConfirmed {
		return errors.New("confirm this route supports the selected context budget")
	}
	if !validCredentialReference(p.EnvKey) {
		return errors.New("credential reference must be an environment variable name that is not reserved by the runtime")
	}
	switch p.Tier {
	case "", "Standard", "default":
	case "Fast", "Ultrafast", "fast", "priority", "ultrafast":
		if !p.TiersConfirmed {
			return errors.New("confirm accelerated tier availability for this exact route and model")
		}
	default:
		return errors.New("unsupported request tier")
	}
	return nil
}

// Mirror credentials.ChildEnv's reference policy without coupling domain
// validation to OS credential access. The cross-package contract test checks
// this list against the launcher for all reserved names and their case variants.
func validCredentialReference(name string) bool {
	if !credentialReferenceName.MatchString(name) {
		return false
	}
	switch strings.ToUpper(name) {
	case "PATH", "HOME", "USER", "LOGNAME", "USERNAME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "SYSTEMROOT", "WINDIR", "COMSPEC", "PATHEXT", "TEMP", "TMP", "TMPDIR", "LANG", "LANGUAGE", "LC_ALL", "LC_CTYPE", "LC_MESSAGES", "LC_COLLATE", "LC_NUMERIC", "LC_TIME", "LC_MONETARY", "TERM", "COLORTERM", "TERMINFO", "TERMINFO_DIRS", "TZ", "SHELL", "XDG_RUNTIME_DIR",
		"CODEX_PROFILE", "CODEX_HOME", "LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES", "DYLD_LIBRARY_PATH", "NODE_OPTIONS", "NODE_PATH", "BASH_ENV", "ENV", "PYTHONPATH", "PYTHONHOME", "GIT_CONFIG", "GIT_CONFIG_COUNT":
		return false
	}
	return true
}

func Defaults() Profile {
	return Profile{Model: "gpt-6.1-sol", Budget: 500000, Compact: 450000, Tier: "Standard", EnvKey: "LLMP_API_KEY"}
}
