package resources

import "fmt"

type personalAccessToken struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CreatedAt      string `json:"created_at"`
	ExpirationDate string `json:"expiration_date"`
}

func (h *ExtendedHandler) importTokens() error {
	fmt.Printf("Importing tokens...\n")
	var users []User
	if err := h.service.Get("/api/users", &users); err != nil {
		return fmt.Errorf("failed to fetch users for tokens: %w", err)
	}

	count := 0
	for _, user := range users {
		var tokens []personalAccessToken
		if err := h.service.Get(fmt.Sprintf("/api/users/%s/tokens", user.ID), &tokens); err != nil {
			fmt.Printf("  Warning: failed to fetch tokens for user %s: %v\n", user.ID, err)
			continue
		}

		for _, token := range tokens {
			resourceName := joinedResourceName(user.Name, token.Name)
			if resourceName == "" {
				resourceName = fmt.Sprintf("token_%s", token.ID)
			}

			h.terraform.AddResource("token", resourceName, map[string]any{
				"id":              fmt.Sprintf("%s/%s", user.ID, token.ID),
				"user_id":         user.ID,
				"name":            token.Name,
				"expiration_days": expirationDays(token.CreatedAt, token.ExpirationDate),
			})
			count++
		}
	}

	fmt.Printf("Imported %d tokens\n", count)
	return nil
}
