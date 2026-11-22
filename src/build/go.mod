module github.com/Kinirin/PTSIP/src/build

go 1.23

require github.com/Kinirin/PTSIP/src/ptsip/common_contract v0.0.0

require github.com/pelletier/go-toml/v2 v2.4.3

require (
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/Kinirin/PTSIP/src/ptsip/common_contract => ../ptsip/common_contract
