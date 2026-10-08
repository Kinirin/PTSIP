package release

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kinirin/PTSIP/developer/automation/internal/machine"
)

var projectVersionPattern = regexp.MustCompile(`^version\s*=\s*["']([^"']+)["']\s*$`)

func projectVersion(repo *machine.Repository) (string, error) {
	path, err := repo.Path("pyproject.toml")
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	inProject := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inProject = line == "[project]"
			continue
		}
		if !inProject {
			continue
		}
		match := projectVersionPattern.FindStringSubmatch(line)
		if len(match) == 2 {
			return match[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", machine.Fail("RELEASE_VERSION_UNRESOLVED", "pyproject.toml [project].version is missing")
}

func run(repo *machine.Repository, name string, args ...string) (string, int, error) {
	command := exec.Command(name, args...)
	command.Dir = repo.Root
	output, err := command.CombinedOutput()
	if err == nil {
		return strings.TrimSpace(string(output)), 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return strings.TrimSpace(string(output)), exit.ExitCode(), nil
	}
	return strings.TrimSpace(string(output)), -1, err
}

func requireRun(repo *machine.Repository, name string, args ...string) (string, error) {
	output, code, err := run(repo, name, args...)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", machine.Fail("RELEASE_COMMAND_FAILED", fmt.Sprintf("%s %v exited %d: %s", name, args, code, output))
	}
	return output, nil
}

func Prepare(repo *machine.Repository, dispatchedSHA, dispatchedRef string) (machine.Object, error) {
	if dispatchedRef != "refs/heads/main" {
		return nil, machine.Fail("RELEASE_NOT_MAIN", "release preparation must be dispatched from refs/heads/main")
	}
	sourceSHA, err := requireRun(repo, "git", "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if sourceSHA != dispatchedSHA {
		return nil, machine.Fail("RELEASE_DISPATCH_SHA_MISMATCH", fmt.Sprintf("checkout %s differs from dispatch %s", sourceSHA, dispatchedSHA))
	}
	if _, err := requireRun(repo, "git", "fetch", "--force", "--tags", "origin", "refs/heads/main:refs/remotes/origin/main"); err != nil {
		return nil, err
	}
	mainSHA, err := requireRun(repo, "git", "rev-parse", "origin/main")
	if err != nil {
		return nil, err
	}
	if mainSHA != sourceSHA {
		return nil, machine.Fail("RELEASE_MAIN_MOVED", fmt.Sprintf("candidate %s is not current origin/main %s", sourceSHA, mainSHA))
	}
	version, err := projectVersion(repo)
	if err != nil {
		return nil, err
	}
	tag := "tool-v" + version
	note := "docs/releasenote/tool/" + version + ".md"
	_, code, err := run(repo, "git", "show-ref", "--verify", "--quiet", "refs/tags/"+tag)
	if err != nil {
		return nil, err
	}
	if code == 0 {
		return nil, machine.Fail("RELEASE_TAG_EXISTS", "tag already exists: "+tag)
	}
	if code != 1 {
		return nil, machine.Fail("RELEASE_TAG_LOOKUP_FAILED", "unable to determine whether tag exists: "+tag)
	}
	notePath, err := repo.Path(note)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(notePath)
	if err != nil {
		return nil, machine.Fail("RELEASE_NOTE_MISSING", note)
	}
	text := string(data)
	if strings.TrimSpace(text) == "" {
		return nil, machine.Fail("RELEASE_NOTE_EMPTY", note)
	}
	hasSection := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			hasSection = true
			break
		}
	}
	if !hasSection {
		return nil, machine.Fail("RELEASE_NOTE_UNCATEGORIZED", note)
	}
	return machine.Object{
		"status":     "PASS",
		"version":    version,
		"tag":        tag,
		"note":       filepath.ToSlash(note),
		"source_sha": sourceSHA,
	}, nil
}

func ReconfirmMain(repo *machine.Repository, sourceSHA string) (machine.Object, error) {
	if _, err := requireRun(repo, "git", "fetch", "--force", "origin", "refs/heads/main:refs/remotes/origin/main"); err != nil {
		return nil, err
	}
	mainSHA, err := requireRun(repo, "git", "rev-parse", "origin/main")
	if err != nil {
		return nil, err
	}
	if mainSHA != sourceSHA {
		return nil, machine.Fail("RELEASE_MAIN_MOVED", fmt.Sprintf("origin/main moved to %s after verification of %s", mainSHA, sourceSHA))
	}
	return machine.Object{"status": "PASS", "source_sha": sourceSHA, "main_sha": mainSHA}, nil
}

func VerifyTag(repo *machine.Repository, releaseTag string) (machine.Object, error) {
	version, err := projectVersion(repo)
	if err != nil {
		return nil, err
	}
	expected := "tool-v" + version
	if releaseTag != expected {
		return nil, machine.Fail("RELEASE_TAG_VERSION_MISMATCH", fmt.Sprintf("%s != %s", releaseTag, expected))
	}
	tags, err := requireRun(repo, "git", "tag", "--points-at", "HEAD")
	if err != nil {
		return nil, err
	}
	found := false
	for _, tag := range strings.Fields(tags) {
		found = found || tag == releaseTag
	}
	if !found {
		return nil, machine.Fail("RELEASE_TAG_NOT_AT_HEAD", releaseTag)
	}
	return machine.Object{"status": "PASS", "version": version, "tag": releaseTag}, nil
}
