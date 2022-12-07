//go:build linux
// +build linux

package system

import (
	"bufio"
	"os"
	"strings"
)

// Inspiration taken from https://github.com/elastic/apm/blob/main/specs/agents/metadata.md#containerkubernetes-metadata
// and https://github.com/elastic/apm-agent-go/blob/main/internal/apmhostutil/container_linux.go
func detectContainer() (*Container, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.SplitN(s.Text(), ":", 3)
		if len(p) != 3 {
			continue
		}

		path := p[2]
		idx := strings.LastIndex(path, ":")
		if idx == -1 {
			if idx = strings.LastIndex(path, "/"); idx == -1 {
				continue
			}
		}
		dirname, basename := path[:idx], path[idx+1:]

		if strings.HasSuffix(basename, ".scope") {
			basename = basename[:len(basename)-6]

			if hyp := strings.Index(basename, "-"); hyp != -1 {
				basename = basename[hyp+1:]
			}
		}

		if match := podUIDRegex.FindStringSubmatch(dirname); match != nil {
			hostname, _ := os.Hostname()

			uid := match[1]
			if uid == "" {
				uid = strings.ReplaceAll(match[2], "_", "-")
			}

			return &Container{
				ID: basename,
				Kubernetes: &Kubernetes{
					PodUID:    uid,
					PodName:   hostname,
					Namespace: os.Getenv("KUBERNETES_NAMESPACE"),
					NodeName:  os.Getenv("KUBERNETES_NODE_NAME"),
				},
			}, nil
		} else if containerIDRegex.MatchString(basename) {
			return &Container{
				ID: basename,
			}, nil
		}
	}

	return nil, nil
}

func detectContainerV2() (*Container, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.Split(s.Text(), " ")
		if len(p) < 4 {
			continue
		}

		if !strings.HasPrefix(p[3], "/var/lib/docker/containers/") {
			continue
		}

		id, _, _ := strings.Cut(p[3][len("/var/lib/docker/containers/"):], "/")
		if !containerIDRegex.MatchString(id) {
			continue
		}

		return &Container{
			ID: id,
		}, nil
	}

	return nil, nil
}
