package mageutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/openimsdk/gomake/internal/priority"
	"github.com/openimsdk/gomake/internal/util"
)

type Cmd struct {
	execCmd *exec.Cmd

	env map[string]string

	priority *priority.Level
}

func NewCmd(command string) *Cmd {
	cmd := &Cmd{execCmd: exec.Command(command)}
	return cmd
}

func (c *Cmd) WithArgs(args ...string) *Cmd {
	c.execCmd.Args = append(c.execCmd.Args, args...)
	return c
}

func (c *Cmd) WithEnv(env map[string]string) *Cmd {
	if len(env) == 0 {
		return c
	}
	if c.env == nil {
		c.env = make(map[string]string, len(env))
	}
	for k, v := range env {
		c.env[k] = v
	}
	return c
}

func (c *Cmd) WithDir(dir string) *Cmd {
	c.execCmd.Dir = strings.TrimSpace(dir)
	return c
}

func (c *Cmd) WithPriority(priority priority.Level) *Cmd {
	c.priority = &priority
	return c
}

func (c *Cmd) WithStdin(stdin io.Reader) *Cmd {
	c.execCmd.Stdin = stdin
	return c
}

func (c *Cmd) WithStdout(stdout io.Writer) *Cmd {
	c.execCmd.Stdout = stdout
	return c
}

func (c *Cmd) WithStderr(stderr io.Writer) *Cmd {
	c.execCmd.Stderr = stderr
	return c
}

func (c *Cmd) Start() error {
	c.execCmd.Env = append(os.Environ(), util.FlattenEnvs(c.env)...)

	if err := c.execCmd.Start(); err != nil {
		return err
	}

	if c.priority != nil && c.execCmd.Process != nil {
		if err := priority.Set(c.execCmd.Process.Pid, *c.priority); err != nil {
			PrintYellow(fmt.Sprintf("Failed to set priority for PID %d: %v", c.execCmd.Process.Pid, err))
		}
	}

	return nil
}

func (c *Cmd) Wait() error {
	return c.execCmd.Wait()
}

func (c *Cmd) Run() error {
	err := c.Start()
	if err != nil {
		return err
	}

	return c.Wait()
}

func (c *Cmd) String() string {
	return c.execCmd.String()
}

func (c *Cmd) Output() ([]byte, error) {
	if c.execCmd.Stdout != nil {
		return nil, fmt.Errorf("stdout already set")
	}
	buf := bytes.NewBuffer(nil)
	c.execCmd.Stdout = buf
	err := c.Run()
	return buf.Bytes(), err
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	if c.execCmd.Stdout != nil || c.execCmd.Stderr != nil {
		return nil, fmt.Errorf("stdout or stderr already set")
	}
	buf := bytes.NewBuffer(nil)
	c.execCmd.Stdout = buf
	c.execCmd.Stderr = buf
	err := c.Run()
	return buf.Bytes(), err
}
