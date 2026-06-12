# NetBird Terraformer

Generate Terraform configuration and import commands from an existing NetBird account.

`netbird-terraformer` is a small Go CLI that reads resources from the NetBird Management API, writes Terraform HCL for the official `netbirdio/netbird` provider, and produces an `import.sh` script that can bring supported resources into Terraform state.

The tool is designed for migrations: point it at a NetBird account, review the generated `.tf` files, import existing resources, then continue managing the account with Terraform.

## Features

- Generates Terraform files grouped by NetBird resource type.
- Produces `terraform import` commands for importable resources.
- Supports Docker usage with Terraform bundled in the runtime image.
- Lets you include or exclude resource families from a run.
- Uses Terraform references between generated resources when possible.
- Falls back to literal NetBird IDs when referenced resources are not generated.
- Keeps provider credentials out of generated Terraform files.
- Handles duplicate Terraform names after sanitization.

## Supported Resources

The importer currently supports these resource selectors:

```text
account_settings
dns_record
dns_settings
dns_zone
group
identity_provider
nameserver_group
network
network_resource
network_router
peer
policy
posture_check
reverse_proxy_domain
reverse_proxy_service
route
scim
setup_key
token
user
```

Plural and hyphenated aliases are accepted for most resource names. For example, `groups`, `dns-zones`, and `reverse-proxy-services` are normalized to the canonical names above.

## Requirements

- Go 1.21 or newer, if building locally.
- Terraform, if running imports locally.
- A NetBird Personal Access Token with enough permissions to read the resources you want to import.
- Network access to the NetBird Management API.

The Docker image includes Terraform, so local Terraform installation is only required when you run the generated configuration outside the container.

## Quick Start

Build the binary:

```bash
go build -o netbird-importer .
```

Export credentials:

```bash
export NB_PAT="pat_your_token_here"
export NB_MANAGEMENT_URL="https://api.netbird.io"
export AUTO_IMPORT=false
```

Generate Terraform files:

```bash
./netbird-importer generated
```

Run the generated imports and check the plan:

```bash
cd generated
./import.sh
terraform plan
```

By default, the CLI tries to run Terraform imports automatically after generation. Set `AUTO_IMPORT=false` when you only want files and the import script:

```bash
AUTO_IMPORT=false ./netbird-importer generated
```

## Docker Usage

Build the image:

```bash
docker build -t netbird-terraformer .
```

Generate files into a local directory:

```bash
mkdir -p generated

docker run --rm \
  -e NB_PAT="pat_your_token_here" \
  -e NB_MANAGEMENT_URL="https://api.netbird.io" \
  -e AUTO_IMPORT=false \
  -v "$(pwd)/generated:/work/generated" \
  netbird-terraformer generated
```

The container writes to `/work/generated`; the bind mount above keeps the generated files on your host.

To import automatically from the container, keep `AUTO_IMPORT` unset or set it to anything other than `false`:

```bash
docker run --rm \
  -e NB_PAT="pat_your_token_here" \
  -e NB_MANAGEMENT_URL="https://api.netbird.io" \
  -v "$(pwd)/generated:/work/generated" \
  netbird-terraformer generated
```

Use an empty output directory for each run when changing resource filters. The importer writes new files but does not clean stale files from previous runs.

## Configuration

Configuration is provided with environment variables and optional CLI flags.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `NB_PAT` | Yes | none | NetBird Personal Access Token. Also used by Terraform provider via environment variable. |
| `NB_MANAGEMENT_URL` | No | `https://api.netbird.io` | NetBird Management API URL. |
| `NB_ACCOUNT` | No | none | Account ID to impersonate. Written as `tenant_account` in `provider.tf` when set. |
| `NB_IMPORT_RESOURCES` | No | all resources | Comma-separated resource selectors to generate. |
| `NB_EXCLUDE_RESOURCES` | No | none | Comma-separated resource selectors to skip. Applied after includes. |
| `AUTO_IMPORT` | No | enabled | Set to `false` to generate files without running Terraform imports. |
| `DEBUG` | No | disabled | Set to `true` for verbose API request logging. |

CLI resource filters override the matching environment variables:

```bash
./netbird-importer --resources groups,policies,routes generated
./netbird-importer --exclude-resources users,tokens generated
```

Use `--help` to print the current resource list:

