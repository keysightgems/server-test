package config

// all the fields are pointers because most of them will store *flags*
type config struct {
	RootDir                   *string
	WebDir                    *string
	CertsDir                  *string
	LogDir                    *string
	MaxLogSizeMB              *int
	MaxLogBackups             *int
	TerminationTimeoutSeconds *int
	DisableStdOutLogging      *bool
	Cleanup                   *bool
	LogLevel                  *string
	NetboxHost                *string
	NetboxUserToken           *string
	FrameworkName             *string
	NetboxApiURL              *string
	L1SwitchLocation          *string
	HTTPPort                  *int
}

var (
	// Config is a singleton which holds app configurations (default or passed
	// via command line options) consumed by all packages
	Config *config
	// BuildVersion is set during compile time
	BuildVersion string
	// BuildRevision is set during compile time
	BuildRevision string
	// BuildCommitHash is set during compile time
	BuildCommitHash string
	// BuildDate is set during compile time
	BuildDate string
	// BuildType is set during compile time
	BuildFlavour string = "production"
	// Global or main logger
	log Logger
)

// LogLevel specifies the depth of logging used in the application
type LogLevel string

const (
	// Different log-level values
	LogInfo  LogLevel = "info"
	LogDebug LogLevel = "debug"
	LogTrace LogLevel = "trace"
)
