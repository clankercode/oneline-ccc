// Package core contains client-management values and the foreground UI port.
package core

import (
	"context"
	"os/exec"
)

type Profile struct {
	BaseURL           string `json:"-"`
	Model             string `json:"model"`
	Budget            int64  `json:"budget"`
	Compact           int64  `json:"compact"`
	Tier              string `json:"tier"`
	EnvKey            string `json:"credential_reference"`
	Catalog           string `json:"-"`
	CapacityConfirmed bool   `json:"capacity_confirmed"`
	TiersConfirmed    bool   `json:"tiers_confirmed"`
}
type Desired struct {
	Enabled bool
	Profile Profile
}
type Client struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Installed  bool    `json:"installed"`
	Supported  bool    `json:"supported"`
	Managed    bool    `json:"managed"`
	Version    string  `json:"version,omitempty"`
	Status     string  `json:"status"`
	Reason     string  `json:"reason,omitempty"`
	Executable string  `json:"-"`
	Profile    Profile `json:"profile"`
}
type Change struct{ Field, Before, After string }
type Plan struct {
	ID       string   `json:"id"`
	Revision string   `json:"revision"`
	ClientID string   `json:"client_id"`
	Changes  []Change `json:"changes"`
	Desired  Desired  `json:"-"`
	Noop     bool     `json:"noop"`
}
type Receipt struct {
	ID       string `json:"id"`
	ClientID string `json:"client_id"`
	Outcome  string `json:"outcome"`
	Summary  string `json:"summary"`
}
type ServiceInfo struct{ Backend, Status, Detail string }
type Snapshot struct {
	Clients    []Client    `json:"clients"`
	Receipts   []Receipt   `json:"receipts"`
	Credential string      `json:"credential_status"`
	Service    ServiceInfo `json:"monitor"`
}
type ModelInfo struct {
	ID      string
	Context int64
	Source  string
}

// Backend executes explicit foreground actions; rendering never performs writes.
type Backend interface {
	Inspect(context.Context) (Snapshot, error)
	Plan(context.Context, string, Desired) (Plan, error)
	Apply(context.Context, string, string) (Receipt, error)
	Command(context.Context, string, string) (*exec.Cmd, error)
	SaveKey(context.Context, string, bool) error
	Discover(context.Context, string) ([]ModelInfo, error)
	Service(context.Context, string) (ServiceInfo, error)
	RepairBrief(context.Context) (string, error)
}
