package lib

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TerraformGenerator handles the generation of Terraform files
type TerraformGenerator struct {
	outputDir      string
	config         *Config
	resources      []TerraformResource
	importCommands []ImportCommand
	usedNames      map[string]map[string]int
}

// NewTerraformGenerator creates a new Terraform generator
func NewTerraformGenerator(outputDir string, config *Config) *TerraformGenerator {
	return &TerraformGenerator{
		outputDir:      outputDir,
		config:         config,
		resources:      make([]TerraformResource, 0),
		importCommands: make([]ImportCommand, 0),
		usedNames:      make(map[string]map[string]int),
	}
}

// AddResource adds a resource to be generated and queues terraform import
func (tg *TerraformGenerator) AddResource(resourceType, name string, attributes map[string]any) {
	name = tg.uniqueName(resourceType, name)

	// Extract and store the ID separately
	var resourceID string
	if id, exists := attributes["id"]; exists {
		if idStr, ok := id.(string); ok {
			resourceID = idStr
		}
	}

	writableAttributes := make(map[string]any, len(attributes))
	for key, value := range attributes {
		if key == "id" && resourceType != "peer" {
			continue
		}
		writableAttributes[key] = value
	}

	tg.resources = append(tg.resources, TerraformResource{
		Type:       resourceType,
		Name:       name,
		Attributes: writableAttributes,
		IsData:     false,
		ID:         resourceID,
	})
	fmt.Printf("  Added %s resource: %s\n", resourceType, name)

	// Queue terraform import for this resource
	tg.QueueImport(resourceType, name, resourceID)
}

// AddResourceWithoutImport adds a resource that cannot be imported with terraform import.
func (tg *TerraformGenerator) AddResourceWithoutImport(resourceType, name string, attributes map[string]any) {
	name = tg.uniqueName(resourceType, name)

	tg.resources = append(tg.resources, TerraformResource{
		Type:       resourceType,
		Name:       name,
		Attributes: attributes,
		IsData:     false,
	})
	fmt.Printf("  Added %s resource: %s\n", resourceType, name)
}

// AddDataSource adds a data source to be generated
func (tg *TerraformGenerator) AddDataSource(dataType, name string, attributes map[string]any) {
	name = tg.uniqueName("data_"+dataType, name)

	tg.resources = append(tg.resources, TerraformResource{
		Type:       dataType,
		Name:       name,
		Attributes: attributes,
		IsData:     true,
	})
	fmt.Printf("  Added %s data source: %s\n", dataType, name)
}

func (tg *TerraformGenerator) uniqueName(resourceType, name string) string {
	if name == "" {
		name = "unnamed_resource"
	}

	if _, exists := tg.usedNames[resourceType]; !exists {
		tg.usedNames[resourceType] = make(map[string]int)
	}

	count := tg.usedNames[resourceType][name]
	tg.usedNames[resourceType][name] = count + 1
	if count == 0 {
		return name
	}

	return fmt.Sprintf("%s_%d", name, count+1)
}

// QueueImport queues a terraform import command
func (tg *TerraformGenerator) QueueImport(resourceType, name string, resourceID string) {
	if resourceID == "" {
		fmt.Printf("  Warning: No ID found for %s resource %s, skipping terraform import\n", resourceType, name)
		return
	}

	resourceAddress := fmt.Sprintf("netbird_%s.%s", resourceType, name)

	tg.importCommands = append(tg.importCommands, ImportCommand{
		ResourceAddress: resourceAddress,
		ResourceID:      resourceID,
	})

	fmt.Printf("  Queued terraform import for %s\n", resourceAddress)
}

// QueueCommentedImport queues a terraform import command as a commented note.
func (tg *TerraformGenerator) QueueCommentedImport(resourceType, name string, resourceID string, comment string) {
	if resourceID == "" {
		fmt.Printf("  Warning: No ID found for %s resource %s, skipping commented terraform import\n", resourceType, name)
		return
	}

	resourceAddress := fmt.Sprintf("netbird_%s.%s", resourceType, name)

	tg.importCommands = append(tg.importCommands, ImportCommand{
		ResourceAddress: resourceAddress,
		ResourceID:      resourceID,
		Commented:       true,
		Comment:         comment,
	})

	fmt.Printf("  Queued commented terraform import for %s\n", resourceAddress)
}

// GetImportCommands returns the list of import commands
func (tg *TerraformGenerator) GetImportCommands() []ImportCommand {
	return tg.importCommands
}

// GetResources returns the list of resources
func (tg *TerraformGenerator) GetResources() []TerraformResource {
	return tg.resources
}

