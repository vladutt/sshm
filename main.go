package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table     table.Model
	textInput textinput.Model
	focus     string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.focus == "table" {
				m.focus = "input"
				m.table.Blur()
				m.textInput.Focus()
			} else {
				m.focus = "table"
				m.textInput.Blur()
				m.table.Focus()
			}
		case "enter":
			selected := ""

			if m.focus == "input" {
				selected = m.textInput.Value()
				m.textInput.SetValue("")
			} else {
				if len(m.table.SelectedRow()) == 0 {
					fmt.Println("Nothing selected")
					return m, nil
				}
				selected = m.table.SelectedRow()[0]
			}

			fmt.Println(selected)

			ID, err := strconv.Atoi(selected)
			if err != nil {
				fmt.Println("ID invalid:", selected)
				return m, tea.Quit
			}

			selectedServer = ID

			return m, tea.Quit
		}
	}
	if m.focus == "table" {
		m.table, cmd = m.table.Update(msg)
	} else {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf("%s\n\n%s\n\n(press TAB to switch the focus)", m.table.View(), m.textInput.View())
}

type SSHConfig struct {
	Host         string
	HostName     string
	User         string
	Port         string
	IdentityFile string
}

type TableRow struct {
	ID     int
	Config SSHConfig
}

var filteredRows []TableRow
var selectedServer int

func main() {

	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 25},
		{Title: "IP", Width: 35},
		{Title: "User", Width: 10},
		{Title: "Port", Width: 10},
		{Title: "Identity", Width: 50},
	}

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

	rows := []table.Row{}

	ID := 1
	for _, cfg := range configs {
		if search != "" && !strings.Contains(strings.ToLower(cfg.Host), search) {
			continue
		}

		if len(cfg.User) == 0 {
			continue
		}

		filteredRows = append(filteredRows, TableRow{ID: ID, Config: cfg})

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", ID),
			cfg.Host,
			cfg.HostName,
			cfg.User,
			orDefault(cfg.Port, "22"),
			cfg.IdentityFile,
		})

		ID = ID + 1
	}

	var tableHeight = 20
	if len(rows) < 20 {
		tableHeight = len(rows) + 2 // to have the last row empty
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	ti := textinput.New()
	ti.Width = 30
	ti.Placeholder = "Type the ID of a server"
	ti.CharLimit = 3

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t, ti, "table"}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if selectedServer > 0 {
		currentServer := getConfigByID(selectedServer, filteredRows)
		sshErr := sshToServer(currentServer, "")
		if sshErr != nil {
			fmt.Println("Error SSH:", sshErr)
		}
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
