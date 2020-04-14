package main

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	cloudClient  CloudClient
	dockerClient DockerClientInterface

	BotUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// CloudClient is the interface for managing cloud installations.
type CloudClient interface {
	GetClusters(*cloud.GetClustersRequest) ([]*cloud.Cluster, error)

	CreateInstallation(request *cloud.CreateInstallationRequest) (*cloud.Installation, error)
	GetInstallation(installationID string, request *cloud.GetInstallationRequest) (*cloud.Installation, error)
	GetInstallations(*cloud.GetInstallationsRequest) ([]*cloud.Installation, error)
	UpdateInstallation(installationID string, request *cloud.PatchInstallationRequest) (*cloud.Installation, error)
	DeleteInstallation(installationID string) error

	GetClusterInstallations(request *cloud.GetClusterInstallationsRequest) ([]*cloud.ClusterInstallation, error)
	RunMattermostCLICommandOnClusterInstallation(clusterInstallationID string, subcommand []string) ([]byte, error)

	GetGroup(groupID string) (*cloud.Group, error)
}

// DockerClientInterface is the interface for interacting with docker.
type DockerClientInterface interface {
	ValidTag(desiredTag, repository string) (bool, error)
}

// BuildHash is the full git hash of the build.
var BuildHash string

// BuildHashShort is the short git hash of the build.
var BuildHashShort string

// BuildDate is the build date of the build.
var BuildDate string

// OnActivate runs when the plugin activates and ensures the plugin is properly
// configured.
func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()
	if err := config.IsValid(); err != nil {
		return err
	}

	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    "cloud",
		DisplayName: "Cloud",
		Description: "Created by the Mattermost Private Cloud plugin.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure github bot")
	}
	p.BotUserID = botID

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	appErr := p.API.SetProfileImage(botID, profileImage)
	if appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	p.setCloudClient()
	p.dockerClient = NewDockerClient()
	return p.API.RegisterCommand(getCommand())
}
