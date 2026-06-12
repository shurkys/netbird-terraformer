package resources

import "fmt"

// DNSSettingsResponse is the API response wrapper for DNS settings.
type DNSSettingsResponse struct {
	Items DNSSettings `json:"items"`
}

// DNSSettings represents account DNS settings.
type DNSSettings struct {
	DisabledManagementGroups []string `json:"disabled_management_groups"`
}

func (h *DNSHandler) fetchDNSSettings() (*DNSSettings, error) {
	var response DNSSettingsResponse
	if err := h.service.Get("/api/dns/settings", &response); err != nil {
		return nil, fmt.Errorf("failed to fetch DNS settings: %w", err)
	}
	return &response.Items, nil
}

func (h *DNSHandler) generateDNSSettingsResource(settings *DNSSettings) {
	attributes := map[string]any{
		"disabled_management_groups": h.groupValues(settings.DisabledManagementGroups),
	}

	h.terraformWriter.AddResourceWithoutImport("dns_settings", "main", attributes)
	h.terraformWriter.QueueCommentedImport(
		"dns_settings",
		"main",
		"main",
		"netbird_dns_settings import is commented out because provider 0.0.9 writes import state to missing schema attribute \"id\".",
	)
}
