// Package tui renders the interactive CLIst interface using bubbletea.
package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"clist/internal/storage"
	"clist/internal/task"
	"clist/internal/vault"
)

type animTickMsg time.Time

func (m Model) animTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

// AppView identifies the active tab.
type AppView int

const (
	ViewAll AppView = iota
	ViewToday
	ViewWaiting
	ViewSomeday
	ViewStats
	ViewArchive

	viewCount // number of views; keep last
)

// InputMode identifies the current keyboard input context.
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeAdding
	ModeEditing
	ModeSearching
	ModeConfirmDelete
	ModePickStatus
	ModePickPriority
	ModePickVault
	ModeVaultAdd
	ModeVaultConfirmRemove
	ModeVaultRemoteToken // step 1: enter existing token (blank = generate new)
	ModeVaultRemoteName  // step 2: enter vault name
)

var (
	pickerStatuses   = []task.Status{task.StatusTodo, task.StatusInProgress, task.StatusWaiting, task.StatusSomeday, task.StatusDone}
	pickerPriorities = []task.Priority{task.PriorityCritical, task.PriorityHigh, task.PriorityMedium, task.PriorityLow}
)

const statsBanner = `
██████╗  ██╗     ██╗███████╗ ████████╗
██╔═══╝  ██║     ██║██╔════╝ ╚══██╔══╝
██║      ██║     ██║███████╗    ██║
██║      ██║     ██║╚════██║    ██║
╚██████╗ ███████╗██║███████║    ██║
 ╚═════╝ ╚══════╝╚═╝╚══════╝    ╚═╝   `

// Model is the bubbletea application state.
type Model struct {
	backend      storage.Backend
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
	pickerIdx    int       // selected row index in the status/priority/vault picker
	lastSync     time.Time // UTC time of last successful full reload, used for incremental polling
	err          error     // last storage error, shown in status bar when non-nil
	animFrame    int       // incremented by animTickCmd for animations

	// token accumulated during remote vault creation (ModeVaultRemote* steps)
	remoteVaultToken string

	// ID of the task currently being edited (ModeEditing)
	editingID int64
}

// New constructs a Model bound to the given backend and vault config.
func New(backend storage.Backend, vc *vault.Config, dataDir string) Model {
	ti := textinput.New()
	ti.CharLimit = 256
	m := Model{backend: backend, vault: vc, dataDir: dataDir, input: ti}
	m.reload()
	return m
}

// Run launches the bubbletea program.
func Run(backend storage.Backend, vc *vault.Config, dataDir string) error {
	p := tea.NewProgram(New(backend, vc, dataDir), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

type tickMsg time.Time

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd { return tea.Batch(m.tickCmd(), m.animTickCmd()) }
