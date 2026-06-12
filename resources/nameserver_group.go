package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

// NameserverGroup represents a NetBird nameserver group.
type NameserverGroup struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Description          string       `json:"description"`
	Nameservers          []Nameserver `json:"nameservers"`
	Enabled              bool         `json:"enabled"`
	Groups               []string     `json:"groups"`
	Primary              bool         `json:"primary"`
	Domains              []string     `json:"domains"`
	SearchDomainsEnabled bool         `json:"search_domains_enabled"`
}

// Nameserver represents a DNS nameserver inside a nameserver group.
type Nameserver struct {
	IP     string `json:"ip"`
	NSType string `json:"ns_type"`
	Port   int    `json:"port"`
}

func (h *DNSHandler) fetchNameserverGroups() ([]NameserverGroup, error) {
	var nameserverGroups []NameserverGroup
	if err := h.service.Get("/api/dns/nameservers", &nameserverGroups); err != nil {
		return nil, fmt.Errorf("failed to fetch nameserver groups: %w", err)
	}
	return nameserverGroups, nil
}

func (h *DNSHandler) generateNameserverGroupResource(nameserverGroup NameserverGroup) bool {
	resourceName := lib.SanitizeResourceName(nameserverGroup.Name)
	if resourceName == "" {
		resourceName = fmt.Sprintf("nameserver_group_%s", nameserverGroup.ID)
	}

	nameservers := make([]any, 0, len(nameserverGroup.Nameservers))
	for _, nameserver := range nameserverGroup.Nameservers {
		nameservers = append(nameservers, map[string]any{
			"ip":      nameserver.IP,
			"ns_type": nameserver.NSType,
			"port":    nameserver.Port,
		})
	}

	groups := h.groupValues(nameserverGroup.Groups)
	if len(groups) == 0 {
		fmt.Printf("  Skipping nameserver_group %s: Terraform provider requires at least one group\n", resourceName)
		return false
	}
	if len(nameservers) == 0 {
		fmt.Printf("  Skipping nameserver_group %s: Terraform provider requires at least one nameserver\n", resourceName)
		return false
	}

	attributes := map[string]any{
		"id":                     nameserverGroup.ID,
		"name":                   nameserverGroup.Name,
		"description":            nameserverGroup.Description,
		"nameservers":            nameservers,
		"enabled":                nameserverGroup.Enabled,
		"groups":                 groups,
		"primary":                nameserverGroup.Primary,
		"domains":                nameserverGroup.Domains,
		"search_domains_enabled": nameserverGroup.SearchDomainsEnabled,
	}

	h.terraformWriter.AddResource("nameserver_group", resourceName, attributes)
	return true
}
