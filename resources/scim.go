package resources

import "fmt"

type scimIntegration struct {
	ID                any      `json:"id"`
	ProviderName      string   `json:"provider_name"`
	Provider          string   `json:"provider"`
	Prefix            string   `json:"prefix"`
	Enabled           bool     `json:"enabled"`
	GroupPrefixes     []string `json:"group_prefixes"`
	UserGroupPrefixes []string `json:"user_group_prefixes"`
}

func (h *ExtendedHandler) importSCIM() error {
	fmt.Printf("Importing SCIM integrations...\n")
	var integrations []scimIntegration
	if err := h.service.Get("/api/integrations/scim-idp", &integrations); err != nil {
		return fmt.Errorf("failed to fetch SCIM integrations: %w", err)
	}

	for _, integration := range integrations {
		id := fmt.Sprintf("%v", integration.ID)
		providerName := integration.ProviderName
		if providerName == "" {
			providerName = integration.Provider
		}
		resourceName := joinedResourceName(providerName, integration.Prefix)
		if resourceName == "" {
			resourceName = fmt.Sprintf("scim_%s", id)
		}
		h.terraform.AddResourceWithoutImport("scim", resourceName, map[string]any{
			"provider_name":       providerName,
			"prefix":              integration.Prefix,
			"enabled":             integration.Enabled,
			"group_prefixes":      integration.GroupPrefixes,
			"user_group_prefixes": integration.UserGroupPrefixes,
		})
	}

	fmt.Printf("Imported %d SCIM integrations\n", len(integrations))
	return nil
}
