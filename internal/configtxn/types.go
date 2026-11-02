// Package configtxn owns protected backups and drift-aware file replacement.
package configtxn

import "io/fs"

type Write struct {
	Path    string
	Before  []byte
	Existed bool
	Mode    fs.FileMode
	Data    []byte
}
type Result struct {
	ID, Outcome      string
	Changed, Backups []string
}
type Store struct {
	Root       string
	checkpoint func(string) error
}
