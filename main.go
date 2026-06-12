package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"netbird-terraformer/lib"
	"netbird-terraformer/resources"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		showHelp()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "--debug-auth" {
		debugAuth()
		return
	}

	// Get configuration
	config := getConfig()

	// Create output directory
	outputDir, cliInclude, cliExclude, err := parseImportArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if cliInclude != "" || cliExclude != "" {
		include := os.Getenv("NB_IMPORT_RESOURCES")
		exclude := os.Getenv("NB_EXCLUDE_RESOURCES")
		if cliInclude != "" {
			include = cliInclude
		}
		if cliExclude != "" {
			exclude = cliExclude
		}

		config.ImportResources, err = parseResourceSelection(include, exclude)
		if err != nil {
			log.Fatalf("Invalid resource selection: %v", err)
		}
	}

	selectedResourceTypes := config.SelectedResourceTypes()
	if len(selectedResourceTypes) == 0 {
		log.Fatal("No resources selected for import")
	}

	fmt.Printf("NetBird Terraform Importer\n")
	fmt.Printf("Server URL: %s\n", config.ServerURL)
	fmt.Printf("Output Directory: %s\n", outputDir)
	fmt.Printf("Selected Resources: %s\n", strings.Join(selectedResourceTypes, ", "))
	fmt.Printf("Starting import...\n\n")

	// Create service and terraform generator
	service := NewNetBirdService(config.ServerURL, config.APIToken, config.Debug)
	terraformGen := lib.NewTerraformGenerator(outputDir, &lib.Config{
		ServerURL:     config.ServerURL,
		APIToken:      config.APIToken,
		TenantAccount: config.TenantAccount,
		Debug:         config.Debug,
		AutoImport:    config.AutoImport,
	})

	// Initialize resource handlers
	groupsHandler := resources.NewGroupsHandler(service, terraformGen)
	usersHandler := resources.NewUsersHandler(service, terraformGen)
	policiesHandler := resources.NewPoliciesHandler(service, terraformGen)
	routesHandler := resources.NewRoutesHandler(service, terraformGen)
	dnsHandler := resources.NewDNSHandler(service, terraformGen)
	extendedHandler := resources.NewExtendedHandler(service, terraformGen)

	groupMapping := make(map[string]string)
	dnsUsesGroups := config.ShouldImport("dns_zone") || config.ShouldImport("nameserver_group") || config.ShouldImport("dns_settings")
	extendedUsesGroups := config.ShouldImport("account_settings") ||
		config.ShouldImport("network_resource") ||
		config.ShouldImport("network_router") ||
		config.ShouldImport("setup_key")

	// Import groups first when requested so other selected resources can reference them.
	if config.ShouldImport("group") {
		err = groupsHandler.ImportAndGenerate()
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
		groupMapping = groupsHandler.GetResourceMapping()
	} else if config.ShouldImport("user") || config.ShouldImport("policy") || dnsUsesGroups || extendedUsesGroups {
		groupMapping, err = fetchGroupMapping(service)
		if err != nil {
			fmt.Printf("Warning: failed to fetch groups for resource mapping: %v\n", err)
		}
	}

	// Set group mapping for resources that need it. If groups were not selected,
	// handlers will fall back to literal group IDs instead of netbird_group references.
	usersHandler.SetGroupMapping(groupMapping)
	policiesHandler.SetGroupMapping(groupMapping)
	policiesHandler.SetUseGroupReferences(config.ShouldImport("group"))
	routesHandler.SetUseGroupReferences(config.ShouldImport("group"))
	dnsHandler.SetGroupMapping(groupMapping)
	dnsHandler.SetUseGroupReferences(config.ShouldImport("group"))
	dnsHandler.SetSelectedResources(
		config.ShouldImport("dns_zone"),
		config.ShouldImport("dns_record"),
		config.ShouldImport("nameserver_group"),
		config.ShouldImport("dns_settings"),
	)
	extendedHandler.SetGroupMapping(groupMapping)
	extendedHandler.SetUseGroupReferences(config.ShouldImport("group"))
	extendedHandler.SetSelectedResources(config.ImportResources)

	// Import other selected resources
	resourceHandlers := make([]lib.ResourceHandler, 0)
	if config.ShouldImport("user") {
		resourceHandlers = append(resourceHandlers, usersHandler)
	}
	if config.ShouldImport("policy") {
		resourceHandlers = append(resourceHandlers, policiesHandler)
	}
	if config.ShouldImport("route") {
		resourceHandlers = append(resourceHandlers, routesHandler)
	}
	if config.ShouldImport("dns_zone") || config.ShouldImport("dns_record") || config.ShouldImport("nameserver_group") || config.ShouldImport("dns_settings") {
		resourceHandlers = append(resourceHandlers, dnsHandler)
	}
	if hasExtendedResources(config) {
		resourceHandlers = append(resourceHandlers, extendedHandler)
	}

	for _, handler := range resourceHandlers {
		err = handler.ImportAndGenerate()
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Generate files and scripts
	err = generateTerraformFiles(terraformGen, outputDir)
	if err != nil {
		log.Fatalf("Failed to generate Terraform files: %v", err)
	}

	err = terraformGen.GenerateImportScript()
	if err != nil {
		log.Fatalf("Failed to generate import script: %v", err)
	}

	// Handle imports
	if config.AutoImport {
		err = runTerraformImports(terraformGen, outputDir)
		if err != nil {
			log.Fatalf("Failed to run terraform imports: %v", err)
		}
	} else {
		fmt.Printf("\nAuto-import disabled. You can manually run terraform imports later.\n")
	}

	fmt.Printf("\nImport completed successfully!\n")
	fmt.Printf("Generated files in: %s\n", outputDir)
	fmt.Printf("\nFiles generated:\n")
	fmt.Printf("  - Terraform configuration files (*.tf)\n")
	fmt.Printf("  - import.sh (terraform import commands)\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. cd %s\n", outputDir)
	if config.AutoImport {
		fmt.Printf("  2. terraform plan\n")
		fmt.Printf("  3. Review and modify the configuration as needed\n")
		fmt.Printf("\nNote: All resources have been automatically imported into Terraform state!\n")
	} else {
		fmt.Printf("  2. Run ./import.sh (or manually run terraform import commands)\n")
		fmt.Printf("  3. terraform plan\n")
		fmt.Printf("  4. Review and modify the configuration as needed\n")
	}
}

