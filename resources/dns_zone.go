package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

// DNSZone represents a NetBird custom DNS zone.
type DNSZone struct {
	ID                 string      `json:"id"`
	Name               string      `json:"name"`
	Domain             string      `json:"domain"`
	Enabled            bool        `json:"enabled"`
	EnableSearchDomain bool        `json:"enable_search_domain"`
	DistributionGroups []string    `json:"distribution_groups"`
	Records            []DNSRecord `json:"records"`
}

func (h *DNSHandler) fetchZones() ([]DNSZone, error) {
	var zones []DNSZone
	if err := h.service.Get("/api/dns/zones", &zones); err != nil {
		return nil, fmt.Errorf("failed to fetch DNS zones: %w", err)
	}
	return zones, nil
}

func (h *DNSHandler) generateDNSZoneResource(zone DNSZone) {
	resourceName := dnsZoneResourceName(zone)

	attributes := map[string]any{
		"id":                   zone.ID,
		"name":                 zone.Name,
		"domain":               zone.Domain,
		"enabled":              zone.Enabled,
		"enable_search_domain": zone.EnableSearchDomain,
		"distribution_groups":  h.groupValues(zone.DistributionGroups),
	}

	h.terraformWriter.AddResource("dns_zone", resourceName, attributes)
}

func dnsZoneResourceName(zone DNSZone) string {
	resourceName := lib.SanitizeResourceName(zone.Name)
	if resourceName == "" {
		resourceName = lib.SanitizeResourceName(zone.Domain)
	}
	if resourceName == "" {
		resourceName = fmt.Sprintf("dns_zone_%s", zone.ID)
	}
	return resourceName
}
