package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sshm/ssh"
	"sshm/update"
	"strconv"
	"strings"

	"sshm/types"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table        table.Model
	textInput    textinput.Model
	focus        string
	selectedRows map[int]bool
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
			canceled = true
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
		case " ":

			if !*hasRunCommands || m.focus == "input" {
				var inputCmd tea.Cmd
				m.textInput, inputCmd = m.textInput.Update(msg)
				return m, inputCmd
			}

			idx := m.table.Cursor()
			m.selectedRows[idx] = !m.selectedRows[idx]

			newRows := []table.Row{}
			for idx, r := range filteredRows {
				mark := "[ ]"
				if m.selectedRows[idx] {
					mark = "[x]"
				}
				newRow := table.Row{
					mark,
					strconv.Itoa(r.ID),
					r.Config.Host,
					r.Config.HostName,
					r.Config.User,
					orDefault(r.Config.Port, "22"),
					r.Config.IdentityFile,
				}

				newRows = append(newRows, newRow)
			}

			m.table.SetRows(newRows)

			return m, nil
		case "enter":

			if *hasRunCommands {
				commandToRun = m.textInput.Value()
				m.textInput.SetValue("")
				return m, tea.Quit
			}

			selected := ""

			if m.focus == "input" {
				selected = m.textInput.Value()
				m.textInput.SetValue("")

			} else {
				if len(m.table.SelectedRow()) == 0 {
					fmt.Println("Nothing selected")
					return m, nil
				}
				selected = m.table.SelectedRow()[1]
			}

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

	rendered := m.table.View()

	return fmt.Sprintf(
		"%s\n\n%s\n\n(press TAB to switch the focus)",
		baseStyle.Render(rendered),
		m.textInput.View(),
	)
}

type TableRow struct {
	ID     int
	Config types.SSHConfig
}

var filteredRows []TableRow
var selectedServer int
var hasRunCommands *bool
var canceled bool
var commandToRun string

func main() {

	columns := []table.Column{
		{Title: "", Width: 4},
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 25},
		{Title: "IP", Width: 35},
		{Title: "User", Width: 10},
		{Title: "Port", Width: 10},
		{Title: "Identity", Width: 50},
	}

	search := flag.String("s", "", "Text to search")
	hasRunCommands = flag.Bool("r", false, "Run Command flag")
	updateBinary := flag.Bool("update", false, "Update sshm to last version")
	versionDisplay := flag.Bool("v", false, "Display sshm version")

	flag.Parse()

	if *updateBinary {
		update.DoSelfUpdate()
		return
	}

	if *versionDisplay {
		update.DisplayCurrentVersion()
		return
	}

	if *search != "" {
		fmt.Println("Caut:", *search)
	}

	if len(*search) == 0 && !*hasRunCommands && len(os.Args) > 1 {
		search = &os.Args[1]
	}

	usr, _ := user.Current()
	configPath := filepath.Join(usr.HomeDir, ".ssh", "config")

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("Could not open SSH config file: %v\n", err)
		return
	}

	defer file.Close()

	var configs []types.SSHConfig
	var current types.SSHConfig

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
			current = types.SSHConfig{Host: value}

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
		if *search != "" && !strings.Contains(strings.ToLower(cfg.Host), *search) {
			continue
		}

		if len(cfg.User) == 0 {
			continue
		}

		filteredRows = append(filteredRows, TableRow{ID: ID, Config: cfg})

		row := table.Row{}

		if *hasRunCommands {
			row = append(row, "[ ]")
		} else {
			row = append(row, "")
		}
		fmt.Println(cfg.IdentityFile)
		row = append(row, fmt.Sprintf("%d", ID), cfg.Host, cfg.HostName, cfg.User, orDefault(cfg.Port, "22"), cfg.IdentityFile)

		rows = append(rows, row)

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
	if *hasRunCommands {
		ti.Placeholder = "Type the command to run"
		ti.CharLimit = 100
	} else {
		ti.Placeholder = "Type the ID of a server"
		ti.CharLimit = 3
	}

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

	m := model{t, ti, "table", make(map[int]bool)}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if canceled {
		return
	}

	if *hasRunCommands {
		for ID := range m.selectedRows {
			currentServer := getConfigByID(ID+1, filteredRows)

			out, err := ssh.RunRemoteCommand(currentServer, commandToRun)
			if err != nil {
				fmt.Println("Eroare:", err)
			}
			fmt.Println("Output:\n", out)
		}

		return
	}

	if selectedServer > 0 {
		currentServer := getConfigByID(selectedServer, filteredRows)
		sshErr := ssh.SshToServer(currentServer, "")
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

func getConfigByID(id int, filteredRows []TableRow) *types.SSHConfig {
	for _, row := range filteredRows {
		if row.ID == id {
			return &row.Config
		}
	}
	return nil
}
