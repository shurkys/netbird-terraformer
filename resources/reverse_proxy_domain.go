package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

type reverseProxyDomain struct {
	ID            string `json:"id"`
	Domain        string `json:"domain"`
	TargetCluster string `json:"target_cluster"`
}

func (h *ExtendedHandler) importReverseProxyDomains() error {
	fmt.Printf("Importing reverse proxy domains...\n")
	var domains []reverseProxyDomain
	if err := h.service.Get("/api/reverse-proxies/domains", &domains); err != nil {
		return fmt.Errorf("failed to fetch reverse proxy domains: %w", err)
	}

	for _, domain := range domains {
		resourceName := lib.SanitizeResourceName(domain.Domain)
		if resourceName == "" {
			resourceName = fmt.Sprintf("reverse_proxy_domain_%s", domain.ID)
		}
		h.terraform.AddResourceWithoutImport("reverse_proxy_domain", resourceName, map[string]any{
			"domain":         domain.Domain,
			"target_cluster": domain.TargetCluster,
		})
	}

	fmt.Printf("Imported %d reverse proxy domains\n", len(domains))
	return nil
}
