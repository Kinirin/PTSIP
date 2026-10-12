module github.com/Kinirin/PTSIP/developer/automation

go 1.26

require (
	github.com/Kinirin/PTSIP/developer/automation/release v0.0.0
	github.com/dlclark/regexp2 v1.11.0
	github.com/odvcencio/gotreesitter v0.55.1
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	go.yaml.in/yaml/v3 v3.0.5
)

replace github.com/Kinirin/PTSIP/developer/automation/release => ../../.github/checks/release

require golang.org/x/text v0.14.0 // indirect
