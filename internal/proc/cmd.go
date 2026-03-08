package proc

import (
	"os"
	"os/exec"
	"sync"
)

type Cmd struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error
	once sync.Once
}

func Command(command string) *Cmd {
	cmd := shellCmd(command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return &Cmd{
		cmd:  cmd,
		done: make(chan struct{}),
	}
}

func (c *Cmd) SetDir(dir string) {
	c.cmd.Dir = dir
}

func (c *Cmd) Start() error {
	if err := c.cmd.Start(); err != nil {
		c.once.Do(func() { close(c.done) })
		return err
	}
	go func() {
		c.err = c.cmd.Wait()
		c.once.Do(func() { close(c.done) })
	}()
	return nil
}

func (c *Cmd) Done() <-chan struct{} {
	return c.done
}

func (c *Cmd) Err() error {
	<-c.done
	return c.err
}

func (c *Cmd) Pid() int {
	if c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}
