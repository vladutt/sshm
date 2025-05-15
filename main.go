package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"os/exec"
	"path/filepath"
	"strings"
	"strconv"
	"github.com/olekukonko/tablewriter"
	"regexp"
)

type SSHConfig struct {
	Host        string
	HostName    string
	User        string
	Port        string
	IdentityFile string
}

type TableRow struct {
	ID     int
	Config SSHConfig
}

func main() {
	search := ""
	if len(os.Args) > 1 {
		search = strings.ToLower(os.Args[1])
	}

	usr, _ := user.Current()
	configPath := filepath.Join(usr.HomeDir, ".ssh", "config")

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("Could not open SSH config file: %v\n", err)
		return
	}

	defer file.Close()

	var configs []SSHConfig
    var filteredRows []TableRow
	var current SSHConfig

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		key := strings.ToLower(fields[0])
		value := strings.Join(fields[1:], " ")

		switch key {
            case "host":
                if current.Host != "" {
                    configs = append(configs, current)
                }
                current = SSHConfig{Host: value}

            case "hostname":
                current.HostName = value
            case "user":
                current.User = value
            case "port":
                current.Port = value
            case "identityfile":
                current.IdentityFile = value
		}
	}

	if current.Host != "" {
		configs = append(configs, current)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "IP", "User", "Port", "Identity"})

    ID := 1
	for _, cfg := range configs {
        if search != "" && !strings.Contains(strings.ToLower(cfg.Host), search) {
            continue
        }

	    filteredRows = append(filteredRows, TableRow{ID: ID, Config: cfg})

		table.Append([]string{
			fmt.Sprintf("%d", ID),
			cfg.Host,
			cfg.HostName,
			cfg.User,
			orDefault(cfg.Port, "22"),
			cfg.IdentityFile,
		})

        ID = ID+1
	}

    table.Render()

    if len(filteredRows) == 0 {
        return
    }

    serverID, options := selectServerPrompt()

    currentServer := getConfigByID(serverID, filteredRows)

    sshErr := sshToServer(currentServer, options)
    if sshErr != nil {
        fmt.Println("Eroare la SSH:", sshErr)
    }
}

func selectServerPrompt() (int, string) {
    reader := bufio.NewReader(os.Stdin)

    fmt.Print("Select a server: ")
    serverSelected, _ := reader.ReadString('\n')
    serverSelected = strings.TrimSpace(serverSelected)

    options := ""

    if _, err := strconv.Atoi(serverSelected); err != nil {
        re := regexp.MustCompile(`([a-zA-Z]+)(\d+)`)
        matches := re.FindStringSubmatch(serverSelected)

        options = matches[1]
    }

    serverSelectedNumber, err := strconv.Atoi(strings.TrimPrefix(serverSelected, options))

    if err != nil {
        return 0, ""
    }

    return serverSelectedNumber, options
}

func orDefault(value, fallback string) string {
    if value == "" {
        return fallback
    }
    return value
}

func getConfigByID(id int, filteredRows []TableRow) *SSHConfig {
    for _, row := range filteredRows {
        if row.ID == id {
            return &row.Config
        }
    }
    return nil
}

func sshToServer(cfg *SSHConfig, options string) error {
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
                    set newSession to split horizontally with default profile
                    tell newSession
                        write text "%s"
                    end tell
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
                    set newSession to split vertically with default profile
                    tell newSession
                        write text "%s"
                    end tell
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