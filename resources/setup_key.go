package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

type setupKey struct {
	ID                  any      `json:"id"`
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	Revoked             bool     `json:"revoked"`
	AutoGroups          []string `json:"auto_groups"`
	UsageLimit          int      `json:"usage_limit"`
	Ephemeral           bool     `json:"ephemeral"`
	AllowExtraDNSLabels bool     `json:"allow_extra_dns_labels"`
}

func (h *ExtendedHandler) importSetupKeys() error {
	fmt.Printf("Importing setup keys...\n")
	var setupKeys []setupKey
	if err := h.service.Get("/api/setup-keys", &setupKeys); err != nil {
		return fmt.Errorf("failed to fetch setup keys: %w", err)
	}

	for _, setupKey := range setupKeys {
		id := fmt.Sprintf("%v", setupKey.ID)
		resourceName := lib.SanitizeResourceName(setupKey.Name)
		if resourceName == "" {
			resourceName = fmt.Sprintf("setup_key_%s", id)
		}

		h.terraform.AddResource("setup_key", resourceName, map[string]any{
			"id":                     id,
			"name":                   setupKey.Name,
			"type":                   setupKey.Type,
			"revoked":                setupKey.Revoked,
			"auto_groups":            h.groupValues(setupKey.AutoGroups),
			"usage_limit":            setupKey.UsageLimit,
			"ephemeral":              setupKey.Ephemeral,
			"allow_extra_dns_labels": setupKey.AllowExtraDNSLabels,
		})
	}

	fmt.Printf("Imported %d setup keys\n", len(setupKeys))
	return nil
}
