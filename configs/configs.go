package configs

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

// Config - Global variable to export
var Config AppConfig

// AppConfig defines
type AppConfig struct {
	Server      ServerConfig      `koanf:"server"`
	Database    DatabaseConfig    `koanf:"database"`
	MgmtBackend MgmtBackendConfig `koanf:"mgmtbackend"`
}

// ServerConfig defines HTTP server configurations
type ServerConfig struct {
	Port  int `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
	CORSOrigins []string `koanf:"corsorigins"`
	Paginate    struct {
		Salt string `koanf:"salt"`
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

// MgmtBackendConfig related to mgmt-backend
type MgmtBackendConfig struct {
	Host  string `koanf:"host"`
	Port  int    `koanf:"port"`
	HTTPS struct {
		Cert string `koanf:"cert"`
		Key  string `koanf:"key"`
	}
}

// Init - Assign global config to decoded config struct
func Init() error {
	logger, _ := logger.GetZapLogger()

	k := koanf.New(".")
	parser := yaml.Parser()

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fileRelativePath := fs.String("file", "configs/config.yaml", "configuration file")
	flag.Parse()

	if err := k.Load(file.Provider(*fileRelativePath), parser); err != nil {
		logger.Fatal(err.Error())
	}

	if err := k.Load(env.Provider("CFG_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "CFG_")), "_", ".", -1)
	}), nil); err != nil {
		logger.Fatal(err.Error())
	}

	if err := k.Unmarshal("", &Config); err != nil {
		logger.Fatal(err.Error())
	}

	return ValidateConfig(&Config)
}

// ValidateConfig is for custom validation rules for the configuration
func ValidateConfig(cfg *AppConfig) error {
	return nil
}
