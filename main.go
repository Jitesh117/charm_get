package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	state     int // 0 for input, 1 for installing
	textInput textinput.Model
	packages  []string
	index     int
	width     int
	height    int
	spinner   spinner.Model
	progress  progress.Model
	done      bool
	project   string
	err       error
}

var (
	currentPkgNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle           = lipgloss.NewStyle().Margin(1, 2)
	checkMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

func newModel() model {
	ti := textinput.New()
	ti.Placeholder = "github.com/username/project_name"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return model{
		state:     0, // start with text input
		textInput: ti,
		packages:  getPackages(),
		spinner:   s,
		progress:  p,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case 0: // Input state
			switch msg.String() {
			case "enter":
				m.project = m.textInput.Value()
				createGoModFile(m.project) // Create go.mod file when the project name is entered
				m.state = 1                // Move to package installation state
				return m, tea.Batch(downloadAndInstall(m.packages[m.index]), m.spinner.Tick)
			case "ctrl+c", "esc":
				return m, tea.Quit
			}
		case 1: // Installation state
			switch msg.String() {
			case "ctrl+c", "esc", "q":
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case installedPkgMsg:
		if msg.Err != nil {
			fmt.Printf("Failed to install %s: %v\n", msg.Package, msg.Err)
		}
		//   else {
		// 	fmt.Printf("%s installed successfully\n", msg.Package)
		// }
		pkg := m.packages[m.index]
		if m.index >= len(m.packages)-1 {
			m.done = true
			return m, tea.Sequence(
				tea.Printf("%s %s", checkMark, pkg), // print the last success message
				tea.Quit,                            // exit the program
			)
		}

		m.index++
		progressCmd := m.progress.SetPercent(float64(m.index) / float64(len(m.packages)))

		return m, tea.Batch(
			progressCmd,
			tea.Printf("%s %s", checkMark, pkg),
			downloadAndInstall(m.packages[m.index]),
		)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if newModel, ok := newModel.(progress.Model); ok {
			m.progress = newModel
		}
		return m, cmd
	}

	// Handle text input updates
	m.textInput, _ = m.textInput.Update(msg)
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case 0: // Project name input view
		return fmt.Sprintf(
			"Enter the project name\n\n%s\n\n%s",
			m.textInput.View(),
			"(esc to quit)",
		) + "\n"
	case 1: // Package installation view
		if m.done {
			return doneStyle.Render(fmt.Sprintf("Done! Installed %d packages.\n", len(m.packages)))
		}

		pkgCount := fmt.Sprintf(" %d/%d", m.index, len(m.packages))
		spin := m.spinner.View() + " "
		prog := m.progress.View()
		cellsAvail := max(0, m.width-lipgloss.Width(spin+prog+pkgCount))

		pkgName := currentPkgNameStyle.Render(m.packages[m.index])
		info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render("Installing " + pkgName)

		cellsRemaining := max(0, m.width-lipgloss.Width(spin+info+prog+pkgCount))
		gap := strings.Repeat(" ", cellsRemaining)

		return spin + info + gap + prog + pkgCount
	}
	return ""
}

type installedPkgMsg struct {
	Package string
	Err     error
}

func downloadAndInstall(pkg string) tea.Cmd {
	return func() tea.Msg {
		// Create the command to install the package
		cmd := exec.Command("go", "get", pkg)

		// Run the command
		err := cmd.Run()

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(500)))

		return installedPkgMsg{
			Package: pkg,
			Err:     err,
		}
	}
}

func createGoModFile(project string) {
	// Create the go.mod file for the project
	modContent := fmt.Sprintf("module %s\n\ngo 1.20", project)
	err := os.WriteFile("go.mod", []byte(modContent), 0644)
	if err != nil {
		fmt.Println("Error creating go.mod:", err)
	}
}

func getPackages() []string {
	return []string{
		"github.com/charmbracelet/bubbles",
		"github.com/charmbracelet/bubbles/textinput",
		"github.com/charmbracelet/bubbles/progress",
		"github.com/charmbracelet/bubbles/list",
		"github.com/charmbracelet/bubbles/textarea",
		"github.com/charmbracelet/lipgloss",
		"github.com/charmbracelet/bubbletea",
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
