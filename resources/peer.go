package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

type peer struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	SSHEnabled                  bool   `json:"ssh_enabled"`
	LoginExpirationEnabled      bool   `json:"login_expiration_enabled"`
	InactivityExpirationEnabled bool   `json:"inactivity_expiration_enabled"`
	ApprovalRequired            bool   `json:"approval_required"`
}

func (h *ExtendedHandler) importPeers() error {
	fmt.Printf("Importing peers...\n")
	var peers []peer
	if err := h.service.Get("/api/peers", &peers); err != nil {
		return fmt.Errorf("failed to fetch peers: %w", err)
	}

	for _, peer := range peers {
		resourceName := lib.SanitizeResourceName(peer.Name)
		if resourceName == "" {
			resourceName = fmt.Sprintf("peer_%s", peer.ID)
		}

		h.terraform.AddResource("peer", resourceName, map[string]any{
			"id":                            peer.ID,
			"name":                          peer.Name,
			"ssh_enabled":                   peer.SSHEnabled,
			"login_expiration_enabled":      peer.LoginExpirationEnabled,
			"inactivity_expiration_enabled": peer.InactivityExpirationEnabled,
			"approval_required":             peer.ApprovalRequired,
		})
	}

	fmt.Printf("Imported %d peers\n", len(peers))
	return nil
}
