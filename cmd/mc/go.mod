module github.com/MikeSquared-Agency/MissionControl/cmd/mc

go 1.22

require (
	github.com/MikeSquared-Agency/MissionControl v0.0.0
	github.com/google/uuid v1.6.0
	github.com/spf13/cobra v1.8.1
)

require github.com/gorilla/websocket v1.5.3 // indirect

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace github.com/MikeSquared-Agency/MissionControl => ../../orchestrator
