package file

import "os"

const (
	EnvPrefix        = "FILE_"
	DefaultMountPath = "/"
)

type Config struct {
	MountPath string `json:"mountPath"`
}

func NewConfig() *Config {
	mountPath, ok := os.LookupEnv(EnvPrefix + "MOUNT_PATH")
	if !ok {
		mountPath = DefaultMountPath
	}

	return &Config{MountPath: mountPath}
}
