package lib

import (
	"os/exec"
	"strings"
)

func DetectMimetype(path string) (string, error) {
	cmd := exec.Command("mimetype", "-b", path)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", &MimetypeError{
				Err:    err,
				Stderr: string(exitErr.Stderr),
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

type MimetypeError struct {
	Err    error
	Stderr string
}

func (e *MimetypeError) Error() string {
	return e.Stderr
}

func (e *MimetypeError) Unwrap() error {
	return e.Err
}
