package pl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const defaultShell = "/bin/sh"

type Builder struct {
	Expander

	cmd string
	env []string
}

func Build(args []string) (*Builder, error) {
	e, err := Parse(args[1:])
	if err != nil {
		return nil, err
	}
	b := Builder{
		Expander: e,
		cmd:      args[0],
		env:      os.Environ(),
	}
	return &b, nil
}

func (b Builder) Dump(xs []string) (string, error) {
	as, err := b.Expand(xs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s", b.cmd, strings.Join(as, " ")), nil
}

func (b Builder) Build(xs []string, env, shell bool) (*exec.Cmd, error) {
	as, err := b.Expand(xs)
	if err != nil {
		return nil, err
	}
	var cmd *exec.Cmd
	if shell {
		cmd = shellCommand(b.cmd, as)
	} else {
		cmd = subCommand(b.cmd, as)
	}
	if env && len(b.env) > 0 {
		cmd.Env = append(cmd.Env, b.env...)
	}
	return cmd, nil
}

func shellCommand(name string, args []string) *exec.Cmd {
	shell, ok := os.LookupEnv("SHELL")
	if !ok || shell == "" {
		shell = defaultShell
	}
	cmd := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	return exec.Command(shell, "-c", cmd)
}

func subCommand(name string, args []string) *exec.Cmd {
	return exec.Command(name, args...)
}
