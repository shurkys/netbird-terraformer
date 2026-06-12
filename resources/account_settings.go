package resources

import "fmt"

type account struct {
	ID       string         `json:"id"`
	Settings map[string]any `json:"settings"`
}

func (h *ExtendedHandler) importAccountSettings() error {
	fmt.Printf("Importing account settings...\n")
	var accounts []account
	if err := h.service.Get("/api/accounts", &accounts); err != nil {
		return fmt.Errorf("failed to fetch account settings: %w", err)
	}
	if len(accounts) == 0 {
		fmt.Printf("Imported 0 account settings\n")
		return nil
	}

	attrs := filterAttributes(accounts[0].Settings, []string{
		"auto_update_version",
		"dns_domain",
		"groups_propagation_enabled",
		"jwt_allow_groups",
		"jwt_groups_claim_name",
		"jwt_groups_enabled",
		"lazy_connection_enabled",
		"network_range",
		"network_traffic_logs_enabled",
		"network_traffic_logs_groups",
		"network_traffic_packet_counter_enabled",
		"peer_approval_enabled",
		"peer_expose_enabled",
		"peer_expose_groups",
		"peer_inactivity_expiration",
		"peer_inactivity_expiration_enabled",
		"peer_login_expiration",
		"peer_login_expiration_enabled",
		"regular_users_view_blocked",
		"routing_peer_dns_resolution_enabled",
		"user_approval_required",
	})
	attrs["id"] = accounts[0].ID
	h.terraform.AddResource("account_settings", "main", attrs)
	fmt.Printf("Imported account settings\n")
	return nil
}
