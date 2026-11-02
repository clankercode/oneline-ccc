package credentials

import (
	"os"
	"strings"
)

// essentialEnv deliberately excludes client profiles, proxy credentials, plugin
// hooks, shell startup injection and ambient API keys/tokens. The selected
// credential is the only entry appended to this OS/runtime allowlist.
func essentialEnv(name string) bool {
	switch strings.ToUpper(name) {
	case "PATH", "HOME", "USER", "LOGNAME", "USERNAME", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "SYSTEMROOT", "WINDIR", "COMSPEC", "PATHEXT", "TEMP", "TMP", "TMPDIR", "LANG", "LANGUAGE", "LC_ALL", "LC_CTYPE", "LC_MESSAGES", "LC_COLLATE", "LC_NUMERIC", "LC_TIME", "LC_MONETARY", "TERM", "COLORTERM", "TERMINFO", "TERMINFO_DIRS", "TZ", "SHELL", "XDG_RUNTIME_DIR":
		return true
	}
	return false
}

// ChildEnv returns a fresh minimal environment, or nil for an invalid reference
// or key. The caller must refuse to launch if it returns nil. It never mutates
// the parent environment; do not log the result or put it in command arguments.
func ChildEnv(envName, key string) []string {
	if !validReference(envName) || !validKey(key) {
		return nil
	}
	out := make([]string, 0, 32)
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if ok && essentialEnv(name) {
			out = append(out, entry)
		}
	}
	return append(out, envName+"="+key)
}
