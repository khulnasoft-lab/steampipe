package workspace

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/turbot/steampipe/constants"
	"github.com/turbot/steampipe/ociinstaller"
	"github.com/turbot/steampipe/plugin"
	"github.com/turbot/steampipe/steampipeconfig"
	"github.com/turbot/steampipe/steampipeconfig/modconfig"
	"github.com/turbot/steampipe/utils"
)

func (w *Workspace) CheckRequiredPluginsInstalled() error {
	// get the list of all installed plugins
	installedPlugins, err := w.getInstalledPlugins()
	if err != nil {
		return err
	}

	// get the list of all the required plugins
	requiredPlugins := w.getRequiredPlugins()

	var pluginsNotInstalled []requiredPluginVersion
	var pluginsWithNoConnection []requiredPluginVersion

	for name, requiredVersion := range requiredPlugins {
		var req = requiredPluginVersion{plugin: name}
		req.SetRequiredVersion(requiredVersion)

		if installedVersion, found := installedPlugins[name]; found {
			req.SetInstalledVersion(installedVersion)
			connectionsForPlugin := steampipeconfig.GlobalConfig.ConnectionsForPlugin(name, installedVersion)
			req.matchingConnections = connectionsForPlugin

			// if there are no matching connections raise an error
			if len(req.matchingConnections) == 0 {
				pluginsWithNoConnection = append(pluginsWithNoConnection, req)
			} else {
				// check if installed plugin satisfies version
				if installedPlugins[name].LessThan(requiredVersion) {
					pluginsNotInstalled = append(pluginsNotInstalled, req)
				}
			}
		} else {
			req.installedVersion = "none"
			pluginsNotInstalled = append(pluginsNotInstalled, req)
		}

	}
	if len(pluginsNotInstalled)+len(pluginsWithNoConnection) > 0 {
		return errors.New(pluginVersionError(pluginsNotInstalled, pluginsWithNoConnection))
	}

	return nil
}

func (w *Workspace) getRequiredPlugins() map[string]*version.Version {
	if w.Mod.Requires != nil {
		requiredPluginVersions := w.Mod.Requires.Plugins
		requiredVersion := make(map[string]*version.Version)
		for _, pluginVersion := range requiredPluginVersions {
			requiredVersion[pluginVersion.ShortName()] = pluginVersion.Version
		}
		return requiredVersion
	}
	return nil
}

func (w *Workspace) getInstalledPlugins() (map[string]*version.Version, error) {
	installedPlugins := make(map[string]*version.Version)
	installedPluginsData, _ := plugin.List(nil)
	for _, plugin := range installedPluginsData {
		org, name, _ := ociinstaller.NewSteampipeImageRef(plugin.Name).GetOrgNameAndStream()
		semverVersion, err := version.NewVersion(plugin.Version)
		if err != nil {
			continue
		}
		pluginShortName := fmt.Sprintf("%s/%s", org, name)
		installedPlugins[pluginShortName] = semverVersion
	}
	return installedPlugins, nil
}

type requiredPluginVersion struct {
	plugin              string
	requiredVersion     string
	installedVersion    string
	matchingConnections []*modconfig.Connection
}

func (v *requiredPluginVersion) SetRequiredVersion(requiredVersion *version.Version) {
	requiredVersionString := requiredVersion.String()
	// if no required version was specified, the version will be 0.0.0
	if requiredVersionString == "0.0.0" {
		v.requiredVersion = "latest"
	} else {
		v.requiredVersion = requiredVersionString
	}
}

func (v *requiredPluginVersion) SetInstalledVersion(installedVersion *version.Version) {
	v.installedVersion = installedVersion.String()
}

func pluginVersionError(pluginsNotInstalled []requiredPluginVersion, pluginsWithNoConnection []requiredPluginVersion) string {
	failureCount := len(pluginsNotInstalled) + len(pluginsWithNoConnection)
	var notificationLines = []string{
		fmt.Sprintf("%d mod plugin %s not satisfied. ", failureCount, utils.Pluralize("requirement", failureCount)),
		"",
	}
	longestNameLength := 0
	for _, report := range pluginsNotInstalled {
		thisName := report.plugin
		if len(thisName) > longestNameLength {
			longestNameLength = len(thisName)
		}
	}

	for _, report := range pluginsWithNoConnection {
		thisName := report.plugin
		if len(thisName) > longestNameLength {
			longestNameLength = len(thisName)
		}
	}

	// sort alphabetically
	sort.Slice(pluginsNotInstalled, func(i, j int) bool {
		return pluginsNotInstalled[i].plugin < pluginsNotInstalled[j].plugin
	})
	sort.Slice(pluginsWithNoConnection, func(i, j int) bool {
		return pluginsWithNoConnection[i].plugin < pluginsWithNoConnection[j].plugin
	})

	// build first part of string
	// recheck longest names
	longestVersionLength := 0

	var notInstalledStrings = make([]string, len(pluginsNotInstalled))
	var noConnectionString = make([]string, len(pluginsWithNoConnection))
	for i, req := range pluginsNotInstalled {
		format := fmt.Sprintf("  %%-%ds  %%-2s", longestNameLength)
		notInstalledStrings[i] = fmt.Sprintf(
			format,
			req.plugin,
			req.installedVersion,
		)

		if len(notInstalledStrings[i]) > longestVersionLength {
			longestVersionLength = len(notInstalledStrings[i])
		}
	}

	for i, req := range pluginsNotInstalled {
		format := fmt.Sprintf("%%-%ds  →  %%2s", longestVersionLength)
		notificationLines = append(notificationLines, fmt.Sprintf(
			format,
			notInstalledStrings[i],
			constants.Bold(req.requiredVersion),
		))
	}

	for i, req := range pluginsWithNoConnection {
		format := fmt.Sprintf("  %%-%ds  [no compatible connection configured]", longestNameLength)
		noConnectionString[i] = fmt.Sprintf(
			format,
			req.plugin,
		)

		if len(noConnectionString[i]) > longestVersionLength {
			longestVersionLength = len(noConnectionString[i])
		}
	}

	for i, _ := range pluginsWithNoConnection {
		format := fmt.Sprintf("%%-%ds", longestVersionLength)
		notificationLines = append(notificationLines, fmt.Sprintf(
			format,
			noConnectionString[i],
		))
	}

	// add blank line (hack - bold the empty string to force it to print blank line as part of error)
	notificationLines = append(notificationLines, fmt.Sprintf("%s", constants.Bold("")))

	return strings.Join(notificationLines, "\n")
}
