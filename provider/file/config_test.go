package file

import (
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		wantMountPath string
	}{
		{
			name:          "Default mount path",
			env:           map[string]string{},
			wantMountPath: "/",
		},
		{
			name: "Custom mount path",
			env: map[string]string{
				"FILE_MOUNT_PATH": "test/secrets",
			},
			wantMountPath: "test/secrets",
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}

			config := NewConfig()
			if config.MountPath != ttp.wantMountPath {
				t.Errorf("NewConfig() = %v, wantMountPath %v", config.MountPath, ttp.wantMountPath)
			}
		})
	}
}
