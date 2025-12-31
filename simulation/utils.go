package simulation

import (
	"os/exec"
	"runtime"
)

func open_url(url string) error {
	var cmd string
	var args []string
	args = []string{}

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	err := exec.Command(cmd, args...).Start()
	return err
}
