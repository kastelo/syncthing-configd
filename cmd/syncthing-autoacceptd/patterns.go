package main

import (
	"errors"

	stconfig "github.com/syncthing/syncthing/lib/config"
	stfs "github.com/syncthing/syncthing/lib/fs"
	"kastelo.dev/syncthing-autoacceptd/internal/config"
)

var errNoMatchingPattern = errors.New("device does not match any pattern")

func getDeviceRejectedConfigs(data *deviceRejectedData, cfg *config.Configuration) (*stconfig.DeviceConfiguration, []*stconfig.FolderConfiguration, error) {
	addFolders := make([]*stconfig.FolderConfiguration, 0)
	var addDevice *stconfig.DeviceConfiguration
	for _, pat := range cfg.Pattern {
		settings := pat.Settings
		if settings == nil { // avoid having to use getters everywhere
			settings = &config.DeviceConfiguration{}
		}

		if pat.MatchesAddress(data.address) {
			addDevice = &stconfig.DeviceConfiguration{
				DeviceID:          data.device,
				Name:              data.name,
				Addresses:         settings.Addresses,
				AllowedNetworks:   settings.AllowedNetworks,
				AutoAcceptFolders: settings.AutoAcceptFolders,
				MaxSendKbps:       int(settings.MaxSendKbps),
				MaxRecvKbps:       int(settings.MaxRecvKbps),
				MaxRequestKiB:     int(settings.MaxRequestKib),
				RawNumConnections: int(settings.NumConnections),
			}

			for _, fld := range pat.Folder {
				settings := fld.Settings
				if settings == nil { // avoid having to use getters everywhere
					settings = &config.FolderConfiguration{}
				}

				id, err := replaceVariables(fld.Id, data)
				if err != nil {
					return nil, nil, err
				}
				path, err := replaceVariables(settings.Path, data)
				if err != nil {
					return nil, nil, err
				}
				label, err := replaceVariables(settings.Label, data)
				if err != nil {
					return nil, nil, err
				}

				folderCfg := &stconfig.FolderConfiguration{
					ID:               id,
					Path:             path,
					Type:             stconfig.FolderType(settings.Type),
					Label:            label,
					RescanIntervalS:  int(settings.RescanIntervalS),
					FSWatcherEnabled: !settings.FsWatcherDisabled,
					FSWatcherDelayS:  settings.FsWatcherDelayS,
					IgnorePerms:      settings.IgnorePermissions,
					AutoNormalize:    !settings.NoAutoNormalize,
					MinDiskFree: stconfig.Size{
						Value: settings.GetMinDiskFree().GetValue(),
						Unit:  settings.GetMinDiskFree().GetUnit(),
					},
					Copiers:                 int(settings.Copiers),
					PullerMaxPendingKiB:     int(settings.PullerMaxPendingKib),
					Hashers:                 int(settings.Hashers),
					Order:                   stconfig.PullOrder(settings.Order),
					IgnoreDelete:            settings.IgnoreDelete,
					ScanProgressIntervalS:   int(settings.ScanProgressIntervalS),
					PullerPauseS:            int(settings.PullerPauseS),
					MaxConflicts:            int(settings.MaxConflicts),
					DisableSparseFiles:      settings.DisableSparseFiles,
					DisableTempIndexes:      settings.DisableTempIndexes,
					WeakHashThresholdPct:    int(settings.WeakHashThresholdPct),
					MarkerName:              settings.MarkerName,
					CopyOwnershipFromParent: settings.CopyOwnershipFromParent,
					RawModTimeWindowS:       int(settings.ModTimeWindowS),
					MaxConcurrentWrites:     int(settings.MaxConcurrentWrites),
					DisableFsync:            settings.DisableFsync,
					BlockPullOrder:          stconfig.BlockPullOrder(settings.BlockPullOrder),
					CopyRangeMethod:         stfs.CopyRangeMethod(settings.CopyRangeMethod),
					CaseSensitiveFS:         settings.CaseSensitiveFs,
					JunctionsAsDirs:         settings.FollowJunctions,
					SyncOwnership:           settings.SyncOwnership,
					SendOwnership:           settings.SendOwnership,
					SyncXattrs:              settings.SyncXattrs,
					SendXattrs:              settings.SendXattrs,
					Devices: []stconfig.FolderDeviceConfiguration{
						{DeviceID: data.device},
					},
				}
				addFolders = append(addFolders, folderCfg)
			}
			break
		}
	}
	if addDevice == nil {
		return nil, nil, errNoMatchingPattern
	}
	return addDevice, addFolders, nil
}
