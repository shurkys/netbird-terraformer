package resources

import "fmt"

type networkRouter struct {
	ID         string   `json:"id"`
	Peer       string   `json:"peer"`
	PeerGroups []string `json:"peer_groups"`
	Metric     int      `json:"metric"`
	Masquerade bool     `json:"masquerade"`
	Enabled    bool     `json:"enabled"`
}

func (h *ExtendedHandler) importNetworkRouters(networks []network) error {
	count := 0
	for _, network := range networks {
		var routers []networkRouter
		if err := h.service.Get(fmt.Sprintf("/api/networks/%s/routers", network.ID), &routers); err != nil {
			return fmt.Errorf("failed to fetch network routers for network %s: %w", network.ID, err)
		}
		for _, router := range routers {
			h.generateNetworkRouter(network, router)
			count++
		}
	}
	fmt.Printf("Imported %d network routers\n", count)
	return nil
}

func (h *ExtendedHandler) generateNetworkRouter(network network, router networkRouter) {
	resourceName := joinedResourceName(network.Name, router.ID)
	if resourceName == "" {
		resourceName = fmt.Sprintf("network_router_%s", router.ID)
	}

	h.terraform.AddResource("network_router", resourceName, map[string]any{
		"id":          fmt.Sprintf("%s/%s", network.ID, router.ID),
		"network_id":  h.networkValue(network.ID),
		"peer":        router.Peer,
		"peer_groups": h.groupValues(router.PeerGroups),
		"metric":      router.Metric,
		"masquerade":  router.Masquerade,
		"enabled":     router.Enabled,
	})
}
