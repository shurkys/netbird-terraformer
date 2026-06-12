package resources

import (
	"fmt"
	"strings"

	"netbird-terraformer/lib"
)

// DNSRecord represents a DNS record within a NetBird custom DNS zone.
type DNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	ZoneID  string `json:"-"`
}

func (h *DNSHandler) fetchRecords(zoneID string) ([]DNSRecord, error) {
	var records []DNSRecord
	if err := h.service.Get(fmt.Sprintf("/api/dns/zones/%s/records", zoneID), &records); err != nil {
		return nil, fmt.Errorf("failed to fetch DNS records for zone %s: %w", zoneID, err)
	}
	return records, nil
}

func (h *DNSHandler) generateDNSRecordResource(zone DNSZone, record DNSRecord) {
	resourceName := lib.SanitizeResourceName(strings.Join([]string{zone.Domain, record.Name, record.Type}, "_"))
	if resourceName == "" {
		resourceName = fmt.Sprintf("dns_record_%s", record.ID)
	}

	zoneID := zone.ID
	if h.includeZones {
		zoneID = lib.CreateTerraformReference("dns_zone", dnsZoneResourceName(zone))
	}

	attributes := map[string]any{
		"id":      fmt.Sprintf("%s/%s", record.ZoneID, record.ID),
		"zone_id": zoneID,
		"name":    record.Name,
		"type":    record.Type,
		"content": record.Content,
		"ttl":     record.TTL,
	}

	h.terraformWriter.AddResource("dns_record", resourceName, attributes)
}
