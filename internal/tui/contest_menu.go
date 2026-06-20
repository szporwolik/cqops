package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

type contestMode int

const (
	contestList contestMode = iota
	contestEdit
	contestCreate
	contestConfirmDelete
)

// ContestChooser manages the contest list, create, edit, and delete flow.
type ContestChooser struct {
	app       *app.App
	mode      contestMode
	names     []string // display order: "None" + sorted contest IDs
	ids       []string // corresponding IDs (empty string for "None")
	cursor    int
	editID    string          // ID being edited (empty for create)
	nameInput textinput.Model // textinput for editing contest name
	toasts    *ToastQueue
	dialog    *DialogModel
	width     int
	height    int
	done      bool
}

func NewContestChooser(a *app.App, tq *ToastQueue) *ContestChooser {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Contest name"
	ti.SetWidth(30)
	ti.Focus()

	cc := &ContestChooser{
		app:       a,
		mode:      contestList,
		nameInput: ti,
		toasts:    tq,
	}
	cc.rebuildNames()
	return cc
}

// rebuildNames refreshes the names and ids slices from config.
func (c *ContestChooser) rebuildNames() {
	c.names = []string{"None"}
	c.ids = []string{""}
	sorted := config.SortedContestIDs(c.app.Config)
	for _, id := range sorted {
		contest := c.app.Config.Contests[id]
		c.names = append(c.names, config.ContestDisplayName(&contest))
		c.ids = append(c.ids, id)
	}
	// Keep cursor on active contest.
	c.cursor = 0
	if c.app.Config.State.ActiveContest != "" {
		for i, id := range c.ids {
			if id == c.app.Config.State.ActiveContest {
				c.cursor = i
				break
			}
		}
	}
}

func (c *ContestChooser) Init() tea.Cmd { return nil }

func (c *ContestChooser) formatDate(t string) string {
	if t == "" {
		return ""
	}
	// Parse ISO date and format as short date.
	parsed, err := time.Parse("2006-01-02", t)
	if err != nil {
		return t
	}
	return parsed.Format("2006-01-02")
}

func (c *ContestChooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case tea.KeyPressMsg:
		k := msg

		switch {
		case k.String() == "esc":
			if c.mode == contestList {
				c.done = true
				return c, nil
			}
			c.mode = contestList

		case c.mode == contestConfirmDelete:
			if c.dialog != nil {
				updated, _ := c.dialog.Update(msg)
				d := updated.(DialogModel)
				*c.dialog = d
				if d.Done() {
					if d.Result.Value == "delete" {
						return c, c.deleteContest()
					}
					c.dialog = nil
					c.mode = contestList
				}
				return c, nil
			}

		case c.mode == contestList && (k.String() == "enter" || k.String() == " " || msg.Code == ' '):
			if c.cursor == 0 {
				c.app.Config.State.ActiveContest = ""
				c.toasts.Success("Contest: None (no contest active)")
			} else if c.cursor > 0 && c.cursor < len(c.ids) {
				id := c.ids[c.cursor]
				c.app.Config.State.ActiveContest = id
				ct := c.app.Config.Contests[id]
				name := config.ContestDisplayName(&ct)
				c.toasts.Success(fmt.Sprintf("Contest activated: %s", name))
			}
			return c, nil

		case c.mode == contestList && k.String() == "insert":
			c.mode = contestCreate
			c.editID = ""
			c.nameInput.SetValue("")
			c.nameInput.Focus()

		case c.mode == contestList && (k.String() == "delete" || msg.Code == tea.KeyDelete):
			if c.cursor > 0 && c.cursor < len(c.ids) {
				id := c.ids[c.cursor]
				ct := c.app.Config.Contests[id]
				name := config.ContestDisplayName(&ct)
				c.mode = contestConfirmDelete
				d := NewDialog("Delete Contest", "Delete \""+name+"\"?",
					DangerOption("Delete", "delete"),
					Option{Label: "Cancel", Value: "cancel"},
				)
				c.dialog = &d
			}

		case c.mode == contestList && k.String() == "e" && c.cursor > 0:
			c.mode = contestEdit
			c.editID = c.ids[c.cursor]
			c.nameInput.SetValue(c.app.Config.Contests[c.editID].Name)
			c.nameInput.Focus()

		case c.mode == contestList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if c.cursor == 0 {
				c.cursor = len(c.names) - 1
			} else {
				c.cursor--
			}

		case c.mode == contestList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if c.cursor == len(c.names)-1 {
				c.cursor = 0
			} else {
				c.cursor++
			}

		case c.mode == contestEdit || c.mode == contestCreate:
			switch {
			case k.String() == "enter":
				return c, c.saveContest()
			case k.String() == "esc":
				c.mode = contestList
			default:
				var cmd tea.Cmd
				c.nameInput, cmd = c.nameInput.Update(msg)
				return c, cmd
			}
		}
	}
	return c, nil
}

