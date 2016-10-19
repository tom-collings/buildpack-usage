package command

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/krujos/cfcurl"
	"github.com/olekukonko/tablewriter"
	"github.com/xchapter7x/lo"
)

type orgSpaceInfo struct {
	OrgName   string
	SpaceName string
}

// BuildpackUsage - the main struct to implement the plugin struct
type BuildpackUsage struct {
	Input  io.Reader
	Output io.Writer
}

// New - returns a new instance of the command
func New() *BuildpackUsage {
	return &BuildpackUsage{
		Input:  os.Stdin,
		Output: os.Stdout,
	}
}

// GetMetadata - return info about the plugin
func (cmd *BuildpackUsage) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "buildpack-usage",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 9,
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
	var buildpackGUID string
	var buildpackName string
	var err error

	if args[0] != cmd.GetMetadata().Commands[0].Name {
		return
	}

	flagSet := flag.NewFlagSet("command-args", flag.ContinueOnError)

	flagSet.StringVar(&buildpackName, "b", "NOT_SET", "The requested buildpack")
	flagSet.Parse(args[1:])

	if buildpackName == "NOT_SET" {
		buildpackGUID, buildpackName, err = getBuildpackFromInput(cli, cmd.Input)
	} else {
		buildpackGUID, err = getBuildpackFromName(cli, buildpackName)
	}

	if err != nil {
		lo.G.Fatal(err)
	}

	orgSpaceMap := make(map[string]*orgSpaceInfo)

	var nextURL interface{}
	nextURL = "/v2/apps"

	table := tablewriter.NewWriter(cmd.Output)
	table.SetHeader([]string{"App Name", "Org", "Space"})

	for nextURL != nil {
		json, err := cfcurl.Curl(cli, nextURL.(string))
		if err != nil {
			lo.G.Fatal(err)
		}

		apps := json["resources"].([]interface{})
		for _, appIntf := range apps {
			app := appIntf.(map[string]interface{})
			appEntity := app["entity"].(map[string]interface{})
			appName := appEntity["name"].(string)
			appSpaceURL := appEntity["space_url"].(string)

			appBuildpackGUID := appEntity["detected_buildpack_guid"]

			if appBuildpackGUID == nil || (appBuildpackGUID.(string) != buildpackGUID) {
				continue
			}

			if orgSpaceMap[appSpaceURL] == nil {
				orgSpaceMap[appSpaceURL], err = getOrgSpaceInfo(cli, appSpaceURL)
				if err != nil {
					lo.G.Fatal(err)
				}
			}

			info := orgSpaceMap[appSpaceURL]
			table.Append([]string{appName, info.OrgName, info.SpaceName})
		}

		nextURL = json["next_url"]
	}

	table.Render()
}

// Start - the entry point for the CF RPC server
func (cmd *BuildpackUsage) Start() {
	plugin.Start(cmd)
}

func getBuildpackFromName(cli plugin.CliConnection, buildpackName string) (buildpackGUID string, err error) {
	var buildpackJSON map[string]interface{}

	buildpackJSON, err = cfcurl.Curl(cli, "/v2/buildpacks")
	if err != nil {
		return
	}

	resources := buildpackJSON["resources"].([]interface{})
	for _, resourceIntf := range resources {
		resource := resourceIntf.(map[string]interface{})
		guid := resource["metadata"].(map[string]interface{})["guid"].(string)
		bpName := resource["entity"].(map[string]interface{})["name"].(string)

		if bpName == buildpackName {
			buildpackGUID = guid
			return
		}
	}

	return
}

// Return the buildpack guid from user input
func getBuildpackFromInput(cli plugin.CliConnection, input io.Reader) (buildpackGUID string, buildpackName string, err error) {
	var buildpackJSON map[string]interface{}
	var choice int

	buildpackJSON, err = cfcurl.Curl(cli, "/v2/buildpacks")
	if err != nil {
		return
	}

	var buildpacks = make(map[string]string)
	var buildpackIndexList = make([]string, 0, 8)

	resources := buildpackJSON["resources"].([]interface{})
	for _, resourceIntf := range resources {
		resource := resourceIntf.(map[string]interface{})
		guid := resource["metadata"].(map[string]interface{})["guid"].(string)
		bpName := resource["entity"].(map[string]interface{})["name"].(string)
		buildpacks[guid] = bpName
		buildpackIndexList = append(buildpackIndexList, guid)
	}

	fmt.Println("Please select which buildpack whose apps you would like to see:")
	for idx, bpGUID := range buildpackIndexList {
		fmt.Printf("%d. %s\n", idx+1, buildpacks[bpGUID])
	}

	count := 0
	for !(choice >= 1 && choice <= len(buildpackIndexList)) {
		fmt.Printf("Please choose: ")
		fmt.Fscanf(input, "%d", &choice)

		if count = count + 1; count > 10 {
			err = fmt.Errorf("You've tried too many times!")
			return
		}
	}

	buildpackGUID = buildpackIndexList[choice-1]
	buildpackName = buildpacks[buildpackGUID]

	return
}

func getOrgSpaceInfo(cli plugin.CliConnection, spaceURL string) (info *orgSpaceInfo, err error) {
	json, err := cfcurl.Curl(cli, spaceURL)
	if err != nil {
		return
	}

	info = new(orgSpaceInfo)
	entity := json["entity"].(map[string]interface{})
	info.SpaceName = entity["name"].(string)

	json, err = cfcurl.Curl(cli, entity["organization_url"].(string))
	if err != nil {
		info = nil
		return
	}

	entity = json["entity"].(map[string]interface{})
	info.OrgName = entity["name"].(string)

	return
}
