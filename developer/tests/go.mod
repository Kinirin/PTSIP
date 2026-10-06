module github.com/Kinirin/PTSIP/developer/tests

go 1.26

require (
	github.com/Kinirin/PTSIP/developer/automation v0.0.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	go.yaml.in/yaml/v3 v3.0.5
)

require golang.org/x/text v0.14.0 // indirect

replace github.com/Kinirin/PTSIP/developer/automation => ../automation
