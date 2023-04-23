package config

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

// Config - Global variable to export
var Config AppConfig

// AppConfig defines
type AppConfig struct {
	Server          ServerConfig          `koanf:"server"`
	Worker          WorkerConfig          `koanf:"worker"`
	Database        DatabaseConfig        `koanf:"database"`
	Temporal        TemporalConfig        `koanf:"temporal"`
	PipelineBackend PipelineBackendConfig `koanf:"pipelinebackend"`
	MgmtBackend     MgmtBackendConfig     `koanf:"mgmtbackend"`
	Controller      ControllerConfig      `koanf:"controller"`
	UsageServer     UsageServerConfig     `koanf:"usageserver"`
}

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	PrivatePort int `koanf:"privateport"`
	PublicPort  int `koanf:"publicport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
	CORSOrigins  []string `koanf:"corsorigins"`
	Edition      string   `koanf:"edition"`
	DisableUsage bool     `koanf:"disableusage"`
	Debug        bool     `koanf:"debug"`
}

// WorkerConfig defines the Temporal Worker configurations
type WorkerConfig struct {
	MountSource struct {
		VDP     string `koanf:"vdp"`
		Airbyte string `koanf:"airbyte"`
	}
	MountTarget struct {
		VDP     string `koanf:"vdp"`
		Airbyte string `koanf:"airbyte"`
	}
}

// DatabaseConfig related to database
type DatabaseConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Name     string `koanf:"name"`
	Version  uint   `koanf:"version"`
	TimeZone string `koanf:"timezone"`
	Pool     struct {
		IdleConnections int           `koanf:"idleconnections"`
		MaxConnections  int           `koanf:"maxconnections"`
		ConnLifeTime    time.Duration `koanf:"connlifetime"`
	}
}

// TemporalConfig related to Temporal
type TemporalConfig struct {
	HostPort   string
	Namespace  string
	Ca         string
	Cert       string
	Key        string
	ServerName string
}

// MgmtBackendConfig related to mgmt-backend
type MgmtBackendConfig struct {
	Host        string `koanf:"host"`
	PrivatePort int    `koanf:"privateport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// PipelineBackendConfig related to pipeline-backend
type PipelineBackendConfig struct {
	Host       string `koanf:"host"`
	PublicPort int    `koanf:"publicport"`
	HTTPS      struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// UsageServerConfig related to usage-backend
type UsageServerConfig struct {
	TLSEnabled bool   `koanf:"tlsenabled"`
	Host       string `koanf:"host"`
	Port       int    `koanf:"port"`
}

// ControllerConfig related to controller
type ControllerConfig struct {
	Host        string `koanf:"host"`
	PrivatePort int    `koanf:"privateport"`
	HTTPS       struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// Init - Assign global config to decoded config struct
func Init() error {

	k := koanf.New(".")
	parser := yaml.Parser()

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fileRelativePath := fs.String("file", "config/config.yaml", "configuration file")
	flag.Parse()

	if err := k.Load(file.Provider(*fileRelativePath), parser); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Load(env.ProviderWithValue("CFG_", ".", func(s string, v string) (string, interface{}) {
		key := strings.Replace(strings.ToLower(strings.TrimPrefix(s, "CFG_")), "_", ".", -1)
		if strings.Contains(v, ",") {
			return key, strings.Split(strings.TrimSpace(v), ",")
		}
		return key, v
	}), nil); err != nil {
		log.Fatal(err.Error())
	}

	if err := k.Unmarshal("", &Config); err != nil {
		log.Fatal(err.Error())
	}

	return ValidateConfig(&Config)
}

// ValidateConfig is for custom validation rules for the configuration
func ValidateConfig(cfg *AppConfig) error {
	return nil
}