func hasExtendedResources(config *Config) bool {
	for _, resourceType := range []string{
		"account_settings",
		"identity_provider",
		"network",
		"network_resource",
		"network_router",
		"peer",
		"posture_check",
		"reverse_proxy_domain",
		"reverse_proxy_service",
		"scim",
		"setup_key",
		"token",
	} {
		if config.ShouldImport(resourceType) {
			return true
		}
	}
	return false
}

func parseImportArgs(args []string) (string, string, string, error) {
	outputDir := "generated"
	include := ""
	exclude := ""
	outputDirSet := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case strings.HasPrefix(arg, "--resources="):
			include = strings.TrimPrefix(arg, "--resources=")
		case arg == "--resources":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--resources requires a comma-separated value")
			}
			i++
			include = args[i]
		case strings.HasPrefix(arg, "--exclude-resources="):
			exclude = strings.TrimPrefix(arg, "--exclude-resources=")
		case arg == "--exclude-resources":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--exclude-resources requires a comma-separated value")
			}
			i++
			exclude = args[i]
		case strings.HasPrefix(arg, "-"):
			return "", "", "", fmt.Errorf("unknown flag %s", arg)
		default:
			if outputDirSet {
				return "", "", "", fmt.Errorf("unexpected argument %s", arg)
			}
			outputDir = arg
			outputDirSet = true
		}
	}

	return outputDir, include, exclude, nil
}

func fetchGroupMapping(service lib.NetBirdAPI) (map[string]string, error) {
	var groups []resources.Group
	if err := service.Get("/api/groups", &groups); err != nil {
		return nil, err
	}

	groupMapping := make(map[string]string, len(groups))
	for _, group := range groups {
		groupMapping[group.ID] = lib.SanitizeResourceName(group.Name)
	}

	return groupMapping, nil
}

// generateTerraformFiles groups resources by type and generates .tf files
func generateTerraformFiles(terraformGen *lib.TerraformGenerator, outputDir string) error {
	fmt.Printf("\nGenerating Terraform files...\n")

	// First generate provider.tf
	err := terraformGen.GenerateProviderFile()
	if err != nil {
		return fmt.Errorf("failed to generate provider file: %w", err)
	}

	// Group resources by type and generate files
	resources := terraformGen.GetResources()
	resourcesByType := make(map[string][]lib.TerraformResource)
	for _, resource := range resources {
		resourcesByType[resource.Type] = append(resourcesByType[resource.Type], resource)
	}

	// Generate a file for each resource type
	for resourceType, resources := range resourcesByType {
		fmt.Printf("Generating %s.tf with %d resources...\n", resourceType, len(resources))
		err := terraformGen.WriteResourceFile(resourceType, resources)
		if err != nil {
			return fmt.Errorf("failed to generate %s resources: %w", resourceType, err)
		}
	}

	fmt.Printf("Terraform files generated successfully\n")
	return nil
}

