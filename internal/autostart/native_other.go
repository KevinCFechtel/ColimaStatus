//go:build !darwin

package autostart

import "errors"

type NativeController struct{}

func NewNativeController() Controller {
	return NativeController{}
}

func (NativeController) Status() (Status, error) {
	return Unsupported, nil
}

func (NativeController) SetEnabled(bool) (Status, error) {
	return Unsupported, errors.New("Autostart wird auf diesem Betriebssystem nicht unterstützt")
}

func (NativeController) OpenSettings() error {
	return errors.New("Autostart wird auf diesem Betriebssystem nicht unterstützt")
}
