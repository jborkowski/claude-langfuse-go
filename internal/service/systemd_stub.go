//go:build !linux

package service

import "fmt"

func (i *Installer) installLinux() error {
	return fmt.Errorf("Linux service installation not available on this platform")
}

func (i *Installer) uninstallLinux() error {
	return fmt.Errorf("Linux service uninstallation not available on this platform")
}
