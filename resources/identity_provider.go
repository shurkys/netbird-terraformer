package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

type identityProvider struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer"`
	ClientID string `json:"client_id"`
}

func (h *ExtendedHandler) importIdentityProviders() error {
	fmt.Printf("Importing identity providers...\n")
	var providers []identityProvider
	if err := h.service.Get("/api/identity-providers", &providers); err != nil {
		return fmt.Errorf("failed to fetch identity providers: %w", err)
	}

	for _, provider := range providers {
		resourceName := lib.SanitizeResourceName(provider.Name)
		if resourceName == "" {
			resourceName = fmt.Sprintf("identity_provider_%s", provider.ID)
		}

		h.terraform.AddResourceWithoutImport("identity_provider", resourceName, map[string]any{
			"type":          provider.Type,
			"name":          provider.Name,
			"issuer":        provider.Issuer,
			"client_id":     provider.ClientID,
			"client_secret": "REPLACE_WITH_CLIENT_SECRET",
		})
	}

	fmt.Printf("Imported %d identity providers\n", len(providers))
	return nil
}
