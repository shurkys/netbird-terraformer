package resources

import (
	"fmt"

	"netbird-terraformer/lib"
)

type postureCheck struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Checks      map[string]any `json:"checks"`
}

func (h *ExtendedHandler) importPostureChecks() error {
	fmt.Printf("Importing posture checks...\n")
	var checks []postureCheck
	if err := h.service.Get("/api/posture-checks", &checks); err != nil {
		return fmt.Errorf("failed to fetch posture checks: %w", err)
	}

	for _, check := range checks {
		resourceName := lib.SanitizeResourceName(check.Name)
		if resourceName == "" {
			resourceName = fmt.Sprintf("posture_check_%s", check.ID)
		}

		attrs := map[string]any{
			"id":          check.ID,
			"name":        check.Name,
			"description": check.Description,
		}
		for key, value := range check.Checks {
			attrs[terraformPostureCheckName(key)] = value
		}

		h.terraform.AddResource("posture_check", resourceName, attrs)
	}

	fmt.Printf("Imported %d posture checks\n", len(checks))
	return nil
}
