//go:build !darwin

package core

import "os"

func renameExclusive(from, to string) error {
	if err := os.Link(from, to); err != nil {
		return err
	}
	if err := os.Remove(from); err != nil {
		_ = os.Remove(to)
		return err
	}
	return nil
}
