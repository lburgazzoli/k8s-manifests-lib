package util

import "os"

// SuppressStderr executes the given function with stderr redirected to /dev/null.
// This is useful when calling third-party libraries that write unwanted output to stderr.
// If stderr redirection fails, the function is executed normally without suppression.
//
// Example:
//
//	err := util.SuppressStderr(func() error {
//	    return someLibrary.DoWork()
//	})
func SuppressStderr(fn func() error) error {
	oldStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return fn()
	}

	os.Stderr = devNull
	defer func() {
		_ = devNull.Close()
		os.Stderr = oldStderr
	}()

	return fn()
}
