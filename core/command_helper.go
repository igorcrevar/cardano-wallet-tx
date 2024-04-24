package core

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
)

const MainnetMagic = uint(764824073)

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

func isFileOrDirExists(fileOrDirPath string) bool {
	_, err := os.Stat(fileOrDirPath)
	return err == nil || !os.IsNotExist(err)
}

func getTestNetMagicArgs(testnetMagic uint) []string {
	if testnetMagic == 0 || testnetMagic == MainnetMagic {
		return []string{"--mainnet"}
	}

	return []string{"--testnet-magic", strconv.FormatUint(uint64(testnetMagic), 10)}
}