// runTerraformImports executes terraform init and import commands
func runTerraformImports(terraformGen *lib.TerraformGenerator, outputDir string) error {
	importCommands := terraformGen.GetImportCommands()
	if len(importCommands) == 0 {
		fmt.Printf("No terraform imports to run\n")
		return nil
	}

	fmt.Printf("\nRunning terraform imports...\n")

	fmt.Printf("Running terraform init...\n")
	err := lib.TerraformInit(outputDir)
	if err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	successCount := 0
	for _, cmd := range importCommands {
		if cmd.Commented {
			if cmd.Comment != "" {
				fmt.Printf("Skipping %s: %s\n", cmd.ResourceAddress, cmd.Comment)
			} else {
				fmt.Printf("Skipping %s: import command is commented out\n", cmd.ResourceAddress)
			}
			continue
		}
		fmt.Printf("Importing %s...\n", cmd.ResourceAddress)
		err := lib.TerraformImport(outputDir, cmd.ResourceAddress, cmd.ResourceID)
		if err != nil {
			fmt.Printf("  Warning: terraform import failed for %s: %v\n", cmd.ResourceAddress, err)
		} else {
			fmt.Printf("  Successfully imported %s\n", cmd.ResourceAddress)
			successCount++
		}
	}

	fmt.Printf("\nTerraform import completed: %d/%d successful\n", successCount, len(importCommands))
	return nil
}

func showHelp() {
	fmt.Println("NetBird terraformer Terraform Importer")
	fmt.Println("=====================================")
	fmt.Println("")
	fmt.Println("Usage: ./netbird-importer [output-directory]")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  NB_PAT                - Your NetBird Personal Access Token (required)")
	fmt.Println("  NB_MANAGEMENT_URL     - NetBird Management API URL (optional)")
	fmt.Println("                          Defaults to https://api.netbird.io")
	fmt.Println("  NB_ACCOUNT            - Account ID to impersonate in Terraform provider (optional)")
	fmt.Println("  DEBUG                 - Enable debug output (optional, set to 'true')")
	fmt.Println("  AUTO_IMPORT           - Auto-run terraform import (optional, set to 'false' to disable)")
	fmt.Println("  NB_IMPORT_RESOURCES   - Comma-separated resources to import")
	fmt.Println("                          Defaults to all supported resources")
	fmt.Println("  NB_EXCLUDE_RESOURCES  - Comma-separated resources to skip")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Import to default 'generated' directory")
	fmt.Println("  export NB_PAT=\"your-personal-access-token\"")
	fmt.Println("  ./netbird-importer")
	fmt.Println("")
	fmt.Println("  # Import to custom directory with custom server")
	fmt.Println("  export NB_PAT=\"your-personal-access-token\"")
	fmt.Println("  export NB_MANAGEMENT_URL=\"https://netbird.example.com\"")
	fmt.Println("  ./netbird-importer my-terraform-config")
	fmt.Println("")
	fmt.Println("  # Import only routes and policies")
	fmt.Println("  ./netbird-importer --resources routes,policies")
	fmt.Println("")
	fmt.Println("  # Import everything except users and groups")
	fmt.Println("  ./netbird-importer --exclude-resources users,groups")
	fmt.Println("")
	fmt.Println("Resource types imported:")
	for _, resourceType := range supportedResourceTypes() {
		fmt.Printf("  - %s\n", resourceType)
	}
	fmt.Println("")
	fmt.Println("Note: Secrets that cannot be read back from the API are generated as placeholders.")
	fmt.Println("")
	fmt.Println("Debug commands:")
	fmt.Println("  ./netbird-importer --debug-auth   # Test authentication")
}

func debugAuth() {
	fmt.Println("=== NetBird Authentication Debug ===")

	pat := os.Getenv("NB_PAT")
	managementURL := os.Getenv("NB_MANAGEMENT_URL")

	if pat == "" {
		fmt.Println("ERROR: NB_PAT is not set")
		return
	}

	if managementURL == "" {
		managementURL = "https://api.netbird.io"
		fmt.Printf("INFO: Using default management URL: %s\n", managementURL)
	} else {
		fmt.Printf("INFO: Using custom management URL: %s\n", managementURL)
	}

	fmt.Printf("INFO: Token length: %d characters\n", len(pat))
	if len(pat) >= 10 {
		fmt.Printf("INFO: Token starts with: %s...\n", pat[:10])
	}

	if len(pat) < 20 {
		fmt.Println("WARNING: Token seems unusually short")
	}

	fmt.Println("\n=== Testing API Connection ===")
	service := NewNetBirdService(managementURL, pat, true)

	var groups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := service.Get("/api/groups", &groups)
	if err != nil {
		fmt.Printf("ERROR: API test failed: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: API test successful! Found %d groups\n", len(groups))
	}
}
