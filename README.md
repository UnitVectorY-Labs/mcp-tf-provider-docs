[![License](https://img.shields.io/badge/license-MIT-blue)](https://opensource.org/licenses/MIT) [![Work In Progress](https://img.shields.io/badge/Status-Work%20In%20Progress-yellow)](https://guide.unitvectorylabs.com/bestpractices/status/#work-in-progress)

# mcp-tf-provider-docs

A configurable MCP server that indexes and serves Terraform/Tofu provider documentation from a local Git repo to power accurate, context-aware code generation.

## Purpose

`mcp-tf-provider-docs` is designed to provide an MCP server that can provide access to Terraform resources. This requires checking out the corresponding Terraform Git repository such as [hashicorp/terraform-provider-google](https://github.com/hashicorp/terraform-provider-google) which will be used as an example in this documentation.  However, this MCP server is generic and can be configured to work with other Terraform providers as well.  The MCP tool provided allows your Agent to look up the Terraform documentation which is useful in code generation as providers such as GCP regurally add and update their APIs and therefore it is common for attributes and features to not exist in the training data and therefore providing the most recent documentation can improve code generation.

## Releases

All official versions of **mcp-tf-provider-docs** are published on [GitHub Releases](https://github.com/UnitVectorY-Labs/mcp-tf-provider-docs/releases). Since this MCP server is written in Go, each release provides pre-compiled executables for macOS, Linux, and Windows—ready to download and run.

## Configuration

The server is configured using environment variables and YAML files.

### Environment Variables
- `TF_CONFIG`: The path to the configuration YAML file. (required)

### YAML Configuration File

The configuration file is used to specify the configuration for the MCP server allowing it to be customized for different Terraform providers.

The following attributes can be specified in the file:

- `docs_path`: The path to the root directory where documentation files live, typically a Git repository of the Terraform provider specifying the location of the documents folder, typically `/website/docs`. (required)
- `match_pattern`: A regex pattern to extract fully-qualified resource or data source names from the documentation files. This pattern should match the prefix used by the Terraform provider. For example, for the Google Cloud Platform provider, the pattern would be `\\bgoogle_[a-z0-9_]+\\b`. (required)
- `tool_description`: A description used by the MCP tool to provide context for the documentation, which can be helpful for users to understand what the tool does. (required)

The configuration file is structured as follows:

```yaml
# Path to the root directory where documentation files live, a Git repository to the Terraform provider
docs_path: "/path-to/terraform-provider-google/website/docs"

# Regex to extract fully-qualified resource or data source names, this looks for the prefix that is used by the Terraform provider
match_pattern: "\\bgoogle_[a-z0-9_]+\\b"

# Description used by the MCP tool to provide context for the documentation
tool_description: "Lookup Terraform documentation for Google Cloud Platform based on the provider name."
```

## Limitations

The index that is built by this application is quite rudimentary as it is simply performing a regex match on the documentation files to find the resource names that are present in each file and returning the entire content of all matching files.  This is driven by a limitation of the Terraform documentation not having a consistent structure or naming conveention between providers or even withing the same provider.
