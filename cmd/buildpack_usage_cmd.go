package cmd

import (
	"github.com/cloudfoundry/cli/plugin"
	"github.com/krujos/cfcurl"
)

// BuildpackUsage - the main struct to implement the plugin struct
type BuildpackUsage struct{}

// GetMetadata - return info about the plugin
func (cmd *BuildpackUsage) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "buildpack-usage",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "buildpack-usage",
				HelpText: "Show all apps using a given buildpack",
				UsageDetails: plugin.Usage{
					Usage: "cf buildpack-usage [-b buildpack]",
				},
			},
		},
	}
}

// Run - do the needful
func (cmd *BuildpackUsage) Run(cli plugin.CliConnection, args []string) {
	cfcurl.Curl(cli, "/v2/apps")
}

// Start - the entry point for the CF RPC server
func (cmd *BuildpackUsage) Start() {
	plugin.Start(cmd)
}
