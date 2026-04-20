package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"bookclubvote/internal/config"
	"bookclubvote/internal/results"
)

type state int

const (
	stateNoPolls state = iota
	stateChoosePoll
	stateEnterName
	stateRank
	stateConfirm
	stateSuccess
	stateError
)

type Model struct {
	activePolls []config.Poll
	selected    config.Poll
	state       state
	width       int
	accessible  bool

	respondentName string
	ranking        []string
	pollCursor     int
	currentRank    int
	rankCursor     int
	confirmCursor  int
	confirmNotice  string
	notice         string
	err            error
	message        string
}

func NewModel(cfg config.Config, now time.Time) Model {
	active := cfg.ActivePolls(now)
	m := Model{
		activePolls: active,
		width:       80,
		accessible:  cfg.Server.Accessible,
	}

	switch len(active) {
	case 0:
		m.state = stateNoPolls
	case 1:
		m.selected = active[0]
		m.enterSelectedPoll()
	default:
		m.state = stateChoosePoll
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "enter":
			if m.state == stateSuccess || m.state == stateError || m.state == stateNoPolls {
				return m, tea.Quit
			}
		case "q", "esc":
			if m.state == stateNoPolls || m.state == stateSuccess || m.state == stateError {
				return m, tea.Quit
			}
		}

		switch m.state {
		case stateChoosePoll:
			return m.updatePollSelection(msg)
		case stateEnterName:
			return m.updateName(msg)
		case stateRank:
			return m.updateRank(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		default:
			return m, nil
		}
	default:
		return m, nil
	}
}

func (m Model) View() tea.View {
	base := lipgloss.NewStyle().Padding(1, 2)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	title := lipgloss.NewStyle().Bold(true)

	switch m.state {
	case stateNoPolls:
		return tea.NewView(base.Render(title.Render("Book Club Vote") + "\n\nNo polls are currently open.\n\n" + muted.Render("Press q to quit.")))
	case stateSuccess:
		return tea.NewView(base.Render(title.Render("Vote recorded") + "\n\n" + m.message + "\n\n" + muted.Render("Press Enter, q, or esc to quit.")))
	case stateError:
		return tea.NewView(base.Render(title.Render("Submission failed") + "\n\n" + m.err.Error() + "\n\n" + muted.Render("Press Enter, q, or esc to quit.")))
	default:
		var header strings.Builder
		header.WriteString(title.Render("Book Club Vote"))
		if m.selected.ID != "" {
			header.WriteString("\n\n")
			header.WriteString(m.selected.Name)
			header.WriteString("\n")
			header.WriteString(m.selected.Description)
		}

		var body string
		switch m.state {
		case stateChoosePoll:
			body = m.pollSelectionView()
		case stateEnterName:
			body = m.nameView()
		case stateRank:
			body = m.rankView()
		case stateConfirm:
			body = m.confirmView()
		}

		footer := muted.Render("Press ctrl+c to disconnect.")
		if body != "" {
			return tea.NewView(base.Render(header.String() + "\n\n" + body + "\n\n" + footer))
		}
		return tea.NewView(base.Render(header.String() + "\n\n" + footer))
	}
}

func (m *Model) enterSelectedPoll() {
	m.ranking = make([]string, len(m.selected.Books))
	m.respondentName = ""
	m.confirmNotice = ""
	m.notice = ""
	m.currentRank = 0
	m.rankCursor = 0
	m.confirmCursor = 0
	if m.selected.RecordRespondentName {
		m.state = stateEnterName
		return
	}
	m.state = stateRank
}

func (m Model) updatePollSelection(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if len(m.activePolls) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		m.pollCursor--
		if m.pollCursor < 0 {
			m.pollCursor = len(m.activePolls) - 1
		}
	case "down", "j":
		m.pollCursor++
		if m.pollCursor >= len(m.activePolls) {
			m.pollCursor = 0
		}
	case "enter":
		m.selected = m.activePolls[m.pollCursor]
		m.enterSelectedPoll()
	}

	return m, nil
}

func (m Model) updateName(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.respondentName = strings.TrimSpace(m.respondentName)
		m.state = stateRank
		m.currentRank = 0
		m.rankCursor = 0
		return m, nil
	case "backspace":
		m.respondentName = trimLastRune(m.respondentName)
	case "esc", "left", "h":
		if len(m.activePolls) > 1 {
			m.state = stateChoosePoll
			m.selected = config.Poll{}
		}
	default:
		if text := msg.Text; text != "" && isPrintableText(text) {
			m.respondentName += text
		}
	}

	return m, nil
}

