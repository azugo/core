//go:build !linux
// +build !linux

package system

func detectContainer() *Container {
	return nil
}
