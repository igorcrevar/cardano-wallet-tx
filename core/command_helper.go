package core

import (
	"bytes"
	"os"
	"os/exec"
)

type runCommandError struct {
	desc string
	base error
}

func (rce runCommandError) Error() string {
	if rce.desc != "" {
		return rce.desc
	}

	return rce.base.Error()
}

func resolveCardanoCliBinary() string {
	bin := os.Getenv("CARDANO_CLI_BINARY")
	if bin != "" {
		return bin
	}
	// fallback
	return "cardano-cli"
}

func runCommand(binary string, args []string, envVariables ...string) (string, error) {
	var (
		stdErrBuffer bytes.Buffer
		stdOutBuffer bytes.Buffer
	)

	cmd := exec.Command(binary, args...)
	cmd.Stderr = &stdErrBuffer
	cmd.Stdout = &stdOutBuffer
	cmd.Env = append(os.Environ(), envVariables...)

	err := cmd.Run()
	if stdErrBuffer.Len() > 0 {
		return "", runCommandError{desc: stdErrBuffer.String()}
	} else if err != nil {
		return "", runCommandError{base: err}
	}

	return stdOutBuffer.String(), nil
}

func createDirectoryIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// If the directory doesn't exist, create it
		return os.MkdirAll(dirPath, 0755)
	}

	return nil
}

func isFileOrDirExists(fileOrDirPath string) bool {
	_, err := os.Stat(fileOrDirPath)
	return err == nil || !os.IsNotExist(err)
}