func (m Model) updateRank(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	optionCount := len(m.rankOptions())
	if m.currentRank > 0 {
		optionCount++
	}
	if optionCount == 0 {
		m.state = stateError
		m.err = fmt.Errorf("no books remain to rank")
		return m, nil
	}

	switch msg.String() {
	case "ctrl+g":
		if book, ok := m.currentHighlightedBook(); ok {
			m.notice = "Copied Goodreads URL to clipboard."
			return m, tea.SetClipboard(book.GoodreadsURL)
		}
	case "ctrl+y":
		if book, ok := m.currentHighlightedBook(); ok {
			m.notice = "Copied Moly URL to clipboard."
			return m, tea.SetClipboard(book.MolyURL)
		}
	case "up", "k":
		m.rankCursor--
		if m.rankCursor < 0 {
			m.rankCursor = optionCount - 1
		}
	case "down", "j":
		m.rankCursor++
		if m.rankCursor >= optionCount {
			m.rankCursor = 0
		}
	case "left", "h":
		if m.currentRank > 0 {
			m.currentRank--
			m.ranking[m.currentRank] = ""
			m.rankCursor = min(m.rankCursor, len(m.rankOptions()))
		}
	case "enter":
		options := m.rankOptions()
		if m.currentRank > 0 && m.rankCursor == len(options) {
			m.currentRank--
			m.ranking[m.currentRank] = ""
			m.rankCursor = min(m.rankCursor, len(m.rankOptions()))
			return m, nil
		}
		if m.rankCursor < 0 || m.rankCursor >= len(options) {
			return m, nil
		}
		m.ranking[m.currentRank] = options[m.rankCursor].ID
		m.notice = ""
		m.currentRank++
		m.rankCursor = 0
		if m.currentRank >= len(m.selected.Books) {
			if err := validateRanking(m.selected, m.ranking); err != nil {
				m.state = stateError
				m.err = err
				return m, nil
			}
			m.state = stateConfirm
			m.confirmCursor = 0
			return m, nil
		}
	}

	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.confirmCursor--
		if m.confirmCursor < 0 {
			m.confirmCursor = len(m.confirmOptions()) - 1
		}
	case "down", "j":
		m.confirmCursor++
		if m.confirmCursor >= len(m.confirmOptions()) {
			m.confirmCursor = 0
		}
	case "esc", "left", "h":
		m.state = stateRank
		m.currentRank = len(m.selected.Books) - 1
		m.rankCursor = 0
		m.confirmNotice = ""
	case "enter":
		if m.confirmCursor == 1 {
			m.confirmNotice = "Ranking reset. Choose your preferences again."
			m.state = stateRank
			m.currentRank = 0
			m.rankCursor = 0
			m.ranking = make([]string, len(m.selected.Books))
			return m, nil
		}
		ballot := results.Ballot{
			SubmittedAt:    time.Now().UTC(),
			RespondentName: strings.TrimSpace(m.respondentName),
			Ranking:        append([]string(nil), m.ranking...),
		}
		if err := results.AppendBallot(m.selected.ResultsPath, ballot, m.selected.ID); err != nil {
			m.state = stateError
			m.err = err
			return m, nil
		}
		m.state = stateSuccess
		m.message = m.successMessage()
	}
	return m, nil
}

func remainingBooks(poll config.Poll, ranking []string) []config.Book {
	selected := make(map[string]struct{}, len(ranking))
	for _, bookID := range ranking {
		if strings.TrimSpace(bookID) == "" {
			continue
		}
		selected[bookID] = struct{}{}
	}

	remaining := make([]config.Book, 0, len(poll.Books)-len(selected))
	for _, book := range poll.Books {
		if _, ok := selected[book.ID]; ok {
			continue
		}
		remaining = append(remaining, book)
	}
	return remaining
}

func validateRanking(poll config.Poll, ranking []string) error {
	if len(ranking) != len(poll.Books) {
		return fmt.Errorf("ranking must contain exactly %d books", len(poll.Books))
	}
	seen := make(map[string]struct{}, len(ranking))
	allowed := make(map[string]struct{}, len(poll.Books))
	for _, book := range poll.Books {
		allowed[book.ID] = struct{}{}
	}
	for i, bookID := range ranking {
		if strings.TrimSpace(bookID) == "" {
			return fmt.Errorf("rank #%d is empty", i+1)
		}
		if _, ok := allowed[bookID]; !ok {
			return fmt.Errorf("rank #%d references unknown book %q", i+1, bookID)
		}
		if _, ok := seen[bookID]; ok {
			return fmt.Errorf("book %q was selected more than once", bookID)
		}
		seen[bookID] = struct{}{}
	}
	return nil
}

func (m Model) ballotSummary() string {
	lines := make([]string, 0, len(m.ranking)+1)
	if strings.TrimSpace(m.respondentName) != "" {
		lines = append(lines, "Name: "+strings.TrimSpace(m.respondentName))
	}
	for i, bookID := range m.ranking {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, m.selected.BookLabel(bookID)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) partialRankingSummary() string {
	lines := make([]string, 0, m.currentRank)
	for i := 0; i < m.currentRank; i++ {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, m.selected.BookLabel(m.ranking[i])))
	}
	return strings.Join(lines, "\n")
}

func (m Model) pollSelectionView() string {
	var b strings.Builder
	b.WriteString(renderFieldHeader("Choose a poll", "Only currently open polls are shown."))
	for i, poll := range m.activePolls {
		label := poll.Name + " (" + poll.ID + ")"
		b.WriteString(renderFieldOption(i == m.pollCursor, label))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderFieldHelp("Use up/down, j/k, Enter to choose."))
	return renderFocusedField(strings.TrimSpace(b.String()))
}

