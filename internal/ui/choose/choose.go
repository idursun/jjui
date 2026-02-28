package choose

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type SelectedMsg struct {
	Value string
}

type CancelledMsg struct{}

type itemClickMsg struct {
	Index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m itemScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	options             []string
	filteredOptions     []string
	selected            int
	title               string
	styles              styles
	listRenderer        *render.ListRenderer
	ensureCursorVisible bool
	filterable          bool
	filtering           bool
	ordered             bool
	input               textinput.Model
}

type styles struct {
	border   lipgloss.Style
	text     lipgloss.Style
	title    lipgloss.Style
	selected lipgloss.Style
	input    lipgloss.Style
}

const maxVisibleItems = 20

func New(options []string) *Model {
	return NewWithTitle(options, "", false)
}

func NewWithTitle(options []string, title string, filterable bool) *Model {
	return NewWithOptions(options, title, filterable, false)
}

func NewWithOptions(options []string, title string, filterable bool, ordered bool) *Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "filter..."
	ti.CharLimit = 100
	ti.SetWidth(20)

	m := &Model{
		options:         options,
		filteredOptions: options,
		title:           title,
		styles: styles{
			border:   common.DefaultPalette.GetBorder("choose border", lipgloss.RoundedBorder()),
			text:     common.DefaultPalette.Get("choose text"),
			title:    common.DefaultPalette.Get("choose title"),
			selected: common.DefaultPalette.Get("choose selected"),
			input:    common.DefaultPalette.Get("choose input"),
		},
		listRenderer: render.NewListRenderer(itemScrollMsg{}),
		filterable:   filterable,
		ordered:      ordered,
		input:        ti,
	}
	m.listRenderer.Z = render.ZMenuContent
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) StackedActionOwner() string {
	return actions.OwnerChoose
}

func (m *Model) IsEditing() bool {
	return m.filtering
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case intents.Intent:
		switch intent := msg.(type) {
		case intents.ChooseNavigate:
			m.move(intent.Delta)
		case intents.ChooseApply:
			return m.selectCurrent()
		case intents.ChooseCancel:
			if m.filtering {
				m.filtering = false
				m.input.Reset()
				m.filteredOptions = m.options
				m.selected = 0
				return nil
			}
			return newCmd(CancelledMsg{})
		}
	case tea.KeyMsg:
		if m.filtering {
			m.input, cmd = m.input.Update(msg)
			m.filterOptions()
			return cmd
		}
		if m.ordered {
			if r := msg.String(); len(r) == 1 && r[0] >= '1' && r[0] <= '9' {
				idx := int(r[0] - '1')
				if idx < len(m.filteredOptions) {
					m.selected = idx
					return m.selectCurrent()
				}
			}
		}
		if m.filterable && msg.String() == "/" {
			m.filtering = true
			m.input.Focus()
			return textinput.Blink
		}
		return nil
	case common.CloseViewMsg:
		return newCmd(CancelledMsg{})
	case itemScrollMsg:
		if msg.Horizontal {
			return nil
		}
		if m.listRenderer == nil {
			m.listRenderer = render.NewListRenderer(itemScrollMsg{})
		}
		m.listRenderer.StartLine += msg.Delta
		if m.listRenderer.StartLine < 0 {
			m.listRenderer.StartLine = 0
		}
	case itemClickMsg:
		if msg.Index < 0 || msg.Index >= len(m.filteredOptions) {
			return nil
		}
		m.selected = msg.Index
		return m.selectCurrent()
	}
	return nil
}

func (m *Model) filterOptions() {
	term := strings.ToLower(m.input.Value())
	if term == "" {
		m.filteredOptions = m.options
	} else {
		filtered := []string{}
		for _, opt := range m.options {
			if strings.Contains(strings.ToLower(opt), term) {
				filtered = append(filtered, opt)
			}
		}
		m.filteredOptions = filtered
	}
	if m.selected >= len(m.filteredOptions) {
		m.selected = 0
	}
}

func (m *Model) move(delta int) {
	if len(m.filteredOptions) == 0 {
		return
	}
	next := m.selected + delta
	n := len(m.filteredOptions)
	if next < 0 {
		next = 0
	}
	if next >= n {
		next = n - 1
	}
	if next == m.selected {
		return
	}
	m.selected = next
	m.ensureCursorVisible = true
}

