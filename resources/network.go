package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

type network struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *ExtendedHandler) importNetworks() error {
	fmt.Printf("Importing networks...\n")
	var networks []network
	if err := h.service.Get("/api/networks", &networks); err != nil {
		return fmt.Errorf("failed to fetch networks: %w", err)
	}

	for _, network := range networks {
		resourceName := h.networkResourceName(network)
		h.networkMapping[network.ID] = resourceName
		if h.selected["network"] {
			h.terraform.AddResource("network", resourceName, map[string]any{
				"id":          network.ID,
				"name":        network.Name,
				"description": network.Description,
			})
		}
	}
	if h.selected["network"] {
		fmt.Printf("Imported %d networks\n", len(networks))
	}

	if h.selected["network_resource"] {
		if err := h.importNetworkResources(networks); err != nil {
			return err
		}
	}
	if h.selected["network_router"] {
		if err := h.importNetworkRouters(networks); err != nil {
			return err
		}
	}

	return nil
}

func (h *ExtendedHandler) networkResourceName(network network) string {
	resourceName := lib.SanitizeResourceName(network.Name)
	if resourceName == "" {
		resourceName = fmt.Sprintf("network_%s", network.ID)
	}
	return resourceName
}

func (h *ExtendedHandler) networkValue(networkID string) string {
	if h.useNetworkRefs {
		if resourceName, exists := h.networkMapping[networkID]; exists {
			return lib.CreateTerraformReference("network", resourceName)
		}
	}
	return networkID
}
