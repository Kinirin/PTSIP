module github.com/Kinirin/PTSIP/developer/tests

go 1.26

require (
	github.com/Kinirin/PTSIP/.ptsip/engine/developer v0.0.0
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller v0.0.0
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/test v0.0.0
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/engine v0.0.0
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/profile v0.0.0
	github.com/Kinirin/PTSIP/src/ptsip/common_contract v0.0.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	go.yaml.in/yaml/v3 v3.0.5
)

require (
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/Kinirin/PTSIP/.ptsip/engine/developer => ../../.ptsip/engine/developer
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller => ../../.ptsip/engine/developer/Distribution/release/controller
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/test => ../../.ptsip/engine/developer/Distribution/release/test
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/engine => ../../.ptsip/engine/developer/Distribution/release/verification/engine
	github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/verification/profile => ../../.ptsip/engine/developer/Distribution/release/verification/profile
	github.com/Kinirin/PTSIP/src/ptsip/common_contract => ../../src/ptsip/common_contract
)
