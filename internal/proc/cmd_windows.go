//go:build windows

package proc

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func shellCmd(command string) *exec.Cmd {
	cmd := exec.Command("cmd", "/C", strings.ReplaceAll(command, "/", `\`))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd
}

func (c *Cmd) Terminate() {
	if c.cmd.Process == nil {
		return
	}
	// Try graceful termination first using taskkill /T (terminate tree) without /F (force)
	// This gives the process a chance to clean up resources
	kill := exec.Command("taskkill", "/T", "/PID", strconv.Itoa(c.cmd.Process.Pid))
	kill.Run()
}

func (c *Cmd) Kill() {
	if c.cmd.Process == nil {
		return
	}
	// Force kill the entire process tree
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(c.cmd.Process.Pid))
	kill.Run()
}
