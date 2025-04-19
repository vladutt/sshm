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

	// Setup table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "IP", "User", "Port", "Identity"})

	for i, cfg := range configs {
        if search != "" && !strings.Contains(strings.ToLower(cfg.Host), search) {
            continue
        }

        i = i+1;

	    filteredRows = append(filteredRows, TableRow{ID: i, Config: cfg})

		table.Append([]string{
			fmt.Sprintf("%d", i),
			cfg.Host,
			cfg.HostName,
			cfg.User,
			orDefault(cfg.Port, "22"),
			cfg.IdentityFile,
		})
	}

    table.Render()

    serverID := selectServerPrompt()

    currentServer := getConfigByID(serverID, filteredRows)

    sshErr := sshToServer(currentServer)
    if sshErr != nil {
        fmt.Println("Eroare la SSH:", sshErr)
    }
}

func selectServerPrompt() int {
    reader := bufio.NewReader(os.Stdin)

    fmt.Print("Select a server: ")
    serverSelected, _ := reader.ReadString('\n')
    serverSelected = strings.TrimSpace(serverSelected)

    serverSelectedNumber, err := strconv.Atoi(serverSelected)
    if err != nil {
        return 0
    }

    return serverSelectedNumber
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

func sshToServer(cfg *SSHConfig) error {
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

    cmd := exec.Command("ssh", args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Run()
}