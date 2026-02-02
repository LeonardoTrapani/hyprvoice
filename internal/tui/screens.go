package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listScreen struct {
	state   *wizardState
	title   string
	desc    []string
	list    list.Model
	footer  string
	errText string
	onPick  func(optionItem) screen
	onBack  func() screen
}

func newListScreen(state *wizardState, title string, desc []string, items []optionItem, onPick func(optionItem) screen, onBack func() screen) *listScreen {
	delegate := list.NewDefaultDelegate()
	l := list.New(itemsToList(items), delegate, 0, 0)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.Title = title
	return &listScreen{
		state:  state,
		title:  title,
		desc:   desc,
		list:   l,
		footer: "enter select • esc back • / filter",
		onPick: onPick,
		onBack: onBack,
	}
}

func (s *listScreen) Init() tea.Cmd {
	return nil
}

func (s *listScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		extraFooterLines := strings.Count(s.footer, "\n")
		s.list.SetSize(msg.Width-4, msg.Height-8-extraFooterLines)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if s.list.FilterState() != list.Filtering {
				if item, ok := s.list.SelectedItem().(optionItem); ok {
					if item.disabled {
						s.errText = "That option isn't available in this environment."
						break
					}
					if s.onPick != nil {
						return s.onPick(item), nil
					}
				}
			}
		case "esc", "q":
			if s.list.FilterState() == list.Unfiltered {
				if s.onBack != nil {
					return s.onBack(), nil
				}
			}
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *listScreen) View() string {
	header := renderHeader(s.title, s.desc, s.errText)
	footer := renderFooter(s.footer, s.list.FilterState() != list.Unfiltered)
	return header + s.list.View() + "\n" + footer
}

type confirmScreen struct {
	state   *wizardState
	title   string
	desc    []string
	list    list.Model
	footer  string
	onYes   func() screen
	onNo    func() screen
	onBack  func() screen
	errText string
}

func newConfirmScreen(state *wizardState, title string, desc []string, yesLabel, yesDesc, noLabel, noDesc string, onYes func() screen, onNo func() screen, onBack func() screen) *confirmScreen {
	items := []optionItem{
		{title: yesLabel, desc: yesDesc, value: "yes"},
		{title: noLabel, desc: noDesc, value: "no"},
	}
	footer := "enter select • esc cancel"
	if onBack != nil {
		footer = "enter select • esc back"
	}
	delegate := list.NewDefaultDelegate()
	l := list.New(itemsToList(items), delegate, 0, 0)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.Title = title
	return &confirmScreen{state: state, title: title, desc: desc, list: l, footer: footer, onYes: onYes, onNo: onNo, onBack: onBack}
}

func (s *confirmScreen) Init() tea.Cmd { return nil }

func (s *confirmScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width-4, msg.Height-8)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := s.list.SelectedItem().(optionItem); ok {
				if item.value == "yes" && s.onYes != nil {
					return s.onYes(), nil
				}
				if item.value == "no" && s.onNo != nil {
					return s.onNo(), nil
				}
			}
		case "esc", "q":
			if s.onBack != nil {
				return s.onBack(), nil
			}
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *confirmScreen) View() string {
	header := renderHeader(s.title, s.desc, s.errText)
	footer := renderFooter(s.footer, false)
	return header + s.list.View() + "\n" + footer
}

type inputScreen struct {
	state      *wizardState
	title      string
	desc       []string
	input      textinput.Model
	footer     string
	errText    string
	onSubmit   func(string) screen
	onCancel   func() screen
	validateFn func(string) error
}

func newInputScreen(state *wizardState, title string, desc []string, value string, placeholder string, password bool, validateFn func(string) error, onSubmit func(string) screen, onCancel func() screen) *inputScreen {
	input := textinput.New()
	input.SetValue(value)
	input.Placeholder = placeholder
	if password {
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '*'
	}
	input.Focus()
	input.CharLimit = 0
	return &inputScreen{
		state:      state,
		title:      title,
		desc:       desc,
		input:      input,
		footer:     "enter save • esc back",
		onSubmit:   onSubmit,
		onCancel:   onCancel,
		validateFn: validateFn,
	}
}

func (s *inputScreen) Init() tea.Cmd { return textinput.Blink }

func (s *inputScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(s.input.Value())
			if s.validateFn != nil {
				if err := s.validateFn(value); err != nil {
					s.errText = err.Error()
					break
				}
			}
			if s.onSubmit != nil {
				return s.onSubmit(value), nil
			}
		case "esc", "q":
			if s.onCancel != nil {
				return s.onCancel(), nil
			}
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *inputScreen) View() string {
	header := renderHeader(s.title, s.desc, s.errText)
	footer := renderFooter(s.footer, false)
	return header + s.input.View() + "\n\n" + footer
}

