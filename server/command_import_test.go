package main

import (
	"encoding/json"
	"testing"

	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{}

	api := &plugintest.API{}
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)
	api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
	plugin.SetAPI(api)

	t.Run("get import successfully from valid DNS", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{"indu.dev.cloud.mattermost.com"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation imported")
	})

	t.Run("Invalid DNS display failed to parse", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{"a"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse DNS value")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("import installation successfully with capitalized DNS to show case insensitivity", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{"InDu.DEVLoud.mAtterMost.com"}, &model.CommandArgs{})
		require.NoError(t, err)
		assert.False(t, isUserError)
		assert.Contains(t, resp.Text, "Installation imported")
	})

	t.Run("no DNS provided", func(t *testing.T) {
		resp, isUserError, err := plugin.runImportCommand([]string{}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must provide an installation DNS")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

	t.Run("installs", func(t *testing.T) {
		t.Run("cloud installation not found based on DNS", func(t *testing.T) {
			pluginInstalls := Plugin{
				cloudClient: &MockClient{
					returnNilDNSInstalation: true,
				},
			}
			resp, isUserError, err := pluginInstalls.runImportCommand([]string{"name1.dev.cloud.mattermost.com"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no installation for the DNS provided")
			assert.True(t, isUserError)
			assert.Nil(t, resp)

		})

		t.Run("cloud installation not found based on DNS", func(t *testing.T) {
			pluginInstalls := Plugin{
				cloudClient: &MockClient{
					returnDNSErrorOverride: errors.New("it broke"),
				},
			}
			resp, isUserError, err := pluginInstalls.runImportCommand([]string{"name1.dev.cloud.mattermost.com"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to get installation by DNS")
			assert.False(t, isUserError)
			assert.Nil(t, resp)

		})
	})
	t.Run("URL with https/https and queries", func(t *testing.T) {
		t.Run("get import successfully from valid https DNS", func(t *testing.T) {
			resp, isUserError, err := plugin.runImportCommand([]string{"https://import-me.dev.cloud.mattermost.com"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation imported")
		})

		t.Run("Bad https value", func(t *testing.T) {
			resp, isUserError, err := plugin.runImportCommand([]string{" https://import-me.dev.cloud.mattermost.com"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "error parsing url")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
		t.Run("get import successfully from valid http DNS", func(t *testing.T) {
			resp, isUserError, err := plugin.runImportCommand([]string{"http://import-me.dev.cloud.mattermost.com"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation imported")
		})

		t.Run("Bad http value", func(t *testing.T) {
			resp, isUserError, err := plugin.runImportCommand([]string{" http://import-me.dev.cloud.mattermost.com"}, &model.CommandArgs{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "error parsing url")
			assert.True(t, isUserError)
			assert.Nil(t, resp)
		})
		t.Run("get import successfully from http url with query parameters", func(t *testing.T) {
			resp, isUserError, err := plugin.runImportCommand([]string{"http://import-me.dev.cloud.mattermost.com/api/v1/ping?q=v2"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation imported")
		})
		t.Run("get import successfully from https url with query parameters", func(t *testing.T) {
			resp, isUserError, err := plugin.runImportCommand([]string{"https://import-me.dev.cloud.mattermost.com/api/v1/ping?q=v2"}, &model.CommandArgs{})
			require.NoError(t, err)
			assert.False(t, isUserError)
			assert.Contains(t, resp.Text, "Installation imported")
		})
	})
}

func TestInstallExist(t *testing.T) {
	plugin := Plugin{}
	plugin.cloudClient = &MockClient{
		overrideGetInstallationDTO: &cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id1"}, DNSRecords: []*cloud.InstallationDNS{{DomainName: "installation-one.dev.cloud.mattermost.com"}}}}
	api := &plugintest.API{}
	plugin.SetAPI(api)

	t.Run("installation already imported", func(t *testing.T) {
		_, installationBytes, err := getFakePluginInstallationsWithDNS()
		require.NoError(t, err)
		api.On("KVGet", mock.AnythingOfType("string")).Return(installationBytes, nil)
		api.On("KVCompareAndSet", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(true, nil)
		api.On("LogWarn", mock.AnythingOfType("string")).Return(nil)

		resp, isUserError, err := plugin.runImportCommand([]string{"installation-one.dev.cloud.mattermost.com"}, &model.CommandArgs{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "installation has already been imported to cloud plugin")
		assert.True(t, isUserError)
		assert.Nil(t, resp)
	})

}

func getFakePluginInstallationsWithDNS() ([]*Installation, []byte, error) {
	installations := []*Installation{
		{Name: "installation-one", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id1", OwnerID: "owner 1"}, DNSRecords: []*cloud.InstallationDNS{{DomainName: "installation-one.dev.cloud.mattermost.com"}}}},
		{Name: "installation-two", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id2", OwnerID: "owner 1"}, DNSRecords: []*cloud.InstallationDNS{{DomainName: "installation-two.dev.cloud.mattermost.com"}}}},
		{Name: "installation-three", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id3", OwnerID: "owner 1"}, DNSRecords: []*cloud.InstallationDNS{{DomainName: "installation-three.dev.cloud.mattermost.com"}}}},
		{Name: "installation-four", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id4", OwnerID: "owner 1"}, DNSRecords: []*cloud.InstallationDNS{{DomainName: "installation-four.dev.cloud.mattermost.com"}}}},
		{Name: "installation-five", InstallationDTO: cloud.InstallationDTO{Installation: &cloud.Installation{ID: "id5", OwnerID: "owner 1"}, DNSRecords: []*cloud.InstallationDNS{{DomainName: "installation-five.dev.cloud.mattermost.com"}}}},
	}
	b, err := json.Marshal(installations)

	return installations, b, err
}
