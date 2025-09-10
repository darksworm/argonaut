# Migration Plan: React+Ink → Go (Bubbletea + Lipgloss)

We break the migration into **4 phases**, each focusing on isolating
logic, setting up Go's TUI foundation, and gradually replacing React/Ink
components. This phased approach maintains clean architecture and
ensures business logic is UI-agnostic (a key "screaming architecture"
principle). Code examples illustrate how TypeScript concepts map to
Go/Bubbletea constructs.

## Phase 1: Decouple Business Logic from TS UI

**Goals:** Isolate core logic/services from React hooks/components and
prepare domain models for Go.\
- **Separate services from UI:** Move data-fetching, orchestration, and
state-change logic out of React hooks or components into standalone
modules (e.g. `services/`). Ensure React components only handle
presentation and dispatching actions.\
- **Flatten state management:** If using Context+Reducer, split complex
reducers or contexts into smaller units. Define action
payloads/interfaces cleanly; these will guide Go message types.\
- **Prepare data contracts:** Keep TypeScript interfaces but start
thinking in Go struct form. For example:\
`ts   // TypeScript   interface Task { id: string; title: string; done: boolean; }`\
becomes\
`go   // Go   type Task struct {       ID    string       Title string       Done  bool   }`\
- **Extract shared utilities:** Any helper logic or pure functions in TS
(formatting, calculations) can often be ported directly or reimplemented
in Go.

## Phase 2: Establish Go Project & Core Model

**Goals:** Create a Go codebase that mirrors the TS architecture and
introduce Bubbletea's MVU pattern.\
- **Initialize Go modules:** Create a new repo/workspace. Organize
packages by feature/domain (e.g. `pkg/model/`, `pkg/services/`,
`cmd/app/`).\
- **Replicate services:** Reimplement orchestrators, API clients, etc.,
as Go packages. Use interfaces or function signatures analogous to TS
clients. For async/streaming, use Go **channels** or Bubbletea
**Cmds**.\
`go   type dataLoadedMsg struct{ Data []Item }   func fetchDataCmd() tea.Msg {       items, _ := api.FetchItems()       return dataLoadedMsg{Data: items}   }`\
- **Define the Bubbletea model:** Create a `model` struct to hold all UI
state.\
`go   type model struct {       items []Task       cursor int   }`\
- **Centralize message definitions:** For each TS action/event, define a
corresponding Go `tea.Msg`.\
`go   type addItemMsg struct{ Item Task }`

## Phase 3: Migrate UI Components to Bubbletea + Lipgloss

**Goals:** Replace each Ink/JSX component with a Bubbletea `View()`
rendering (using Lipgloss).\
- **Map JSX to `View()`:** For each React component, write a Go function
that returns a styled string.\
`go   var headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))   func headerView(title string) string {       return headerStyle.Render(title)   }`\
- **Use Lipgloss for layout:** Use Lipgloss
`JoinHorizontal`/`JoinVertical` or style properties like `Width`,
`Padding`.\
`go   title := lipgloss.NewStyle().Bold(true).Render("Menu")   list  := lipgloss.JoinVertical(lipgloss.Left, title, itemsView)`\
- **Handle sub-components:** Use Bubbletea "bubbles" (e.g. text input,
list).\
- **Example mapping:**\
`jsx   // Ink   <Box borderStyle="round">     {items.map(item => <Text>{item.title}</Text>)}   </Box>`\
→\
`go   func todoListView(items []Task) string {       var lines []string       for _, t := range items {           lines = append(lines, t.Title)       }       list := lipgloss.JoinVertical(lipgloss.Left, lines...)       return lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Render(list)   }`

## Phase 4: Consolidate State & Final Integration

**Goals:** Tie together logic and UI into one coherent Bubbletea
program.\
- **Replace Context/Reducer with Update loop:**\
`go   type incrementMsg struct{}   func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {       switch msg.(type) {       case incrementMsg:           m.count++       }       return m, nil   }`\
- **Channel-based events (optional):** Spawn goroutines and send
`tea.Msg` via channels if needed.\
- **Example final integration:**\
`go   p := tea.NewProgram(initialModel())   if err := p.Start(); err != nil {       fmt.Fprintf(os.Stderr, "Error: %v\n", err)       os.Exit(1)   }`

**Summary:** Each phase moves the app closer to idiomatic Go: decouple
TS logic, build a Go Bubbletea skeleton, replace JSX with Lipgloss, and
unify state under Bubbletea's update loop. This preserves and improves
architecture by centralizing state and separating concerns.