type multiSelectScreen struct {
	state      *wizardState
	title      string
	desc       []string
	list       list.Model
	footer     string
	errText    string
	onSubmit   func([]toggleItem) screen
	onCancel   func() screen
	requireOne bool
}

func newMultiSelectScreen(state *wizardState, title string, desc []string, items []toggleItem, requireOne bool, onSubmit func([]toggleItem) screen, onCancel func() screen) *multiSelectScreen {
	delegate := list.NewDefaultDelegate()
	l := list.New(toggleItemsToList(items), delegate, 0, 0)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.Title = title
	return &multiSelectScreen{
		state:      state,
		title:      title,
		desc:       desc,
		list:       l,
		footer:     "space toggle • enter save • esc back • / filter",
		onSubmit:   onSubmit,
		onCancel:   onCancel,
		requireOne: requireOne,
	}
}

func (s *multiSelectScreen) Init() tea.Cmd { return nil }

func (s *multiSelectScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.list.SetSize(msg.Width-4, msg.Height-8)
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			idx := s.list.Index()
			items := s.list.Items()
			if idx >= 0 && idx < len(items) {
				if item, ok := items[idx].(toggleItem); ok {
					item.selected = !item.selected
					items[idx] = item
					s.list.SetItems(items)
				}
			}
		case "enter":
			items := listToToggleItems(s.list.Items())
			if s.requireOne {
				has := false
				for _, item := range items {
					if item.selected {
						has = true
						break
					}
				}
				if !has {
					s.errText = "Select at least one option to continue."
					break
				}
			}
			if s.onSubmit != nil {
				return s.onSubmit(items), nil
			}
		case "esc", "q":
			if s.list.FilterState() == list.Unfiltered {
				if s.onCancel != nil {
					return s.onCancel(), nil
				}
			}
		}
	}

	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *multiSelectScreen) View() string {
	header := renderHeader(s.title, s.desc, s.errText)
	footer := renderFooter(s.footer, s.list.FilterState() != list.Unfiltered)
	return header + s.list.View() + "\n" + footer
}

type infoScreen struct {
	state  *wizardState
	title  string
	desc   []string
	footer string
	next   func() screen
	back   func() screen
}

func newInfoScreen(state *wizardState, title string, desc []string, next func() screen, back func() screen) *infoScreen {
	return &infoScreen{state: state, title: title, desc: desc, footer: "enter continue • esc back", next: next, back: back}
}

func (s *infoScreen) Init() tea.Cmd {
	return nil
}

func (s *infoScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if s.next != nil {
				return s.next(), nil
			}
		case "esc", "q":
			if s.back != nil {
				return s.back(), nil
			}
		}
	}
	return s, nil
}

func (s *infoScreen) View() string {
	header := renderHeader(s.title, s.desc, "")
	footer := renderFooter(s.footer, false)
	return header + "\n" + footer
}

type formField struct {
	key       string
	label     string
	desc      string
	input     textinput.Model
	validate  func(string) error
	required  bool
	sensitive bool
}

type formScreen struct {
	state    *wizardState
	title    string
	desc     []string
	fields   []formField
	focused  int
	footer   string
	errText  string
	onSubmit func(map[string]string) screen
	onCancel func() screen
}

func newFormScreen(state *wizardState, title string, desc []string, fields []formField, onSubmit func(map[string]string) screen, onCancel func() screen) *formScreen {
	if len(fields) > 0 {
		fields[0].input.Focus()
	}
	return &formScreen{state: state, title: title, desc: desc, fields: fields, onSubmit: onSubmit, onCancel: onCancel}
}

func (s *formScreen) Init() tea.Cmd { return textinput.Blink }

func (s *formScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if s.onCancel != nil {
				return s.onCancel(), nil
			}
		case "tab", "down":
			s.moveFocus(1)
		case "shift+tab", "up":
			s.moveFocus(-1)
		case "enter":
			if s.focused == len(s.fields)-1 {
				values, err := s.validateAll()
				if err != nil {
					s.errText = err.Error()
					break
				}
				if s.onSubmit != nil {
					return s.onSubmit(values), nil
				}
			} else {
				s.moveFocus(1)
			}
		}
	}

	var cmd tea.Cmd
	if s.focused >= 0 && s.focused < len(s.fields) {
		s.fields[s.focused].input, cmd = s.fields[s.focused].input.Update(msg)
	}
	return s, cmd
}

func (s *formScreen) View() string {
	header := renderHeader(s.title, s.desc, s.errText)
	body := strings.Builder{}
	for i, field := range s.fields {
		label := StyleLabel.Render(field.label)
		if i == s.focused {
			label = StyleHighlight.Render(field.label)
		}
		body.WriteString(label)
		if field.desc != "" {
			body.WriteString("\n")
			body.WriteString(StyleSubtle.Render(field.desc))
		}
		body.WriteString("\n")
		body.WriteString(field.input.View())
		body.WriteString("\n\n")
	}
	footer := renderFooter(s.footer, false)
	return header + body.String() + footer
}

