package mageutil

import (
	"fmt"
	"os"
	"os/exec"
)

type PriorityLevel int

const (
	PriorityLow PriorityLevel = iota
	PriorityBelowNormal
	PriorityNormal
	PriorityHigh
)

func RunWithPriority(priority PriorityLevel, env map[string]string, cmd string, args ...string) error {
	execCmd := exec.Command(cmd, args...)
	execCmd.Env = os.Environ()
	for k, v := range env {
		execCmd.Env = append(execCmd.Env, k+"="+v)
	}
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Start(); err != nil {
		return err
	}

	pid := execCmd.Process.Pid
	if err := SetPriority(pid, priority); err != nil {
		PrintYellow(fmt.Sprintf("Failed to set priority for PID %d: %v", pid, err))
	}

	return execCmd.Wait()
}
