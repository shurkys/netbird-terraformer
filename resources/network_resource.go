package resources

import "fmt"

type networkResource struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Address     string      `json:"address"`
	Enabled     bool        `json:"enabled"`
	Groups      []GroupInfo `json:"groups"`
}

func (h *ExtendedHandler) importNetworkResources(networks []network) error {
	count := 0
	for _, network := range networks {
		var resources []networkResource
		if err := h.service.Get(fmt.Sprintf("/api/networks/%s/resources", network.ID), &resources); err != nil {
			return fmt.Errorf("failed to fetch network resources for network %s: %w", network.ID, err)
		}
		for _, resource := range resources {
			if h.generateNetworkResource(network, resource) {
				count++
			}
		}
	}
	fmt.Printf("Imported %d network resources\n", count)
	return nil
}

func (h *ExtendedHandler) generateNetworkResource(network network, resource networkResource) bool {
	resourceName := joinedResourceName(network.Name, resource.Name)
	if resourceName == "" {
		resourceName = fmt.Sprintf("network_resource_%s", resource.ID)
	}

	groups := h.groupInfoValues(resource.Groups)
	if len(groups) == 0 {
		fmt.Printf("  Skipping network_resource %s: Terraform provider requires at least one group\n", resourceName)
		return false
	}

	h.terraform.AddResource("network_resource", resourceName, map[string]any{
		"id":          fmt.Sprintf("%s/%s", network.ID, resource.ID),
		"network_id":  h.networkValue(network.ID),
		"name":        resource.Name,
		"description": resource.Description,
		"address":     resource.Address,
		"enabled":     resource.Enabled,
		"groups":      groups,
	})
	return true
}
