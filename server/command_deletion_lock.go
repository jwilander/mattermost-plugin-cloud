package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

func (p *Plugin) lockForDeletion(installationID string, userID string) error {
	installations, err := p.getUpdatedInstallsForUser(userID)
	if err != nil {
		return err
	}

	maxLockedInstallations := p.getConfiguration().DeletionLockInstallationsAllowedPerPerson

	numExistingLockedInstallations := 0
	var installationToLock *Installation
	for _, install := range installations {
		if install.OwnerID == userID && install.ID == installationID {
			installationToLock = install
			break
		}
		if install.DeletionLocked {
			numExistingLockedInstallations++
		}
	}

	if maxLockedInstallations <= numExistingLockedInstallations {
		return fmt.Errorf("you may only have at most %d installations locked for deletion at a time", maxLockedInstallations)
	}

	if installationToLock == nil {
		return errors.New("installation to be locked not found")
	}

	err = p.cloudClient.LockDeletionLockForInstallation(installationToLock.ID)
	return err
}

func (p *Plugin) unlockForDeletion(installationID string, userID string) error {
	installations, err := p.getUpdatedInstallsForUser(userID)
	if err != nil {
		return err
	}

	var installationToLock *Installation
	for _, install := range installations {
		if install.OwnerID == userID && install.ID == installationID {
			installationToLock = install
			break
		}
	}

	if installationToLock == nil {
		return errors.New("installation to be locked not found")
	}

	err = p.cloudClient.UnlockDeletionLockForInstallation(installationToLock.ID)
	return err
}

func (p *Plugin) runDeletionLockCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installations, err := p.getUpdatedInstallsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}
	var installationIdToLock string
	for _, installation := range installations {
		if installation.OwnerID == extra.UserId && installation.Name == name {
			installationIdToLock = installation.ID
			break
		}
	}

	if installationIdToLock == "" {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	err = p.lockForDeletion(installationIdToLock, extra.UserId)

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Deletion lock has been applied, your workspace will be preserved.", extra), false, nil
}

func (p *Plugin) runDeletionUnlockCommand(args []string, extra *model.CommandArgs) (*model.CommandResponse, bool, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return nil, true, errors.Errorf("must provide an installation name")
	}

	name := standardizeName(args[0])

	installs, err := p.getUpdatedInstallsForUser(extra.UserId)
	if err != nil {
		return nil, false, err
	}

	var installationIdToUnlock string
	for _, install := range installs {
		if install.OwnerID == extra.UserId && install.Name == name {
			installationIdToUnlock = install.ID
			break
		}
	}

	if installationIdToUnlock == "" {
		return nil, true, errors.Errorf("no installation with the name %s found", name)
	}

	err = p.unlockForDeletion(installationIdToUnlock, extra.UserId)

	return getCommandResponse(model.CommandResponseTypeEphemeral, "Deletion lock has been applied, your workspace will be preserved.", extra), false, nil
}
