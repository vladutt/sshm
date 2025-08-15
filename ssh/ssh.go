package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"sshm/types"
	"strings"
)

func SshToServer(cfg *types.SSHConfig, options string) error {
	if cfg.Port == "" {
		cfg.Port = "22"
	}

	args := []string{
		"-p", cfg.Port,
	}

	if cfg.IdentityFile != "" {
		args = append(args, "-i", cfg.IdentityFile)
	}

	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.HostName))
	sshCmd := fmt.Sprintf("ssh %s", strings.Join(args, " "))

	if strings.Contains(options, "n") {
		// AppleScript for iTerm2
		osaScript := fmt.Sprintf(`tell application "iTerm2"
            tell current window
                set newTab to create tab with default profile
                tell current session of newTab
                    write text "%s"
                end tell
            end tell
        end tell`, sshCmd)

		cmd := exec.Command("osascript", "-e", osaScript)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if strings.Contains(options, "h") {
		// AppleScript for iTerm2
		osaScript := fmt.Sprintf(`tell application "iTerm2"
            tell current window
                tell current session
					set oldSession to id
                    set newSession to split horizontally with default profile
                    tell newSession
                        write text "%s"
                    end tell
	            	select session id oldSession
                end tell
            end tell
        end tell`, sshCmd)

		cmd := exec.Command("osascript", "-e", osaScript)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if strings.Contains(options, "v") {
		// AppleScript for iTerm2
		osaScript := fmt.Sprintf(`tell application "iTerm2"
            tell current window
                tell current session
					set oldSession to id
                    set newSession to split vertically with default profile
                    tell newSession
                        write text "%s"
                    end tell
					select session id oldSession
                end tell
            end tell
        end tell`, sshCmd)

		cmd := exec.Command("osascript", "-e", osaScript)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func RunRemoteCommand(cfg *types.SSHConfig, remoteCmd string) (string, error) {
	if cfg.Port == "" {
		cfg.Port = "22"
	}

	args := []string{
		"-p", cfg.Port,
	}
	if cfg.IdentityFile != "" {
		args = append(args, "-i", cfg.IdentityFile)
	}
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.HostName), remoteCmd)

	cmd := exec.Command("ssh", args...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}
