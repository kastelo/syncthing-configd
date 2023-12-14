package main

import (
	"net/netip"
	"slices"
	"testing"

	"kastelo.dev/syncthing-autoacceptd/internal/config"
)

func TestPatterns(t *testing.T) {
	t.Parallel()

	cfg := &config.Configuration{
		Pattern: []*config.DevicePattern{
			{
				AcceptCidr: []string{"127.0.0.0/8"},
				Folder: []*config.FolderPattern{
					{Id: "test1"},
				},
			},
			{
				AcceptCidr: []string{"10.0.0.0/8"},
				Folder: []*config.FolderPattern{
					{Id: "test2"},
				},
			},
			{
				AcceptCidr: []string{"127.0.0.0/8"},
				Folder: []*config.FolderPattern{
					{Id: "test3"},
				},
			},
		},
	}

	cases := []struct {
		address string
		accept  bool
		folders []string
	}{
		{
			address: "127.2.3.4",
			accept:  true,
			folders: []string{"test1"},
		},
		{
			address: "10.5.6.7",
			accept:  true,
			folders: []string{"test2"},
		},
		{
			address: "192.168.0.1",
			accept:  false,
			folders: nil,
		},
	}

	for _, c := range cases {
		data := &deviceRejectedData{
			address: netip.MustParseAddr(c.address),
		}
		device, folders, err := getDeviceRejectedConfigs(data, cfg)
		if c.accept {
			if err != nil {
				t.Errorf("getDeviceRejectedConfigs(%q) returned error: %v", c.address, err)
				continue
			}
			var folderNames []string
			for _, f := range folders {
				folderNames = append(folderNames, f.ID)
			}
			if !slices.Equal(folderNames, c.folders) {
				t.Errorf("getDeviceRejectedConfigs(%q) returned folders %v, want %v", c.address, folderNames, c.folders)
			}
		} else if err == nil {
			t.Errorf("getDeviceRejectedConfigs(%q) return no error, want error", device)
		}
	}
}
