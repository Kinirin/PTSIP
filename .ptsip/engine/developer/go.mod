module github.com/Kinirin/PTSIP/.ptsip/engine/developer

go 1.23

require gopkg.in/yaml.v3 v3.0.1

require github.com/Kinirin/PTSIP/src/ptsip/common_contract v0.0.0

require (
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/Kinirin/PTSIP/src/ptsip/common_contract => ../../../src/ptsip/common_contract
