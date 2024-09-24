module keysight/laas/controller

go 1.21.4

replace github.com/openconfig/ondatra => github.com/open-traffic-generator/ondatra v0.0.0-20240422051422-f92428db5b29

require (
	github.com/gorilla/mux v1.8.1
	github.com/open-traffic-generator/openl1s/gol1s v0.0.0-20240730105808-bdfb71f88b3d
	github.com/open-traffic-generator/opentestbed/goopentestbed v0.0.4
	github.com/openconfig/featureprofiles v0.0.0-20240730070341-f0c6d0220f2e
	github.com/openconfig/ondatra v0.6.0
	github.com/rs/zerolog v1.33.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/golang/glog v1.2.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
