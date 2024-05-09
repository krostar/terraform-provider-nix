package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/krostar/terraform-provider-nix/internal/nix/shell"
)

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &nixProvider{version: version} }
}

type (
	nixProvider struct {
		version string
	}

	nixProviderModel struct{}
)

func (p *nixProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nix"
	resp.Version = p.version
}

func (p *nixProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Interact with nix."}
}

func (p *nixProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config nixProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	n := nixshell.New()

	resp.DataSourceData = n
	resp.ResourceData = n
}

func (p *nixProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{newResourceBuild, newResourceCopyStorePath}
}

func (p *nixProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newDataSourceStorePath}
}
