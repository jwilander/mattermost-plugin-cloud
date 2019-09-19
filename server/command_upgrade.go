package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

func getUpgradeFlagSet() *flag.FlagSet {
	upgradeFlagSet := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	upgradeFlagSet.String("version", "", "Mattermost version to run, e.g. '5.12.4'")
	upgradeFlagSet.String("license", "", "The enterprise license to use. Can be 'e10' or 'e20'")

	return upgradeFlagSet
}

func parseUpgradeArgs(args []string) (string, string, error) {
	upgradeFlagSet := getUpgradeFlagSet()
	err := upgradeFlagSet.Parse(args)
	if err != nil {
		return "", "", err
	}

	version, err := upgradeFlagSet.GetString("version")
	if err != nil {
		return "", "", err
	}
	if version == "" {
		return "", "", errors.New("must specify a version")
	}
	license, err := upgradeFlagSet.GetString("license")
	if err != nil {
		return "", "", err
	}
	if license != "" && !validLicenseOption(license) {
		return "", "", fmt.Errorf("invalid license option %s, must be %s or %s", license, licenseOptionE10, licenseOptionE20)
	}

	return version, license, nil
}

func (p *Plugin) runUpgradeCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, fmt.Errorf("must provide an installation name")
	}

	name := args[0]

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
		return nil, true, fmt.Errorf("no installation with the name %s found", name)
	}

	version, newLicense, err := parseUpgradeArgs(args)
	if err != nil {
		return nil, true, err
	}

	repository := "mattermost/mattermost-enterprise-edition"
	exists, err := p.dockerClient.ValidTag(version, repository)
	if err != nil {
		p.API.LogError(errors.Wrapf(err, "unable to check if %s:%s exists", repository, version).Error())
	}
	if !exists {
		return nil, true, fmt.Errorf("%s is not a valid docker tag for repository %s", version, repository)
	}

	config := p.getConfiguration()

	// Only change the license if a value was provided.
	license := installToUpgrade.License
	if newLicense != "" {
		license = config.E20License
		if newLicense == licenseOptionE10 {
			license = config.E10License
		}
	}

	err = p.cloudClient.UpgradeInstallation(installToUpgrade.ID, version, license)
	if err != nil {
		return nil, false, err
	}

	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Upgrade of installation %s has begun. You will receive a notification when it is ready. Use /cloud list to check on the status of your installations.", name)), false, nil
}
