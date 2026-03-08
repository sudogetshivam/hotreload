//go:build !windows

package proc

import (
	"os/exec"
	"syscall"
)

func shellCmd(command string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func (c *Cmd) Terminate() {
	if c.cmd.Process == nil {
		return
	}
	// Send SIGTERM to the entire process group for graceful shutdown
	syscall.Kill(-c.cmd.Process.Pid, syscall.SIGTERM)
}

func (c *Cmd) Kill() {
	if c.cmd.Process == nil {
		return
	}
	// Send SIGKILL to the entire process group for force kill
	syscall.Kill(-c.cmd.Process.Pid, syscall.SIGKILL)
}
