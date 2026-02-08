module github.com/DarlingtonDeveloper/MissionControl/cmd/mc

go 1.25.3

require (
	github.com/google/uuid v1.6.0
	github.com/mike/mission-control v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace github.com/mike/mission-control => ../../orchestrator
