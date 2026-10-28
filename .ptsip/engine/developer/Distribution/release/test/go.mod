module github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/test

go 1.26

require (
    github.com/Kinirin/PTSIP/.ptsip/engine/developer v0.0.0
    github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller v0.0.0
)

require (
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	go.yaml.in/yaml/v3 v3.0.5
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/Kinirin/PTSIP/.ptsip/engine/developer/Distribution/release/controller => ../controller

replace github.com/Kinirin/PTSIP/.ptsip/engine/developer => ../../..
