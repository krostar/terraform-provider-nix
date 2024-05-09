package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	nixcli "github.com/krostar/terraform-provider-nix/internal/nix/cli"
)

// New creates a new provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &nixProvider{version: version} }
}

type (
	nixProvider      struct{ version string }
	nixProviderModel struct{}
)

// Metadata implements provider.Provider for terraform plugin framework.
func (p *nixProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nix"
	resp.Version = p.version
}

// Schema implements provider.Provider for terraform plugin framework.
func (*nixProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Interact with nix."}
}

// Configure implements provider.Provider for terraform plugin framework.
func (*nixProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config nixProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nixCLI := nixcli.New()

	resp.DataSourceData = nixCLI
	resp.ResourceData = nixCLI
}

// Resources implements provider.Provider for terraform plugin framework.
func (*nixProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newResourceBuild, newResourceCopyStorePath}
}

// DataSources implements provider.Provider for terraform plugin framework.
func (*nixProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newDataSourceStorePath}
}
