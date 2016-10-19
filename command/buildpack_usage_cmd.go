package command

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/cloudfoundry/cli/plugin"

	"github.com/bradfitz/slice"
	"github.com/krujos/cfcurl"
)

type orgSpaceInfo struct {
	OrgName   string
	SpaceName string
}

type appLocator struct {
	orgSpaceInfo
	Name string
}

// BuildpackUsage - the main struct to implement the plugin struct
type BuildpackUsage struct {
	Input  io.Reader
	Output io.Writer
}

// New - returns a new instance of the command
func New() *BuildpackUsage {
	i18n.T = func(translationID string, args ...interface{}) string {
		if len(args) == 0 {
			return fmt.Sprintf("%s\n", translationID)
		}

		return fmt.Sprintf(translationID+"\n", args...)
	}

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
			Build: 9,
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
	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if recover() != nil {
			os.Exit(1)
		}
	}()

	var buildpackGUID string
	var buildpackName string
	var err error

	if args[0] != cmd.GetMetadata().Commands[0].Name {
		return
	}

	flagSet := flag.NewFlagSet("command-args", flag.ContinueOnError)

	flagSet.StringVar(&buildpackName, "b", "NOT_SET", "The requested buildpack")
	flagSet.Parse(args[1:])

	ui := terminal.NewUI(
		os.Stdin,
		Writer,
		terminal.NewTeePrinter(Writer),
		trace.NewLogger(Writer, false, "false", ""),
	)

	if buildpackName == "NOT_SET" {
		buildpackGUID, buildpackName, err = getBuildpackFromInput(cli, cmd.Input, cmd.Output)
	} else {
		buildpackGUID, err = getBuildpackFromName(cli, buildpackName)
	}

	ui.Say("Checking which apps use buildpack %s ...\n", terminal.CommandColor(buildpackName))

	var apps []appLocator
	if err == nil {
		apps, err = getAppsByBuildpackGUID(cli, buildpackGUID)
	}

	if err != nil {
		ui.Failed("Error completing request: %v", err)
		return
	}

	ui.Ok()

	if len(apps) == 0 {
		ui.Say("No apps found")
		os.Exit(0)
	}

	table := ui.Table([]string{"org", "space", "application"})
	for _, app := range apps {
		table.Add(app.OrgName, app.SpaceName, app.Name)
	}

	table.Print()
}

// Start - the entry point for the CF RPC server
func (cmd *BuildpackUsage) Start() {
	plugin.Start(cmd)
}

func getAppsByBuildpackGUID(cli plugin.CliConnection, buildpackGUID string) (apps []appLocator, err error) {
	apps = make([]appLocator, 0, 5)
	orgSpaceMap := make(map[string]*orgSpaceInfo)

	var json map[string]interface{}

	var nextURL interface{}
	nextURL = "/v2/apps"

	for nextURL != nil {
		json, err = cfcurl.Curl(cli, nextURL.(string))
		if err != nil {
			return
		}

		appsResources := toJSONArray(json["resources"])
		for _, appIntf := range appsResources {
			app := toJSONObject(appIntf)
			appEntity := toJSONObject(app["entity"])
			appName := appEntity["name"].(string)
			appSpaceURL := appEntity["space_url"].(string)

			appBuildpackGUID := appEntity["detected_buildpack_guid"]

			if appBuildpackGUID == nil || (appBuildpackGUID.(string) != buildpackGUID) {
				continue
			}

			if orgSpaceMap[appSpaceURL] == nil {
				orgSpaceMap[appSpaceURL], err = getOrgSpaceInfo(cli, appSpaceURL)
				if err != nil {
					return
				}
			}

			info := orgSpaceMap[appSpaceURL]

			appInfo := appLocator{orgSpaceInfo: *info, Name: appName}

			apps = append(apps, appInfo)
		}

		nextURL = json["next_url"]
	}

	slice.Sort(apps, func(i, j int) bool {
		locator1, locator2 := apps[i], apps[j]
		if locator1.OrgName < locator2.OrgName {
			return true
		} else if locator1.OrgName > locator2.OrgName {
			return false
		}

		if locator1.SpaceName < locator2.SpaceName {
			return true
		} else if locator1.SpaceName > locator2.SpaceName {
			return false
		}

		if locator1.Name <= locator2.Name {
			return true
		}

		return false
	})

	return
}

func getBuildpackFromName(cli plugin.CliConnection, buildpackName string) (buildpackGUID string, err error) {
	var buildpackJSON map[string]interface{}

	buildpackJSON, err = cfcurl.Curl(cli, "/v2/buildpacks")
	if err != nil {
		return
	}

	resources := toJSONArray(buildpackJSON["resources"])
	for _, resourceIntf := range resources {
		resource := toJSONObject(resourceIntf)
		guid := toJSONObject(resource["metadata"])["guid"].(string)
		bpName := toJSONObject(resource["entity"])["name"].(string)

		if bpName == buildpackName {
			buildpackGUID = guid
			return
		}
	}

	err = fmt.Errorf("Could not find buildpack %s", terminal.CommandColor(buildpackName))
	return
}

// Return the buildpack guid from user input
func getBuildpackFromInput(cli plugin.CliConnection, input io.Reader, output io.Writer) (buildpackGUID string, buildpackName string, err error) {
	var buildpackJSON map[string]interface{}
	var choice int

	buildpackJSON, err = cfcurl.Curl(cli, "/v2/buildpacks")
	if err != nil {
		return
	}

	var buildpacks = make(map[string]string)
	var buildpackIndexList = make([]string, 0, 8)

	resources := toJSONArray(buildpackJSON["resources"])
	for _, resourceIntf := range resources {
		resource := toJSONObject(resourceIntf)
		guid := toJSONObject(resource["metadata"])["guid"].(string)
		bpName := toJSONObject(resource["entity"])["name"].(string)
		buildpacks[guid] = bpName
		buildpackIndexList = append(buildpackIndexList, guid)
	}

	fmt.Fprintln(output, "Please select which buildpack whose apps you would like to see:")
	for idx, bpGUID := range buildpackIndexList {
		fmt.Printf("%d. %s\n", idx+1, buildpacks[bpGUID])
	}

	for !(choice >= 1 && choice <= len(buildpackIndexList)) {
		fmt.Fprintf(output, "Please choose: ")
		fmt.Fscanf(input, "%d", &choice)
		fmt.Fprintln(output)
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
	entity := toJSONObject(json["entity"])
	info.SpaceName = entity["name"].(string)

	json, err = cfcurl.Curl(cli, entity["organization_url"].(string))
	if err != nil {
		info = nil
		return
	}

	entity = toJSONObject(json["entity"])
	info.OrgName = entity["name"].(string)

	return
}

func toJSONArray(obj interface{}) []interface{} {
	return obj.([]interface{})
}

func toJSONObject(obj interface{}) map[string]interface{} {
	return obj.(map[string]interface{})
}
