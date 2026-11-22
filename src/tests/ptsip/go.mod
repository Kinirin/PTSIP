module github.com/Kinirin/PTSIP/contracts-tests

go 1.23

require (
	github.com/Kinirin/PTSIP/src/ptsip v0.0.0
	github.com/Kinirin/PTSIP/src/ptsip/common_contract v0.0.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/dlclark/regexp2 v1.11.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/Kinirin/PTSIP/src/ptsip/common_contract => ../../ptsip/common_contract

replace github.com/Kinirin/PTSIP/src/ptsip => ../../ptsip