func (m *Model) selectCurrent() tea.Cmd {
	if len(m.filteredOptions) == 0 {
		return newCmd(CancelledMsg{})
	}
	value := m.filteredOptions[m.selected]
	return newCmd(SelectedMsg{Value: value})
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if m.listRenderer == nil {
		m.listRenderer = render.NewListRenderer(itemScrollMsg{})
	}

	maxContentWidth := max(box.R.Dx()-2, 0)
	maxContentHeight := max(box.R.Dy()-2, 0)
	if maxContentWidth <= 0 || maxContentHeight <= 0 {
		return
	}

	titleHeight := 0
	if m.title != "" {
		titleHeight = 1
	}
	inputHeight := 0
	if m.filtering {
		inputHeight = 1
	}

	itemWidth := 0
	orderPrefix := 0
	if m.ordered {
		orderPrefix = 3 // "N. " prefix
	}
	for _, opt := range m.options {
		itemWidth = max(itemWidth, lipgloss.Width(opt)+2+orderPrefix)
	}
	if m.title != "" {
		itemWidth = max(itemWidth, lipgloss.Width(m.title))
	}
	if m.filtering {
		itemWidth = max(itemWidth, 25)
	}

	contentWidth := min(itemWidth, maxContentWidth)
	listHeightLimit := maxContentHeight - titleHeight - inputHeight
	if listHeightLimit < 0 {
		listHeightLimit = 0
	}
	// Use total options for height calculation to prevent resizing during filtering
	listHeight := min(min(len(m.options), listHeightLimit), maxVisibleItems)
	if listHeight == 0 && len(m.options) > 0 {
		listHeight = 1 // Ensure at least one line if there are options
	}
	if len(m.options) == 0 {
		listHeight = 0
	}

	contentHeight := titleHeight + inputHeight + listHeight
	if contentWidth <= 0 || contentHeight <= 0 {
		return
	}

	frame := box.Center(contentWidth+2, contentHeight+2)
	if frame.R.Dx() <= 0 || frame.R.Dy() <= 0 {
		return
	}

	dl.AddBackdrop(box.R, render.ZMenuBorder-1)
	contentBox := frame.Inset(1)
	if contentBox.R.Dx() <= 0 || contentBox.R.Dy() <= 0 {
		return
	}

	borderBase := lipgloss.NewStyle().Width(contentBox.R.Dx()).Height(contentBox.R.Dy()).Render("")
	dl.AddDraw(frame.R, m.styles.border.Render(borderBase), render.ZMenuBorder)

	listBox := contentBox
	if titleHeight > 0 {
		var titleBox layout.Box
		titleBox, listBox = contentBox.CutTop(1)
		dl.AddDraw(titleBox.R, m.styles.title.Render(m.title), render.ZMenuContent)
	}

	if inputHeight > 0 {
		var inputBox layout.Box
		inputBox, listBox = listBox.CutTop(1)
		dl.AddDraw(inputBox.R, m.styles.input.Render(m.input.View()), render.ZMenuContent)
	}

	if listBox.R.Dx() <= 0 || listBox.R.Dy() <= 0 {
		return
	}

	itemCount := len(m.filteredOptions)
	m.listRenderer.StartLine = render.ClampStartLine(m.listRenderer.StartLine, listBox.R.Dy(), itemCount)
	m.listRenderer.Render(
		dl,
		listBox,
		itemCount,
		m.selected,
		m.ensureCursorVisible,
		func(_ int) int { return 1 },
		func(dl *render.DisplayContext, index int, rect layout.Rectangle) {
			if index < 0 || index >= itemCount || rect.Dx() <= 0 || rect.Dy() <= 0 {
				return
			}
			style := m.styles.text
			if index == m.selected {
				style = m.styles.selected
			}
			label := m.filteredOptions[index]
			if m.ordered && index < 9 {
				label = fmt.Sprintf("%d. %s", index+1, label)
			}
			line := style.Padding(0, 1).Width(rect.Dx()).Render(label)
			dl.AddDraw(rect, line, render.ZMenuContent)
		},
		func(index int, _ tea.Mouse) tea.Msg { return itemClickMsg{Index: index} },
	)
	m.listRenderer.RegisterScroll(dl, listBox)
	m.ensureCursorVisible = false
}

func newCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func ShowWithTitle(options []string, title string, filter bool) tea.Cmd {
	return func() tea.Msg {
		return common.ShowChooseMsg{Options: options, Title: title, Filter: filter}
	}
}

func ShowOrdered(options []string, title string, filter bool, ordered bool) tea.Cmd {
	return func() tea.Msg {
		return common.ShowChooseMsg{Options: options, Title: title, Filter: filter, Ordered: ordered}
	}
}

func Show(options []string) tea.Cmd {
	return ShowWithTitle(options, "", false)
}
