package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

// Route represents a NetBird route
type Route struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	NetworkID   string   `json:"network_id"`
	Network     string   `json:"network"`
	NetworkType string   `json:"network_type"`
	Peer        string   `json:"peer"`
	PeerGroups  []string `json:"peer_groups"`
	Metric      int      `json:"metric"`
	Masquerade  bool     `json:"masquerade"`
	Enabled     bool     `json:"enabled"`
	Groups      []string `json:"groups"`
	KeepRoute   bool     `json:"keep_route"`
}

// RouteGroup represents a NetBird group (minimal struct for group fetching)
type RouteGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Handler implements ResourceHandler for routes
type RoutesHandler struct {
	service            lib.NetBirdAPI
	terraformWriter    lib.TerraformWriter
	useGroupReferences bool
}

// NewHandler creates a new routes handler
func NewRoutesHandler(service lib.NetBirdAPI, terraformWriter lib.TerraformWriter) *RoutesHandler {
	return &RoutesHandler{
		service:            service,
		terraformWriter:    terraformWriter,
		useGroupReferences: true,
	}
}

// SetUseGroupReferences controls whether route groups reference generated group resources.
func (h *RoutesHandler) SetUseGroupReferences(useGroupReferences bool) {
	h.useGroupReferences = useGroupReferences
}

// ImportAndGenerate imports routes from NetBird and generates Terraform resources
func (h *RoutesHandler) ImportAndGenerate() error {
	fmt.Printf("Importing routes...\n")

	groupIDToResourceName := make(map[string]string)
	if h.useGroupReferences {
		var groups []RouteGroup
		err := h.service.Get("/api/groups", &groups)
		if err != nil {
			return fmt.Errorf("failed to fetch groups for route mapping: %w", err)
		}

		for _, group := range groups {
			resourceName := lib.SanitizeResourceName(group.Name)
			groupIDToResourceName[group.ID] = resourceName
		}
	}

	var routes []Route
	err := h.service.Get("/api/routes", &routes)
	if err != nil {
		return fmt.Errorf("failed to fetch routes: %w", err)
	}

	count := 0
	for _, route := range routes {
		if h.generateRouteResource(route, groupIDToResourceName) {
			count++
		}
	}

	fmt.Printf("Imported %d routes\n", count)
	return nil
}

// GetResourceMapping returns an empty mapping since routes don't need to be referenced
func (h *RoutesHandler) GetResourceMapping() map[string]string {
	return make(map[string]string)
}

// GetResourceType returns the resource type
func (h *RoutesHandler) GetResourceType() string {
	return "route"
}

// generateRouteResource generates a Terraform resource for a route
func (h *RoutesHandler) generateRouteResource(route Route, groupIDToResourceName map[string]string) bool {
	resourceName := lib.SanitizeResourceName(route.NetworkID)
	if resourceName == "" {
		resourceName = lib.SanitizeResourceName(route.Network)
	}
	if resourceName == "" {
		resourceName = fmt.Sprintf("route_%s", route.ID)
	}

	groupRefs := routeGroupValues(route.Groups, h.useGroupReferences, groupIDToResourceName)
	peerGroupRefs := routeGroupValues(route.PeerGroups, h.useGroupReferences, groupIDToResourceName)
	if len(groupRefs) == 0 {
		fmt.Printf("  Skipping route %s: Terraform provider requires at least one group\n", resourceName)
		return false
	}

	attributes := map[string]any{
		"id":          route.ID,
		"description": route.Description,
		"network_id":  route.NetworkID,
		"network":     route.Network,
		"peer":        route.Peer,
		"peer_groups": peerGroupRefs,
		"metric":      route.Metric,
		"masquerade":  route.Masquerade,
		"enabled":     route.Enabled,
		"groups":      groupRefs,
		"keep_route":  route.KeepRoute,
	}

	h.terraformWriter.AddResource("route", resourceName, attributes)
	return true
}

func routeGroupValues(groupIDs []string, useGroupReferences bool, groupIDToResourceName map[string]string) []string {
	groupRefs := make([]string, 0)
	for _, groupID := range groupIDs {
		if useGroupReferences {
			resourceName, exists := groupIDToResourceName[groupID]
			if !exists {
				continue
			}
			groupRefs = append(groupRefs, lib.CreateTerraformReference("group", resourceName))
		} else {
			groupRefs = append(groupRefs, groupID)
		}
	}
	return groupRefs
}
