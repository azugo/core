package system

import "os"

type Info struct {
	Hostname  string
	Container *Container
}

func (i Info) IsKubernetes() bool {
	return i.IsContainer() && i.Container.Kubernetes != nil
}

func (i Info) IsContainer() bool {
	return i.Container != nil
}

// Info collects system information.
func CollectInfo() *Info {
	// TODO: use FQDN to get hostname
	// Ignore error
	hostname, _ := os.Hostname()

	return &Info{
		Hostname:  hostname,
		Container: containerInfo(),
	}
}