```bash
./netbird-importer --help
```

Use `--debug-auth` to validate token and API connectivity:

```bash
NB_PAT="pat_your_token_here" ./netbird-importer --debug-auth
```

## Resource Filtering

Generate only selected resource families:

```bash
NB_IMPORT_RESOURCES="groups,policies,routes" ./netbird-importer generated
```

Generate everything except selected resource families:

```bash
NB_EXCLUDE_RESOURCES="users,tokens" ./netbird-importer generated
```

Combine include and exclude lists:

```bash
NB_IMPORT_RESOURCES="all" \
NB_EXCLUDE_RESOURCES="users,tokens,identity_providers" \
./netbird-importer generated
```

When `group` is included, resources that reference groups are generated with Terraform references such as `netbird_group.example.id`. When `group` is excluded, the importer uses literal NetBird group IDs instead.

## Generated Output

A typical output directory looks like this:

```text
generated/
├── provider.tf
├── import.sh
├── account_settings.tf
├── dns_record.tf
├── dns_settings.tf
├── dns_zone.tf
├── group.tf
├── identity_provider.tf
├── nameserver_group.tf
├── network.tf
├── network_resource.tf
├── network_router.tf
├── peer.tf
├── policy.tf
├── posture_check.tf
├── reverse_proxy_domain.tf
├── reverse_proxy_service.tf
├── route.tf
├── scim.tf
├── setup_key.tf
├── token.tf
└── user.tf
```

Only files for selected and discovered resources are generated.

`provider.tf` contains the provider source, version constraint, `management_url`, and optional `tenant_account`. It intentionally does not write the PAT token:

```hcl
provider "netbird" {
  management_url = "https://api.netbird.io"
  tenant_account = "account-id"
}
```

Keep `NB_PAT` exported when running `terraform init`, `terraform import`, `terraform plan`, or `terraform apply`.

## Import Workflow

Recommended migration flow:

```bash
export NB_PAT="pat_your_token_here"
export NB_MANAGEMENT_URL="https://api.netbird.io"
export AUTO_IMPORT=false

./netbird-importer generated

cd generated
terraform init
./import.sh
terraform plan
```

Review the plan carefully. A clean migration usually ends with no resources to add, change, or destroy, except for resources that the provider cannot import or fields that cannot be read back from the API.

## Secrets

Some NetBird resources contain secret values that are not returned by the API after creation, such as identity provider client secrets. The importer cannot reconstruct secrets that NetBird does not expose.

When a required secret is needed for valid Terraform configuration, the generated file may contain a placeholder. Replace placeholders before running `terraform apply`.

The NetBird PAT is never written to generated Terraform files. Use `NB_PAT` for both this importer and the Terraform provider.

## Known Limitations

- `netbird_dns_settings` is generated as a managed resource, but its import command is commented out in `import.sh`. Provider version `0.0.9` declares import support through an `id` attribute that is not present in that resource schema, so `terraform import netbird_dns_settings.main main` fails with a provider state write error.
- Resources that violate provider validation are skipped with a warning. For example, a `netbird_network_resource` without any groups cannot be represented because the provider requires at least one group.
- Some singleton or API-only resources may update existing account-wide settings during `terraform apply`. Always review generated HCL before applying.
- Generated names are sanitized for Terraform and deduplicated with suffixes when needed.

## Troubleshooting

Authentication failed:

```bash
echo "$NB_PAT"
./netbird-importer --debug-auth
```

Wrong API URL:

```bash
curl -H "Authorization: Token $NB_PAT" \
  "$NB_MANAGEMENT_URL/api/groups"
```

Terraform wants to create many resources:

```bash
cd generated
./import.sh
terraform plan
```

Make sure the import script completed successfully and that you are using the same backend and workspace for both import and plan.

Generated output contains stale resources:

```bash
rm -rf generated
AUTO_IMPORT=false ./netbird-importer generated
```

The importer does not remove files that were generated by previous runs with different filters.

## Development

Build:

```bash
make build
```

Run tests:

```bash
go test ./...
```

Build release-style binaries:

```bash
make build-all
```

Clean generated artifacts:

```bash
make clean
```

## Project Status

This project is intended as a practical migration helper for NetBird accounts. Review generated configuration before applying it, especially for account-wide settings, DNS settings, identity providers, and resources containing secrets.
