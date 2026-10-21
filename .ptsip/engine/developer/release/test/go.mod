module github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/test

go 1.26

require github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller v0.0.0

require (
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5
)

replace github.com/Kinirin/PTSIP/.ptsip/engine/developer/release/controller => ../controller
