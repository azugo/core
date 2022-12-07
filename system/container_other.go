//go:build !linux
// +build !linux

package system

func detectContainer() (*Container, error) {
	return nil, nil
}

func detectContainerV2() (*Container, error) {
	return nil, nil
}
