//go:build !darwin

package service

import "fmt"

func (i *Installer) installMacOS() error {
	return fmt.Errorf("macOS service installation not available on this platform")
}

func (i *Installer) uninstallMacOS() error {
	return fmt.Errorf("macOS service uninstallation not available on this platform")
}
