package resources

import (
	"fmt"
	"math"
	"strings"
	"time"

	"netbird-terraformer/lib"
)

// ExtendedHandler imports NetBird resources that do not have dedicated handlers.
type ExtendedHandler struct {
	service        lib.NetBirdAPI
	terraform      lib.TerraformWriter
	groupMapping   map[string]string
	networkMapping map[string]string
	useGroupRefs   bool
	useNetworkRefs bool
	selected       map[string]bool
}

func NewExtendedHandler(service lib.NetBirdAPI, terraformWriter lib.TerraformWriter) *ExtendedHandler {
	return &ExtendedHandler{
		service:        service,
		terraform:      terraformWriter,
		groupMapping:   make(map[string]string),
		networkMapping: make(map[string]string),
		useGroupRefs:   true,
		useNetworkRefs: true,
		selected:       make(map[string]bool),
	}
}

func (h *ExtendedHandler) SetGroupMapping(groupMapping map[string]string) {
	h.groupMapping = groupMapping
}

func (h *ExtendedHandler) SetUseGroupReferences(useGroupReferences bool) {
	h.useGroupRefs = useGroupReferences
}

func (h *ExtendedHandler) SetSelectedResources(selected map[string]bool) {
	h.selected = selected
	h.useNetworkRefs = selected["network"]
}

func (h *ExtendedHandler) ImportAndGenerate() error {
	importers := []struct {
		enabled bool
		run     func() error
	}{
		{h.selected["account_settings"], h.importAccountSettings},
		{h.selected["identity_provider"], h.importIdentityProviders},
		{h.selected["network"] || h.selected["network_resource"] || h.selected["network_router"], h.importNetworks},
		{h.selected["peer"], h.importPeers},
		{h.selected["posture_check"], h.importPostureChecks},
		{h.selected["reverse_proxy_domain"], h.importReverseProxyDomains},
		{h.selected["reverse_proxy_service"], h.importReverseProxyServices},
		{h.selected["scim"], h.importSCIM},
		{h.selected["setup_key"], h.importSetupKeys},
		{h.selected["token"], h.importTokens},
	}

	for _, importer := range importers {
		if importer.enabled {
			if err := importer.run(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *ExtendedHandler) GetResourceMapping() map[string]string {
	return make(map[string]string)
}

func (h *ExtendedHandler) GetResourceType() string {
	return "extended"
}

func (h *ExtendedHandler) groupValues(groupIDs []string) []string {
	groups := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		if h.useGroupRefs {
			if groupResourceName, exists := h.groupMapping[groupID]; exists {
				groups = append(groups, lib.CreateTerraformReference("group", groupResourceName))
				continue
			}
		}
		groups = append(groups, groupID)
	}
	return groups
}

func (h *ExtendedHandler) groupInfoValues(groups []GroupInfo) []string {
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	return h.groupValues(groupIDs)
}

func filterAttributes(source map[string]any, allowed []string) map[string]any {
	result := make(map[string]any)
	for _, key := range allowed {
		if value, exists := source[key]; exists && value != nil {
			result[key] = value
		}
	}
	return result
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func terraformPostureCheckName(apiName string) string {
	if apiName == "nb_version_check" {
		return "netbird_version_check"
	}
	return apiName
}

func expirationDays(createdAt, expirationDate string) int {
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return 0
	}
	expires, err := time.Parse(time.RFC3339Nano, expirationDate)
	if err != nil {
		return 0
	}
	days := math.Ceil(expires.Sub(created).Hours() / 24)
	if days < 0 {
		return 0
	}
	return int(days)
}

func joinedResourceName(parts ...string) string {
	return lib.SanitizeResourceName(strings.Join(parts, "_"))
}
