package main

import (
	"fmt"
	"time"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	flag "github.com/spf13/pflag"
)

const (
	clusterTableHeader = `
| Cluster | Size | State | Created |
| -- | -- | -- | -- |
`

	installationTableHeader = `
| Installation | DNS | Size | Version | State | Created |
| -- | -- | -- | -- | -- | -- |
`
)

func getStatusFlagSet() *flag.FlagSet {
	statusFlagSet := flag.NewFlagSet("status", flag.ContinueOnError)
	statusFlagSet.Bool("include-clusters", false, "Whether to get cluster status or not")

	return statusFlagSet
}

func parseStatusArgs(args []string) (bool, error) {
	statusFlagSet := getStatusFlagSet()
	err := statusFlagSet.Parse(args)
	if err != nil {
		return false, err
	}

	return statusFlagSet.GetBool("include-clusters")
}

// The status command is primarily intended to help the team administrating the
// cloud infrastructure, so we don't publish the command in the help info.
func (p *Plugin) runStatusCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	includeClusters, err := parseStatusArgs(args)
	if err != nil {
		return nil, true, err
	}

	installations, err := p.cloudClient.GetInstallations(&cloud.GetInstallationsRequest{
		Page:           0,
		PerPage:        100,
		IncludeDeleted: false,
	})
	if err != nil {
		return nil, false, err
	}

	status := installationTableHeader
	for _, installation := range installations {
		status += fmt.Sprintf("| `%s` | [%s](https://%s) | %s | %s | %s | %s |\n",
			installation.ID,
			installation.DNS, installation.DNS,
			installation.Size,
			installation.Version,
			installation.State,
			getTimeFromMillis(installation.CreateAt).Format("Jan-02-2006"),
		)
	}

	if !includeClusters {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, status), false, nil
	}

	clusters, err := p.cloudClient.GetClusters(&cloud.GetClustersRequest{
		Page:           0,
		PerPage:        100,
		IncludeDeleted: false,
	})
	if err != nil {
		return nil, false, err
	}

	status += "\n"
	status += clusterTableHeader
	for _, cluster := range clusters {
		status += fmt.Sprintf("| `%s` | %s | %s |\n",
			cluster.ID,
			cluster.State,
			getTimeFromMillis(cluster.CreateAt).Format("Jan-02-2006"),
		)
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, status), false, nil
}

func getTimeFromMillis(millis int64) time.Time {
	return time.Unix(millis/1000, 0)
}
