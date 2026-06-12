package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

// DNSHandler coordinates importing selected NetBird DNS resources.
type DNSHandler struct {
	service            lib.NetBirdAPI
	terraformWriter    lib.TerraformWriter
	groupMapping       map[string]string
	useGroupReferences bool
	includeZones       bool
	includeRecords     bool
	includeNameservers bool
	includeSettings    bool
}

// NewDNSHandler creates a new DNS handler.
func NewDNSHandler(service lib.NetBirdAPI, terraformWriter lib.TerraformWriter) *DNSHandler {
	return &DNSHandler{
		service:            service,
		terraformWriter:    terraformWriter,
		groupMapping:       make(map[string]string),
		useGroupReferences: true,
	}
}

// SetGroupMapping sets the group ID to resource name mapping.
func (h *DNSHandler) SetGroupMapping(groupMapping map[string]string) {
	h.groupMapping = groupMapping
}

// SetUseGroupReferences controls whether DNS resources reference generated group resources.
func (h *DNSHandler) SetUseGroupReferences(useGroupReferences bool) {
	h.useGroupReferences = useGroupReferences
}

// SetSelectedResources controls which DNS resources are imported.
func (h *DNSHandler) SetSelectedResources(includeZones, includeRecords, includeNameservers, includeSettings bool) {
	h.includeZones = includeZones
	h.includeRecords = includeRecords
	h.includeNameservers = includeNameservers
	h.includeSettings = includeSettings
}

// ImportAndGenerate imports selected DNS resources from NetBird and generates Terraform resources.
func (h *DNSHandler) ImportAndGenerate() error {
	fmt.Printf("Importing DNS resources...\n")

	if h.includeZones || h.includeRecords {
		zones, err := h.fetchZones()
		if err != nil {
			return err
		}

		if h.includeZones {
			for _, zone := range zones {
				h.generateDNSZoneResource(zone)
			}
			fmt.Printf("Imported %d DNS zones\n", len(zones))
		}

		if h.includeRecords {
			recordCount := 0
			for _, zone := range zones {
				records := zone.Records
				if records == nil {
					fetchedRecords, err := h.fetchRecords(zone.ID)
					if err != nil {
						return err
					}
					records = fetchedRecords
				}

				for _, record := range records {
					record.ZoneID = zone.ID
					h.generateDNSRecordResource(zone, record)
					recordCount++
				}
			}
			fmt.Printf("Imported %d DNS records\n", recordCount)
		}
	}

	if h.includeNameservers {
		nameserverGroups, err := h.fetchNameserverGroups()
		if err != nil {
			return err
		}

		count := 0
		for _, nameserverGroup := range nameserverGroups {
			if h.generateNameserverGroupResource(nameserverGroup) {
				count++
			}
		}
		fmt.Printf("Imported %d nameserver groups\n", count)
	}

	if h.includeSettings {
		settings, err := h.fetchDNSSettings()
		if err != nil {
			return err
		}
		h.generateDNSSettingsResource(settings)
		fmt.Printf("Imported DNS settings\n")
	}

	return nil
}

// GetResourceMapping returns an empty mapping since DNS resources do not need to be referenced by other handlers.
func (h *DNSHandler) GetResourceMapping() map[string]string {
	return make(map[string]string)
}

// GetResourceType returns the resource type group handled by this handler.
func (h *DNSHandler) GetResourceType() string {
	return "dns"
}

func (h *DNSHandler) groupValues(groupIDs []string) []string {
	groups := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		if h.useGroupReferences {
			if groupResourceName, exists := h.groupMapping[groupID]; exists {
				groups = append(groups, lib.CreateTerraformReference("group", groupResourceName))
				continue
			}
		}
		groups = append(groups, groupID)
	}
	return groups
}
