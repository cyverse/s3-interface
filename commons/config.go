package commons

import (
	"os"
	"path"
	"path/filepath"

	"golang.org/x/xerrors"
	yaml "gopkg.in/yaml.v2"
)

const (
	ServicePortDefault        int    = 8080
	IrodsPortDefault          int    = 1247
	IrodsSharedDirnameDefault string = "public"
)

func GetDefaultDataRootDirPath() string {
	dirPath, err := os.Getwd()
	if err != nil {
		return "/var/lib/s3rods"
	}
	return dirPath
}

// Config holds the parameters list which can be configured
type Config struct {
	Port         int    `yaml:"port"`
	DataRootPath string `yaml:"data_root_path,omitempty"`

	LogPath string `yaml:"log_path,omitempty"`

	IrodsHost          string `yaml:"irods_host"`
	IrodsPort          int    `yaml:"irods_port"`
	IrodsAdminUsername string `yaml:"irods_admin_username"`
	IrodsAdminPassword string `yaml:"irods_admin_password"`

	IrodsSharedDirname string `yaml:"irods_shared_dirname,omitempty"`

	Foreground   bool `yaml:"foreground,omitempty"`
	Debug        bool `yaml:"debug,omitempty"`
	ChildProcess bool `yaml:"childprocess,omitempty"`
}

// NewDefaultConfig returns a default config
func NewDefaultConfig() *Config {
	return &Config{
		Port:         ServicePortDefault,
		DataRootPath: GetDefaultDataRootDirPath(),

		LogPath: "", // use default

		IrodsHost:          "",
		IrodsPort:          IrodsPortDefault,
		IrodsAdminUsername: "",
		IrodsAdminPassword: "",
		IrodsSharedDirname: IrodsSharedDirnameDefault,

		Foreground:   false,
		Debug:        false,
		ChildProcess: false,
	}
}

// NewConfigFromYAML creates Config from YAML
func NewConfigFromYAML(yamlBytes []byte) (*Config, error) {
	config := NewDefaultConfig()

	err := yaml.Unmarshal(yamlBytes, config)
	if err != nil {
		return nil, xerrors.Errorf("failed to unmarshal yaml into config: %w", err)
	}

	return config, nil
}

// GetLogFilePath returns log file path
func (config *Config) GetLogFilePath() string {
	if len(config.LogPath) > 0 {
		return config.LogPath
	}

	// default
	return path.Join(config.DataRootPath, "service.log")
}

// MakeLogDir makes a log dir required
func (config *Config) MakeLogDir() error {
	logFilePath := config.GetLogFilePath()
	logDirPath := filepath.Dir(logFilePath)
	err := config.makeDir(logDirPath)
	if err != nil {
		return err
	}

	return nil
}

// MakeWorkDirs makes dirs required
func (config *Config) MakeWorkDirs() error {
	err := config.makeDir(config.DataRootPath)
	if err != nil {
		return err
	}

	return nil
}

// CleanWorkDirs cleans dirs used
func (config *Config) CleanWorkDirs() error {
	return nil
}

// makeDir makes a dir for use
func (config *Config) makeDir(path string) error {
	if len(path) == 0 {
		return xerrors.Errorf("failed to create a dir with empty path")
	}

	dirInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// make
			mkdirErr := os.MkdirAll(path, 0775)
			if mkdirErr != nil {
				return xerrors.Errorf("making a dir (%s) error: %w", path, mkdirErr)
			}

			return nil
		}

		return xerrors.Errorf("stating a dir (%s) error: %w", path, err)
	}

	if !dirInfo.IsDir() {
		return xerrors.Errorf("a file (%s) exist, not a directory", path)
	}

	dirPerm := dirInfo.Mode().Perm()
	if dirPerm&0200 != 0200 {
		return xerrors.Errorf("a dir (%s) exist, but does not have the write permission", path)
	}

	return nil
}

// Validate validates configuration
func (config *Config) Validate() error {
	if config.Port <= 0 {
		return xerrors.Errorf("service port must be given")
	}

	if len(config.DataRootPath) == 0 {
		return xerrors.Errorf("data root dir must be given")
	}

	if len(config.IrodsHost) == 0 {
		return xerrors.Errorf("irods host must be given")
	}

	if config.IrodsPort <= 0 {
		return xerrors.Errorf("irods port must be given")
	}

	if len(config.IrodsAdminUsername) == 0 {
		return xerrors.Errorf("irods admin username must be given")
	}

	if len(config.IrodsAdminPassword) == 0 {
		return xerrors.Errorf("irods admin password must be given")
	}

	return nil
}