func (c *ContestChooser) saveContest() tea.Cmd {
	name := strings.TrimSpace(c.nameInput.Value())
	if name == "" {
		c.toasts.Warn("Contest name cannot be empty")
		return nil
	}

	if c.mode == contestCreate {
		id := config.NewID(name)
		if c.app.Config.Contests == nil {
			c.app.Config.Contests = make(map[string]config.Contest)
		}
		c.app.Config.Contests[id] = config.Contest{
			ID:        id,
			Name:      name,
			CreatedAt: time.Now().Format("2006-01-02"),
		}
		c.app.Config.State.ActiveContest = id
		c.toasts.Success(fmt.Sprintf("Contest created: %s", name))
	} else {
		ct := c.app.Config.Contests[c.editID]
		ct.Name = name
		c.app.Config.Contests[c.editID] = ct
		c.toasts.Success(fmt.Sprintf("Contest saved: %s", name))
	}

	c.rebuildNames()
	c.mode = contestList
	return nil
}

func (c *ContestChooser) deleteContest() tea.Cmd {
	id := c.ids[c.cursor]
	ct := c.app.Config.Contests[id]
	name := config.ContestDisplayName(&ct)
	delete(c.app.Config.Contests, id)
	if c.app.Config.State.ActiveContest == id {
		c.app.Config.State.ActiveContest = ""
	}
	c.rebuildNames()
	c.toasts.Success(fmt.Sprintf("Contest deleted: %s", name))
	c.mode = contestList
	c.dialog = nil
	return nil
}

func (c *ContestChooser) View() tea.View {
	w := c.width
	if w < 40 {
		w = 80
	}

	switch c.mode {
	case contestEdit, contestCreate:
		return tea.NewView(c.viewEdit(w))
	case contestConfirmDelete:
		return tea.NewView(c.viewListWithDialog(w))
	default:
		return tea.NewView(c.viewList(w))
	}
}

func (c *ContestChooser) viewList(w int) string {
	var b strings.Builder

	if len(c.names) == 0 {
		b.WriteString("  No contests configured.\n")
	} else {
		for i, name := range c.names {
			prefix := "  "
			if i == c.cursor {
				prefix = S.FormPrefixOn.Render("> ")
			} else {
				prefix = "  "
			}

			// Build row: [Active] | date | name
			active := " "
			if c.ids[i] == c.app.Config.State.ActiveContest && c.ids[i] != "" {
				active = "*"
			} else if i == 0 && c.app.Config.State.ActiveContest == "" {
				active = "*"
			}

			dateStr := "          "
			if i > 0 {
				ct := c.app.Config.Contests[c.ids[i]]
				if ct.CreatedAt != "" {
					dateStr = c.formatDate(ct.CreatedAt)
				}
			}

			b.WriteString(fmt.Sprintf("%s[%s] %-10s %s\n", prefix, active, dateStr, name))
		}
	}
	return drawMenuWithHeader("Configuration \u2014 Contests", b.String(), w)
}

func (c *ContestChooser) viewEdit(w int) string {
	var b strings.Builder
	title := "Create Contest"
	if c.mode == contestEdit {
		title = "Edit Contest Name"
	}

	b.WriteString("  ")
	b.WriteString(S.FormLabel.Render("Name: "))
	b.WriteString(c.nameInput.View())
	return drawMenuWithHeader("Configuration \u2014 Contests \u2014 "+title, b.String(), w)
}

func (c *ContestChooser) viewListWithDialog(w int) string {
	body := c.viewList(w)
	if c.dialog != nil {
		return RenderDialogOverlay(body, *c.dialog, w, 15)
	}
	return body
}
