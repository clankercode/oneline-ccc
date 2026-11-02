package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProfileValidatesExactModelAndAttestedBudgetTier(t *testing.T) {
	p := Profile{BaseURL: "https://gateway.example/v1", Model: "gpt-6.1-sol", Budget: 500000, Compact: 450000, Tier: "Standard", EnvKey: "LLMP_API_KEY", CapacityConfirmed: true}
	if err := ValidateProfile(p); err != nil {
		t.Fatal(err)
	}
	p.Tier = "Ultrafast"
	if ValidateProfile(p) == nil {
		t.Fatal("unconfirmed acceleration accepted")
	}
	p.TiersConfirmed = true
	if err := ValidateProfile(p); err != nil {
		t.Fatal(err)
	}
	p.Budget = 0
	if ValidateProfile(p) == nil {
		t.Fatal("zero budget accepted")
	}
}
func TestUnsafeEndpointsAndIdentifiersAreRefused(t *testing.T) {
	for _, endpoint := range []string{"http://remote.example/v1", "https://u:secret@gateway.example/v1", "https://gateway.example/v1?key=secret", "https://gateway.example/v1#secret"} {
		p := Profile{BaseURL: endpoint, Model: "exact", Budget: 500000, Compact: 450000, Tier: "Standard", EnvKey: "LLMP_API_KEY", CapacityConfirmed: true}
		if ValidateProfile(p) == nil {
			t.Fatal("unsafe endpoint accepted")
		}
	}
}
func TestPublicJSONDoesNotExposeEndpointOrExecutable(t *testing.T) {
	value := Snapshot{Clients: []Client{{Executable: "/private/path", Profile: Profile{BaseURL: "https://private.example/v1", Catalog: "/private/catalog"}}}}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private") {
		t.Fatal("private data in public JSON")
	}
}
