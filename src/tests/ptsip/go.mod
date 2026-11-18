module github.com/Kinirin/PTSIP/contracts-tests

go 1.23

require github.com/Kinirin/PTSIP/src/ptsip v0.0.0

require (
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/Kinirin/PTSIP/src/ptsip => ../../ptsip