// WriteResource writes a single resource or data source to the file
func (tg *TerraformGenerator) WriteResource(file *os.File, resource TerraformResource) error {
	// Write resource or data source block
	if resource.IsData {
		fmt.Fprintf(file, "data \"netbird_%s\" \"%s\" {\n", resource.Type, resource.Name)
	} else {
		fmt.Fprintf(file, "resource \"netbird_%s\" \"%s\" {\n", resource.Type, resource.Name)
	}

	// Write attributes
	for key, value := range resource.Attributes {
		err := tg.writeAttribute(file, key, value, 1)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(file, "}\n")
	return nil
}

// WriteResourceFile writes resources to a specific file
func (tg *TerraformGenerator) WriteResourceFile(resourceType string, resources []TerraformResource) error {
	filename := filepath.Join(tg.outputDir, fmt.Sprintf("%s.tf", resourceType))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write file header
	fmt.Fprintf(file, "# NetBird %s resources\n# Generated by NetBird Terraformer\n\n", resourceType)

	// Write each resource
	for _, resource := range resources {
		err := tg.WriteResource(file, resource)
		if err != nil {
			return err
		}
		fmt.Fprintf(file, "\n")
	}

	return nil
}

// writeAttribute writes an attribute to the file with proper formatting
func (tg *TerraformGenerator) writeAttribute(file *os.File, key string, value any, indent int) error {
	// Skip read-only fields in Terraform
	if key == "network_type" || key == "peers" {
		return nil
	}

	indentStr := strings.Repeat("  ", indent)

	switch v := value.(type) {
	case string:
		if v != "" || isRequiredScalarAttribute(key) {
			if strings.HasPrefix(v, "netbird_") {
				fmt.Fprintf(file, "%s%s = %s\n", indentStr, key, v)
			} else {
				fmt.Fprintf(file, "%s%s = \"%s\"\n", indentStr, key, EscapeString(v))
			}
		}
	case bool:
		fmt.Fprintf(file, "%s%s = %t\n", indentStr, key, v)
	case int, int64, float64:
		fmt.Fprintf(file, "%s%s = %v\n", indentStr, key, v)
	case []any:
		if len(v) > 0 {
			// Check if this is a list of maps (like rules)
			if len(v) > 0 {
				if _, isMap := v[0].(map[string]any); isMap {
					if isObjectListAttribute(key) {
						return tg.writeObjectListAttribute(file, key, v, indent)
					}
					// Handle as blocks (e.g., rules blocks)
					for _, item := range v {
						if itemMap, ok := item.(map[string]any); ok {
							// Use singular form for block name
							blockName := GetBlockName(key)
							fmt.Fprintf(file, "%s%s {\n", indentStr, blockName)
							for k, val := range itemMap {
								err := tg.writeAttribute(file, k, val, indent+1)
								if err != nil {
									return err
								}
							}
							fmt.Fprintf(file, "%s}\n", indentStr)
						}
					}
					return nil
				}
			}

			// Handle as regular list
			fmt.Fprintf(file, "%s%s = [\n", indentStr, key)
			for _, item := range v {
				if str, ok := item.(string); ok && str != "" {
					// Check if this is a Terraform reference (starts with netbird_)
					if strings.HasPrefix(str, "netbird_") {
						// Output without quotes for Terraform references
						fmt.Fprintf(file, "%s  %s,\n", indentStr, str)
					} else {
						fmt.Fprintf(file, "%s  \"%s\",\n", indentStr, EscapeString(str))
					}
				}
			}
			fmt.Fprintf(file, "%s]\n", indentStr)
		} else if isRequiredCollectionAttribute(key) {
			fmt.Fprintf(file, "%s%s = []\n", indentStr, key)
		}
	case []string:
		if len(v) > 0 {
			fmt.Fprintf(file, "%s%s = [\n", indentStr, key)
			for _, item := range v {
				if item != "" {
					// Check if this is a Terraform reference
					if strings.HasPrefix(item, "netbird_") {
						// Output without quotes for Terraform references
						fmt.Fprintf(file, "%s  %s,\n", indentStr, item)
					} else {
						fmt.Fprintf(file, "%s  \"%s\",\n", indentStr, EscapeString(item))
					}
				}
			}
			fmt.Fprintf(file, "%s]\n", indentStr)
		} else if isRequiredCollectionAttribute(key) {
			fmt.Fprintf(file, "%s%s = []\n", indentStr, key)
		}
	case []map[string]any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, item)
		}
		if len(items) > 0 || isRequiredCollectionAttribute(key) {
			return tg.writeObjectListAttribute(file, key, items, indent)
		}
	case map[string]any:
		if isObjectAttribute(key) {
			return tg.writeObjectAttribute(file, key, v, indent)
		}
		fmt.Fprintf(file, "%s%s {\n", indentStr, key)
		for k, val := range v {
			tg.writeAttribute(file, k, val, indent+1)
		}
		fmt.Fprintf(file, "%s}\n", indentStr)
	}

	return nil
}

