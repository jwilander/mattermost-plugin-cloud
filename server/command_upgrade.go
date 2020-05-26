package main

import (
	"fmt"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

const (
	dockerRepository = "mattermost/mattermost-enterprise-edition"
)

func getUpgradeFlagSet() *flag.FlagSet {
	upgradeFlagSet := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	upgradeFlagSet.String("version", "", "Mattermost version to run, e.g. '5.12.4'")
	upgradeFlagSet.String("license", "", "The enterprise license to use. Can be 'e10', 'e20', or 'te'")
	upgradeFlagSet.String("size", "", "Size of the Mattermost installation e.g. 'miniSingleton' or 'miniHA'")

	return upgradeFlagSet
}

func buildPatchInstallationRequestFromArgs(args []string) (*cloud.PatchInstallationRequest, error) {
	upgradeFlagSet := getUpgradeFlagSet()
	err := upgradeFlagSet.Parse(args)
	if err != nil {
		return nil, err
	}

	version, err := upgradeFlagSet.GetString("version")
	if err != nil {
		return nil, err
	}
	license, err := upgradeFlagSet.GetString("license")
	if err != nil {
		return nil, err
	}
	size, err := upgradeFlagSet.GetString("size")
	if err != nil {
		return nil, err
	}

	if version == "" && license == "" && size == "" {
		return nil, errors.New("must specify at least one option: version, license, or size")
	}
	if license != "" && !validLicenseOption(license) {
		return nil, errors.Errorf("invalid license option %s; must be %s, %s or %s", license, licenseOptionE10, licenseOptionE20, licenseOptionTE)
	}

	request := &cloud.PatchInstallationRequest{}
	if version != "" {
		request.Version = &version
	}
	if license != "" {
		request.License = &license
	}
	if size != "" {
		request.Size = &size
	}

	return request, nil
}

// runUpgradeCommand requests an upgrade and returns the response, an error, and a boolean set to true if a non-nil error is returned due to user error, and false if the error was caused by something else.
func (p *Plugin) runUpgradeCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, _, err := p.getInstallations()
	if err != nil {
		return nil, false, err
	}

	var installToUpgrade *Installation
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installToUpgrade = install
			break
		}
	}

	if installToUpgrade == nil {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	request, err := buildPatchInstallationRequestFromArgs(args)
	if err != nil {
		return nil, true, err
	}

	if request.Version != nil {
		// Check that new version exists.
		var exists bool
		exists, err = p.dockerClient.ValidTag(*request.Version, dockerRepository)
		if err != nil {
			p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", dockerRepository, *request.Version).Error())
		}
		if !exists {
			return nil, true, errors.Errorf("%s is not a valid docker tag for repository %s", *request.Version, dockerRepository)
		}
	}

	if request.License != nil {
		// Translate the license option.
		config := p.getConfiguration()
		switch *request.License {
		case licenseOptionE20:
			request.License = &config.E20License
		case licenseOptionE10:
			request.License = &config.E10License
		case licenseOptionTE:
			var noLicense string
			request.License = &noLicense
		default:
			// This should be checked already, but just in case...
			return nil, true, errors.Errorf("invalid license option %s; must be %s, %s or %s", *request.License, licenseOptionE10, licenseOptionE20, licenseOptionTE)
		}
	}

	_, err = p.cloudClient.UpdateInstallation(installToUpgrade.ID, request)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Upgrade of installation %s has begun. You will receive a notification when it is ready. Use /cloud list to check on the status of your installations.", name)), false, nil
}
