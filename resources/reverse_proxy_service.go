package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

func (h *ExtendedHandler) importReverseProxyServices() error {
	fmt.Printf("Importing reverse proxy services...\n")
	var services []map[string]any
	if err := h.service.Get("/api/reverse-proxies/services", &services); err != nil {
		return fmt.Errorf("failed to fetch reverse proxy services: %w", err)
	}

	count := 0
	for _, service := range services {
		id := stringValue(service["id"])
		resourceName := lib.SanitizeResourceName(stringValue(service["name"]))
		if resourceName == "" {
			resourceName = fmt.Sprintf("reverse_proxy_service_%s", id)
		}
		attrs := filterAttributes(service, []string{
			"name",
			"domain",
			"targets",
			"auth",
			"enabled",
			"pass_host_header",
			"rewrite_redirects",
		})
		if !hasItems(attrs["targets"]) {
			fmt.Printf("  Skipping reverse_proxy_service %s: Terraform provider requires at least one target\n", resourceName)
			continue
		}
		if !hasObject(attrs["auth"]) {
			fmt.Printf("  Skipping reverse_proxy_service %s: Terraform provider requires auth\n", resourceName)
			continue
		}
		h.terraform.AddResourceWithoutImport("reverse_proxy_service", resourceName, attrs)
		count++
	}

	fmt.Printf("Imported %d reverse proxy services\n", count)
	return nil
}

func hasItems(value any) bool {
	switch v := value.(type) {
	case []any:
		return len(v) > 0
	case []map[string]any:
		return len(v) > 0
	default:
		return false
	}
}

func hasObject(value any) bool {
	v, ok := value.(map[string]any)
	return ok && len(v) > 0
}
