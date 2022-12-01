package system

import (
	"regexp"
	"sync"
)

// Kubernetes represents information about the kubernetes environment.
type Kubernetes struct {
	// Namespace is the namespace of the pod.
	Namespace string
	// PodName is the name of the pod.
	PodName string
	// PodUID is the unique identifier of the pod.
	PodUID string
	// NodeName is the name of the node.
	NodeName string
}

// Container metadata.
type Container struct {
	// ID of the container.
	ID string
	// Kubernetes metadata.
	Kubernetes *Kubernetes
}

var (
	containerOnce sync.Once
	container     *Container

	podUIDRegex      = regexp.MustCompile(`(?:^/kubepods[\\S]*/pod([^/]+)$)|(?:kubepods[^/]*-pod([^/]+)\.slice)`)
	containerIDRegex = regexp.MustCompile(`^[[:xdigit:]]{64}$|^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4,}$|^[[:xdigit:]]{32}-[[:digit:]]{10}$`)
)

func containerInfo() *Container {
	containerOnce.Do(func() {
		container, _ = detectContainer()
	})
	return container
}
