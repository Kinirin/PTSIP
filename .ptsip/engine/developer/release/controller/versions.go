package releasecontroller

import (
	"fmt"
	"regexp"
)

func ToolDistributionVersion(identity string) (string, error) {
	if regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:(a|b|rc)[1-9][0-9]*)?$`).MatchString(identity) {
		return identity, nil
	}
	parts := regexp.MustCompile(`^T\.([0-9]+)\.([0-9]+)\.([0-9]+)\.((?:a|b|rc)[1-9][0-9]*)$`).FindStringSubmatch(identity)
	if len(parts) != 5 {
		return "", fmt.Errorf("Tool identity must be T.major.minor.micro.state")
	}
	return parts[1] + "." + parts[2] + "." + parts[3] + parts[4], nil
}

func PackageDistributionVersion(base string, revision int, state string) (string, error) {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(base) || revision < 0 || (revision == 0 && state == "") {
		return "", fmt.Errorf("explicit Package base and positive revision required")
	}
	if state != "" && !regexp.MustCompile(`^(a|b|rc)[1-9][0-9]*$`).MatchString(state) {
		return "", fmt.Errorf("invalid Package prerelease state")
	}
	if revision == 0 {
		return base + state, nil
	}
	return fmt.Sprintf("%s.%d%s", base, revision, state), nil
}
