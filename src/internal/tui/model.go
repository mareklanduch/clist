// Package tui renders the interactive CLIst interface using bubbletea.
package tui

import (
	"database/sql"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"clist/internal/storage"
	"clist/internal/task"
	"clist/internal/vault"
)

// AppView identifies the active tab.
type AppView int

const (
	ViewAll AppView = iota
	ViewToday
	ViewWaiting
	ViewSomeday
	ViewStats
	ViewArchive
)

// InputMode identifies the current keyboard input context.
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeAdding
	ModeSearching
	ModeConfirmDelete
	ModePickStatus
	ModePickPriority
	ModePickVault
	ModeVaultAdd
	ModeVaultConfirmRemove
)

var (
	pickerStatuses   = []task.Status{task.StatusTodo, task.StatusInProgress, task.StatusWaiting, task.StatusSomeday, task.StatusDone}
	pickerPriorities = []task.Priority{task.PriorityCritical, task.PriorityHigh, task.PriorityMedium, task.PriorityLow}
)

const statsBanner = ` ██████╗ ██╗     ██╗███████╗ ████████╗
██╔════╝ ██║     ██║██╔════╝ ╚══██╔══╝
██║      ██║     ██║███████╗    ██║
██║      ██║     ██║╚════██╗    ██║
╚██████╗ ███████╗██║███████╔╝   ██║
 ╚═════╝ ╚══════╝╚═╝╚══════╝    ╚═╝   `

// Model is the bubbletea application state.
type Model struct {
	db           *sql.DB
	vault        *vault.Config
	dataDir      string
	width        int
	height       int
	view         AppView
	tasks        []task.Task
	filtered     []task.Task
	counts       [5]int // cached view badge counts; refreshed on reload
	selected     int
	scrollOffset int
	helpScroll   int
	mode         InputMode
	input        textinput.Model
	search       string
	status       string
	showHelp     bool
	stats        []storage.DayStat
	pickerIdx    int   // selected row index in the status/priority picker
	err          error // last storage error, shown in status bar when non-nil
}

// New constructs a Model bound to the given DB and vault config.
func New(db *sql.DB, vc *vault.Config, dataDir string) Model {
	ti := textinput.New()
	ti.CharLimit = 256
	m := Model{db: db, vault: vc, dataDir: dataDir, input: ti}
	m.reload()
	return m
}

// Run launches the bubbletea program.
func Run(db *sql.DB, vc *vault.Config, dataDir string) error {
	p := tea.NewProgram(New(db, vc, dataDir), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd { return tickCmd() }