func (s *formScreen) moveFocus(delta int) {
	if len(s.fields) == 0 {
		return
	}
	s.fields[s.focused].input.Blur()
	s.focused = (s.focused + delta + len(s.fields)) % len(s.fields)
	s.fields[s.focused].input.Focus()
}

func (s *formScreen) validateAll() (map[string]string, error) {
	values := make(map[string]string, len(s.fields))
	for _, field := range s.fields {
		value := strings.TrimSpace(field.input.Value())
		if field.required && value == "" {
			return nil, fmt.Errorf("%s is required", field.label)
		}
		if field.validate != nil {
			if err := field.validate(value); err != nil {
				return nil, err
			}
		}
		values[field.key] = value
	}
	return values, nil
}

type downloadProgressMsg struct {
	downloaded int64
	total      int64
}

type downloadDoneMsg struct {
	err error
}

type downloadScreen struct {
	state     *wizardState
	title     string
	desc      []string
	modelID   string
	progress  int
	total     int64
	footer    string
	errText   string
	onSuccess func() screen
	onCancel  func() screen
	updates   chan tea.Msg
	started   bool
}

func newDownloadScreen(state *wizardState, title string, desc []string, modelID string, onSuccess func() screen, onCancel func() screen) *downloadScreen {
	return &downloadScreen{
		state:     state,
		title:     title,
		desc:      desc,
		modelID:   modelID,
		onSuccess: onSuccess,
		onCancel:  onCancel,
		updates:   make(chan tea.Msg),
	}
}

func (s *downloadScreen) Init() tea.Cmd {
	if s.started {
		return listenForDownload(s.updates)
	}
	s.started = true
	return tea.Batch(s.startDownloadCmd(), listenForDownload(s.updates))
}

func (s *downloadScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadProgressMsg:
		s.total = msg.total
		if msg.total > 0 {
			s.progress = int(msg.downloaded * 100 / msg.total)
		}
		return s, listenForDownload(s.updates)
	case downloadDoneMsg:
		if msg.err != nil {
			s.errText = msg.err.Error()
			return s, nil
		}
		if s.onSuccess != nil {
			return s.onSuccess(), nil
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if s.onCancel != nil {
				return s.onCancel(), nil
			}
		}
	}

	return s, nil
}

func (s *downloadScreen) View() string {
	header := renderHeader(s.title, s.desc, s.errText)
	progressLine := "Downloading"
	if s.total > 0 {
		progressLine = fmt.Sprintf("Downloading... %d%%", s.progress)
	}
	body := StyleMuted.Render(progressLine) + "\n\n"
	footer := renderFooter(s.footer, false)
	return header + body + footer
}

func (s *downloadScreen) startDownloadCmd() tea.Cmd {
	modelID := s.modelID
	ch := s.updates
	return func() tea.Msg {
		err := downloadWhisperModel(modelID, func(downloaded, total int64) {
			ch <- downloadProgressMsg{downloaded: downloaded, total: total}
		})
		ch <- downloadDoneMsg{err: err}
		return nil
	}
}

func listenForDownload(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func itemsToList(items []optionItem) []list.Item {
	result := make([]list.Item, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

func toggleItemsToList(items []toggleItem) []list.Item {
	result := make([]list.Item, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

func listToToggleItems(items []list.Item) []toggleItem {
	result := make([]toggleItem, 0, len(items))
	for _, item := range items {
		if t, ok := item.(toggleItem); ok {
			result = append(result, t)
		}
	}
	return result
}

func renderHeader(title string, desc []string, errText string) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(StyleHeader.Render(title))
		b.WriteString("\n")
	}
	for _, line := range desc {
		if line == "" {
			continue
		}
		b.WriteString(StyleMuted.Render(line))
		b.WriteString("\n")
	}
	if errText != "" {
		b.WriteString("\n")
		b.WriteString(StyleError.Render(errText))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderFooter(extra string, filtering bool) string {
	if extra != "" {
		return StyleSubtle.Render(extra)
	}
	if filtering {
		return StyleSubtle.Render("enter apply • esc clear")
	}
	return StyleSubtle.Render("enter select • esc back")
}

func makeInputField(key, label, desc, value, placeholder string, validate func(string) error) formField {
	input := textinput.New()
	input.SetValue(value)
	input.Placeholder = placeholder
	input.Prompt = ""
	input.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return formField{key: key, label: label, desc: desc, input: input, validate: validate}
}

func parseDurationOrEmpty(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("duration is required")
	}
	return time.ParseDuration(value)
}
