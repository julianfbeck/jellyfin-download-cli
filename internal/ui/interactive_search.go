package ui

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type SearchResult struct {
	ID    string
	Name  string
	Type  string
	Extra string
}

var ErrCanceled = errors.New("selection canceled")

func InteractiveSearch(ctx context.Context, title string, initialQuery string, searchFn func(context.Context, string) ([]SearchResult, error)) (*SearchResult, error) {
	model := newSearchModel(ctx, title, initialQuery, searchFn)
	program := tea.NewProgram(model)
	result, err := program.Run()
	if err != nil {
		return nil, err
	}
	m, ok := result.(searchModel)
	if !ok {
		return nil, ErrCanceled
	}
	if m.selected == nil {
		return nil, ErrCanceled
	}
	return m.selected, nil
}

type searchModel struct {
	ctx       context.Context
	searchFn  func(context.Context, string) ([]SearchResult, error)
	input     textinput.Model
	list      list.Model
	focused   bool
	query     string
	status    string
	selected  *SearchResult
	title     string
}

type searchMsg struct {
	query string
	items []SearchResult
	err   error
}

func newSearchModel(ctx context.Context, title string, initialQuery string, searchFn func(context.Context, string) ([]SearchResult, error)) searchModel {
	input := textinput.New()
	input.Placeholder = "Search..."
	input.SetValue(initialQuery)
	input.Focus()
	input.Prompt = "> "

	lst := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	lst.Title = "Results"
	lst.SetShowHelp(false)

	m := searchModel{
		ctx:      ctx,
		searchFn: searchFn,
		input:    input,
		list:     lst,
		focused:  true,
		query:    initialQuery,
		status:   "Type and press Enter to search",
		title:    title,
	}

	return m
}

func (m searchModel) Init() tea.Cmd {
	if m.query != "" {
		return m.search(m.query)
	}
	return nil
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focused = !m.focused
			if m.focused {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil
		case "enter":
			if m.focused {
				query := m.input.Value()
				if query == "" {
					m.status = "Enter a query to search"
					return m, nil
				}
				m.query = query
				m.status = "Searching..."
				return m, m.search(query)
			}
			if item, ok := m.list.SelectedItem().(resultItem); ok {
				selected := item.result
				m.selected = &selected
				return m, tea.Quit
			}
			return m, nil
		}
	case searchMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}
		items := make([]list.Item, 0, len(msg.items))
		for _, item := range msg.items {
			items = append(items, resultItem{result: item})
		}
		m.list.SetItems(items)
		if len(items) == 0 {
			m.status = "No results"
		} else {
			m.status = fmt.Sprintf("%d results", len(items))
		}
		return m, nil
	case tea.WindowSizeMsg:
		height := msg.Height - 8
		if height < 4 {
			height = 4
		}
		width := msg.Width - 2
		if width < 20 {
			width = 20
		}
		m.list.SetSize(width, height)
		return m, nil
	}

	var cmd tea.Cmd
	if m.focused {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m searchModel) View() string {
	help := "Enter=search, Tab=toggle, up/down navigate, Enter=select, q=quit"
	return fmt.Sprintf("%s\n\n%s\n\n%s\n%s\n", m.title, m.input.View(), m.list.View(), m.status+"\n"+help)
}

func (m searchModel) search(query string) tea.Cmd {
	return func() tea.Msg {
		items, err := m.searchFn(m.ctx, query)
		return searchMsg{query: query, items: items, err: err}
	}
}

type resultItem struct {
	result SearchResult
}

func (r resultItem) Title() string       { return r.result.Name }
func (r resultItem) Description() string { return r.result.Extra }
func (r resultItem) FilterValue() string { return r.result.Name }
