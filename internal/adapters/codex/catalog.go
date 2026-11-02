package codex

import (
	"context"
	"errors"

	"github.com/clankercode/oneline-ccc/internal/codexrpc"
)

// validateCatalog asks a fresh native model manager to deserialize the full
// catalog. A config/read result alone does not establish usable model metadata.
func (a *Adapter) validateCatalog(ctx context.Context, stage, model string, confirmedBoth bool) error {
	return codexrpc.Run(ctx, a.Program, stage, func(c *codexrpc.Client) error {
		var response struct {
			Data []struct {
				Model        string
				ServiceTiers []struct{ ID string }
			}
		}
		if err := c.Call("model/list", map[string]any{"includeHidden": true, "limit": 100}, &response); err != nil {
			return err
		}
		for _, entry := range response.Data {
			if entry.Model != model {
				continue
			}
			fast, ultra := false, false
			for _, tier := range entry.ServiceTiers {
				fast = fast || tier.ID == "priority"
				ultra = ultra || tier.ID == "ultrafast"
			}
			if !confirmedBoth || fast && ultra {
				return nil
			}
		}
		return errors.New("Fresh native model catalog did not expose the selected model and confirmed tiers")
	})
}
