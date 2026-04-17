package system

import "os"

// Info contains system information including hostname and container metadata.
type Info struct {
	Hostname  string
	Container *Container
}

// IsKubernetes returns true if running inside a Kubernetes pod.
func (i Info) IsKubernetes() bool {
	return i.IsContainer() && i.Container.Kubernetes != nil
}

// IsContainer returns true if running inside a container.
func (i Info) IsContainer() bool {
	return i.Container != nil
}

// CollectInfo collects system information.
func CollectInfo() *Info {
	// TODO: use FQDN to get hostname
	// Ignore error
	hostname, _ := os.Hostname()

	return &Info{
		Hostname:  hostname,
		Container: containerInfo(),
	}
}
