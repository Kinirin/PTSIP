module github.com/Kinirin/PTSIP/.github/checks/release/test

go 1.26

require github.com/Kinirin/PTSIP/.github/checks/release/controller v0.0.0

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/Kinirin/PTSIP/.github/checks/release/controller => ../controller