func isRequiredScalarAttribute(key string) bool {
	switch key {
	case "address", "client_id", "client_secret", "content", "domain", "expiration_days", "id", "issuer", "name", "network_id", "provider_name", "target_cluster", "type", "user_id", "zone_id":
		return true
	default:
		return false
	}
}

func isRequiredCollectionAttribute(key string) bool {
	switch key {
	case "disabled_management_groups", "distribution_groups", "groups", "nameservers", "targets":
		return true
	default:
		return false
	}
}

func isObjectListAttribute(key string) bool {
	switch key {
	case "nameservers", "port_ranges", "targets":
		return true
	default:
		return false
	}
}

func isObjectAttribute(key string) bool {
	switch key {
	case "auth", "bearer_auth", "destination_resource", "link_auth", "password_auth", "pin_auth", "source_resource":
		return true
	default:
		return false
	}
}

func (tg *TerraformGenerator) writeObjectListAttribute(file *os.File, key string, value []any, indent int) error {
	indentStr := strings.Repeat("  ", indent)

	fmt.Fprintf(file, "%s%s = [\n", indentStr, key)
	for _, item := range value {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		fmt.Fprintf(file, "%s  {\n", indentStr)
		for k, val := range itemMap {
			if err := tg.writeAttribute(file, k, val, indent+2); err != nil {
				return err
			}
		}
		fmt.Fprintf(file, "%s  },\n", indentStr)
	}
	fmt.Fprintf(file, "%s]\n", indentStr)

	return nil
}

func (tg *TerraformGenerator) writeObjectAttribute(file *os.File, key string, value map[string]any, indent int) error {
	indentStr := strings.Repeat("  ", indent)

	fmt.Fprintf(file, "%s%s = {\n", indentStr, key)
	for k, val := range value {
		if err := tg.writeAttribute(file, k, val, indent+1); err != nil {
			return err
		}
	}
	fmt.Fprintf(file, "%s}\n", indentStr)

	return nil
}

// GenerateProviderFile generates the provider.tf file
func (tg *TerraformGenerator) GenerateProviderFile() error {
	// Create output directory
	err := os.MkdirAll(tg.outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := filepath.Join(tg.outputDir, "provider.tf")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	tenantAccountConfig := ""
	if tg.config.TenantAccount != "" {
		tenantAccountConfig = fmt.Sprintf("\n  tenant_account = %q", tg.config.TenantAccount)
	}

	providerConfig := fmt.Sprintf(`# NetBird Terraform Provider Configuration
# Generated by NetBird Terraformer

terraform {
  required_providers {
    netbird = {
      source  = "netbirdio/netbird"
      version = "~> 0.0.9"
    }
  }
}

provider "netbird" {
  management_url = "%s"%s
}
`, tg.config.ServerURL, tenantAccountConfig)

	fmt.Fprint(file, providerConfig)
	return nil
}

// GenerateImportScript generates a script with all terraform import commands
func (tg *TerraformGenerator) GenerateImportScript() error {
	if len(tg.importCommands) == 0 {
		return nil
	}

	scriptPath := filepath.Join(tg.outputDir, "import.sh")
	file, err := os.Create(scriptPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		return err
	}

	fmt.Fprintln(file, "#!/bin/bash")
	fmt.Fprintln(file, "# NetBird Terraform Import Script")
	fmt.Fprintln(file, "# Generated by NetBird Terraformer")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "set -e")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "echo \"Running terraform init...\"")
	fmt.Fprintln(file, "terraform init")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "echo \"Running terraform imports...\"")

	for _, cmd := range tg.importCommands {
		if cmd.Commented {
			if cmd.Comment != "" {
				fmt.Fprintf(file, "# %s\n", cmd.Comment)
			}
			fmt.Fprintf(file, "# echo \"Importing %s...\"\n", cmd.ResourceAddress)
			fmt.Fprintf(file, "# terraform import \"%s\" \"%s\"\n", cmd.ResourceAddress, cmd.ResourceID)
			fmt.Fprintln(file, "")
			continue
		}
		fmt.Fprintf(file, "echo \"Importing %s...\"\n", cmd.ResourceAddress)
		fmt.Fprintf(file, "terraform import \"%s\" \"%s\"\n", cmd.ResourceAddress, cmd.ResourceID)
		fmt.Fprintln(file, "")
	}

	fmt.Fprintln(file, "echo \"All imports completed!\"")

	return nil
}

// TerraformInit runs terraform init in the specified directory
func TerraformInit(folderPath string) error {
	cmd := exec.Command("terraform", "init")
	cmd.Dir = folderPath

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// TerraformImport runs terraform import for a specific resource
func TerraformImport(folderPath string, resourceAddress string, resourceID string) error {
	cmd := exec.Command("terraform", "import", resourceAddress, resourceID)
	cmd.Dir = folderPath

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
