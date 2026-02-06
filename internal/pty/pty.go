package pty

import (
	"fmt"
	"os"
	"os/exec"
	"github.com/creack/pty"
)

// Service handles the PTY session
type Service struct {
	ptyFile *os.File
	cmd     *exec.Cmd
}

// NewService creates a new PTY service instance
func NewService() *Service {
	return &Service{}
}

// Start launches the shell in a PTY and returns the file descriptor for IO
func (s *Service) Start() (*os.File, error) {
	// Determine shell to use (zsh > bash > sh)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	c := exec.Command(shell)
	
	// Create PTY
	f, err := pty.Start(c)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	s.ptyFile = f
	s.cmd = c

	return f, nil
}

// Resize updates the terminal size
func (s *Service) Resize(rows, cols uint16) error {
	if s.ptyFile == nil {
		return fmt.Errorf("pty not started")
	}

	sz := &pty.Winsize{
		Rows: rows,
		Cols: cols,
		X:    0,
		Y:    0,
	}

	if err := pty.Setsize(s.ptyFile, sz); err != nil {
		return fmt.Errorf("failed to resize pty: %w", err)
	}
	return nil
}

// Close terminates the PTY session
func (s *Service) Close() error {
	if s.ptyFile != nil {
		s.ptyFile.Close() // Close the PTY file
	}
	if s.cmd != nil && s.cmd.Process != nil {
		// Try to kill the process if it's still running
		s.cmd.Process.Kill() 
	}
	return nil
}

// Default size
func (s *Service) SetDefaultSize() error {
	return s.Resize(24, 80)
}