func (m Model) nameView() string {
	desc := "Enter your name for this ballot. Leave it blank if you prefer."
	if len(m.activePolls) > 1 {
		desc += " Left/h returns to poll selection."
	}
	value := m.respondentName
	if value == "" {
		value = renderFieldPlaceholder("Optional name")
	}

	var b strings.Builder
	b.WriteString(renderFieldHeader("Your name", desc))
	b.WriteString(renderTextInputLine(value))
	b.WriteString("\n\n")
	b.WriteString(renderFieldHelp("Type your name and press Enter to continue."))
	return renderFocusedField(strings.TrimSpace(b.String()))
}

func (m Model) rankOptions() []config.Book {
	return remainingBooks(m.selected, m.ranking[:m.currentRank])
}

func (m Model) rankView() string {
	var b strings.Builder
	b.WriteString(renderFieldHeader(
		fmt.Sprintf("Rank #%d of %d", m.currentRank+1, len(m.selected.Books)),
		"Choose your next preference.",
	))
	if strings.TrimSpace(m.confirmNotice) != "" {
		b.WriteString(renderFieldNotice(m.confirmNotice))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(m.notice) != "" {
		b.WriteString(renderFieldNotice(m.notice))
		b.WriteString("\n\n")
	}
	if m.currentRank > 0 {
		b.WriteString(renderFieldDescription("Current ranking:\n" + m.partialRankingSummary()))
		b.WriteString("\n\n")
	}
	for i, book := range m.rankOptions() {
		b.WriteString(renderFieldOption(i == m.rankCursor, renderBookOptionLabel(book)))
		b.WriteString("\n")
	}
	if m.currentRank > 0 {
		b.WriteString(renderFieldOption(m.rankCursor == len(m.rankOptions()), "Back to previous rank"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderFieldHelp("Use up/down, j/k, Enter to select. Left/h goes back one rank. Ctrl+g copies Goodreads, Ctrl+y copies Moly."))
	return renderFocusedField(strings.TrimSpace(b.String()))
}

func (m Model) confirmOptions() []string {
	return []string{"Submit ballot", "Edit ranking from start"}
}

func (m Model) confirmView() string {
	var b strings.Builder
	b.WriteString(renderFieldHeader("Review your ballot", "Choose what to do with this ranking."))
	if strings.TrimSpace(m.confirmNotice) != "" {
		b.WriteString(renderFieldNotice(m.confirmNotice))
		b.WriteString("\n\n")
	}
	b.WriteString(renderFieldDescription(m.ballotSummary()))
	b.WriteString("\n\n")
	for i, option := range m.confirmOptions() {
		b.WriteString(renderFieldOption(i == m.confirmCursor, option))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderFieldHelp("Use up/down, j/k, Enter to choose. Esc returns to ranking."))
	return renderFocusedField(strings.TrimSpace(b.String()))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func renderFocusedField(body string) string {
	return lipgloss.NewStyle().
		PaddingLeft(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("238")).
		Render(body)
}

func renderFieldHeader(title string, description string) string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9")).Bold(true)
	descriptionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	return titleStyle.Render(title) + "\n" + descriptionStyle.Render(description) + "\n"
}

func renderFieldDescription(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(text)
}

func renderFieldNotice(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Render(text)
}

func renderFieldOption(selected bool, text string) string {
	selector := lipgloss.NewStyle().SetString("  ")
	optionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	if selected {
		selector = lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2")).SetString("> ")
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, selector.String(), optionStyle.Render(text))
}

func renderFieldHelp(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(text)
}

func (m Model) currentHighlightedBook() (config.Book, bool) {
	options := m.rankOptions()
	if m.rankCursor < 0 || m.rankCursor >= len(options) {
		return config.Book{}, false
	}
	return options[m.rankCursor], true
}

func renderBookOptionLabel(book config.Book) string {
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	linkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9")).Underline(true)

	goodreads := linkStyle.Hyperlink(book.GoodreadsURL).Render("Goodreads")
	moly := linkStyle.Hyperlink(book.MolyURL).Render("Moly")

	return metaStyle.Render(fmt.Sprintf("%s by %s", book.Title, book.Author)) +
		separatorStyle.Render("  [") +
		goodreads +
		separatorStyle.Render(" | ") +
		moly +
		separatorStyle.Render("]")
}

func renderFieldPlaceholder(text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(text)
}

func renderTextInputLine(value string) string {
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#F780E2")).Render("> ")
	text := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(value)
	return lipgloss.JoinHorizontal(lipgloss.Left, prompt, text)
}

func trimLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}

func isPrintableText(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func (m Model) successMessage() string {
	var b strings.Builder
	b.WriteString("Your ballot was saved!\n\n")
	for i, bookID := range m.ranking {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.selected.BookLabel(bookID)))
	}
	return strings.TrimSpace(b.String())
}
