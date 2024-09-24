package config

import (
	"flag"
	"fmt"
	"keysight/laas/controller/internal/utils"
	"os"
	"path"
)

func initConfig() error {
	Config = &config{
		RootDir:                   new(string),
		LogDir:                    new(string),
		WebDir:                    new(string),
		CertsDir:                  new(string),
		MaxLogSizeMB:              new(int),
		MaxLogBackups:             new(int),
		TerminationTimeoutSeconds: new(int),
		NetboxHost:                new(string),
		NetboxUserToken:           new(string),
		FrameworkName:             new(string),
		NetboxApiURL:              new(string),
		L1SwitchLocation:          new(string),
	}
	*Config.MaxLogSizeMB = 25
	*Config.MaxLogBackups = 25
	*Config.TerminationTimeoutSeconds = 3

	// reset flag.CommandLine to show only native flags during usage
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	Config.NetboxHost = flag.String(
		"netbox-host", "",
		"NetBox hostname/ip:port (mandatory)",
	)

	Config.NetboxUserToken = flag.String(
		"netbox-user-token", "",
		"NetBox User Token (mandatory)",
	)

	Config.FrameworkName = flag.String(
		"framework-name", "generic",
		"Generated testbed file format",
	)
	Config.L1SwitchLocation = flag.String(
		"trs-l1s-controller", "l1s-controller:9000",
		"L1Switch hostname/ip:port",
	)

	// In dev-env std-out logging is disabled (check do.sh run)
	// In production-env, std-out logging is enabled by default unless user uses this flag
	Config.DisableStdOutLogging = flag.Bool(
		"no-stdout", false,
		"Disable streaming logs to stdout",
	)
	Config.Cleanup = flag.Bool(
		"cleanup", false,
		"Cleanup logs (and any unwanted assets) before starting service",
	)
	Config.LogLevel = flag.String(
		"log-level", string(LogInfo),
		"Log level for application - info/debug/trace",
	)
	Config.HTTPPort = flag.Int("http-port", 8080, "HTTP Server Port")

	// this is set by Dockerfile
	if *Config.RootDir = os.Getenv("SRC_ROOT"); *Config.RootDir == "" {
		*Config.RootDir = "."
	}

	*Config.LogDir = path.Join(*Config.RootDir, "logs")
	*Config.WebDir = path.Join(*Config.RootDir, "web")
	*Config.CertsDir = path.Join(*Config.RootDir, "certs")

	return nil
}

// ParseFlags parses all command-line flags and exits upon detecting bad flag
func ParseFlags() {
	flag.Parse()
	if len(flag.Args()) != 0 {
		flag.Usage()
		os.Exit(1)
	}

	addr, err := utils.ParseAddr(*Config.NetboxHost)
	if err != nil {
		flag.Usage()
		log.Fatal().Msgf("Error parsing value '%s' for mandatory input netbox-host: %s",
			*Config.NetboxHost, err.Error())
		os.Exit(2)
	}
	*Config.NetboxApiURL = fmt.Sprintf("http://%s:%d/api/", addr.Host, addr.Port)

	if len(*Config.NetboxUserToken) == 0 {
		flag.Usage()
		log.Fatal().Msgf("Error parsing value '%s' for mandatory input netbox-user-token: token can not be empty",
			*Config.NetboxUserToken)
		os.Exit(3)
	}

	if err := validateLogLevel(*Config.LogLevel); err != nil {
		flag.Usage()
		log.Fatal().Msgf("Error parsing value '%s' for input log-level: %s", *Config.LogLevel, err.Error())
		os.Exit(4)
	}

	// since log level depends on LogLevel flag
	RefreshLogLevel()
}

func validateLogLevel(logLevel string) error {
	switch logLevel {
	case string(LogInfo):
	case string(LogDebug):
	case string(LogTrace):
	default:
		return fmt.Errorf("invalid log-level value, please check usage")
	}
	return nil
}

func init() {
	if err := initConfig(); err != nil {
		panic(fmt.Errorf("config init failed: %v", err))
	}

	if err := initLoggers(); err != nil {
		panic(fmt.Errorf("Logger init failed: %v", err))
	}

	log := GetLogger("config")

	log.Info().
		Str("BuildVersion", BuildVersion).
		Str("BuildRevision", BuildRevision).
		Str("BuildCommitHash", BuildCommitHash).
		Str("BuildDate", BuildDate).
		Str("BuildFlavour", BuildFlavour).
		Msg("Build Details")
}
