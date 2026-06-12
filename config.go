package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

type Config struct {
	ServerURL       string
	APIToken        string
	TenantAccount   string
	Debug           bool
	AutoImport      bool
	ImportResources map[string]bool
}

func getConfig() *Config {
	serverURL := os.Getenv("NB_MANAGEMENT_URL")
	if serverURL == "" {
		serverURL = "https://api.netbird.io"
	}

	if len(serverURL) > 0 && serverURL[len(serverURL)-1] == '/' {
		serverURL = serverURL[:len(serverURL)-1]
	}

	apiToken := os.Getenv("NB_PAT")
	if apiToken == "" {
		log.Fatal("NB_PAT environment variable is required (NetBird Personal Access Token)")
	}
	tenantAccount := os.Getenv("NB_ACCOUNT")

	debug := os.Getenv("DEBUG") == "true"
	autoImport := os.Getenv("AUTO_IMPORT") != "false"
	importResources, err := parseResourceSelection(os.Getenv("NB_IMPORT_RESOURCES"), os.Getenv("NB_EXCLUDE_RESOURCES"))
	if err != nil {
		log.Fatalf("Invalid resource selection: %v", err)
	}

	return &Config{
		ServerURL:       serverURL,
		APIToken:        apiToken,
		TenantAccount:   tenantAccount,
		Debug:           debug,
		AutoImport:      autoImport,
		ImportResources: importResources,
	}
}

func (c *Config) ShouldImport(resourceType string) bool {
	return c.ImportResources[resourceType]
}

func (c *Config) SelectedResourceTypes() []string {
	return selectedResourceTypes(c.ImportResources)
}

func parseResourceSelection(includeCSV, excludeCSV string) (map[string]bool, error) {
	selected := make(map[string]bool)

	if strings.TrimSpace(includeCSV) == "" {
		for _, resourceType := range supportedResourceTypes() {
			selected[resourceType] = true
		}
	} else {
		included, err := parseResourceList(includeCSV)
		if err != nil {
			return nil, err
		}
		for _, resourceType := range included {
			selected[resourceType] = true
		}
	}

	if strings.TrimSpace(excludeCSV) != "" {
		excluded, err := parseResourceList(excludeCSV)
		if err != nil {
			return nil, err
		}
		for _, resourceType := range excluded {
			delete(selected, resourceType)
		}
	}

	return selected, nil
}

func parseResourceList(csv string) ([]string, error) {
	parts := strings.Split(csv, ",")
	resourceTypes := make([]string, 0, len(parts))

	for _, part := range parts {
		resourceType, err := normalizeResourceType(part)
		if err != nil {
			return nil, err
		}
		if resourceType == "all" {
			return supportedResourceTypes(), nil
		}
		resourceTypes = append(resourceTypes, resourceType)
	}

	return resourceTypes, nil
}

func normalizeResourceType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all":
		return "all", nil
	case "group", "groups":
		return "group", nil
	case "user", "users":
		return "user", nil
	case "policy", "policies":
		return "policy", nil
	case "route", "routes":
		return "route", nil
	case "dns_zone", "dns_zones", "dns-zone", "dns-zones":
		return "dns_zone", nil
	case "dns_record", "dns_records", "dns-record", "dns-records":
		return "dns_record", nil
	case "nameserver_group", "nameserver_groups", "nameserver-group", "nameserver-groups":
		return "nameserver_group", nil
	case "dns_settings", "dns-settings":
		return "dns_settings", nil
	case "account_settings", "account-settings":
		return "account_settings", nil
	case "identity_provider", "identity_providers", "identity-provider", "identity-providers":
		return "identity_provider", nil
	case "network", "networks":
		return "network", nil
	case "network_resource", "network_resources", "network-resource", "network-resources":
		return "network_resource", nil
	case "network_router", "network_routers", "network-router", "network-routers":
		return "network_router", nil
	case "peer", "peers":
		return "peer", nil
	case "posture_check", "posture_checks", "posture-check", "posture-checks":
		return "posture_check", nil
	case "reverse_proxy_domain", "reverse_proxy_domains", "reverse-proxy-domain", "reverse-proxy-domains":
		return "reverse_proxy_domain", nil
	case "reverse_proxy_service", "reverse_proxy_services", "reverse-proxy-service", "reverse-proxy-services":
		return "reverse_proxy_service", nil
	case "scim":
		return "scim", nil
	case "setup_key", "setup_keys", "setup-key", "setup-keys":
		return "setup_key", nil
	case "token", "tokens":
		return "token", nil
	default:
		return "", fmt.Errorf("unknown resource type %q (supported: %s)", value, strings.Join(supportedResourceTypes(), ", "))
	}
}

func supportedResourceTypes() []string {
	return []string{
		"account_settings",
		"dns_record",
		"dns_settings",
		"dns_zone",
		"group",
		"identity_provider",
		"nameserver_group",
		"network",
		"network_resource",
		"network_router",
		"peer",
		"policy",
		"posture_check",
		"reverse_proxy_domain",
		"reverse_proxy_service",
		"route",
		"scim",
		"setup_key",
		"token",
		"user",
	}
}

func selectedResourceTypes(selected map[string]bool) []string {
	resourceTypes := make([]string, 0, len(selected))
	for resourceType, enabled := range selected {
		if enabled {
			resourceTypes = append(resourceTypes, resourceType)
		}
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}
