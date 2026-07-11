package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	app                 *app.App
	mode                contestMode
	names               []string
	ids                 []string
	cursor              int
	editID              string
	focus               int // 0=name,1=date,2=inUse,3=nextQSO,4=contestID,5=prefillSent,6=exchSent,7=prefillRcvd,8=exchRcvd
	nameInput           textinput.Model
	dateInput           textinput.Model
	nextInput           textinput.Model
	contInput           textinput.Model
	exchSentInput       textinput.Model
	exchRcvdInput       textinput.Model
	prefillExchange     bool
	prefillExchangeRcvd bool
	inUse               bool
	adifIdx             int // index into adifContestIDs for space cycling
	needsSave           bool
	toasts              *ToastQueue
	dialog              *DialogModel
	width               int
	height              int
	done                bool

	// Render cache — avoids rebuilding views on every frame.
	cachedList string
	cachedForm string
	listSig    string
	formSig    string

	// Viewport for scrolling form content on small terminals.
	vp              viewport.Model
	lastFormContent string
	lastListContent string
}

func NewContestChooser(a *app.App, tq *ToastQueue) *ContestChooser {
	ni := newTextinput()
	ni.Placeholder = "Contest name"
	ni.SetWidth(30)

	di := newTextinput()
	di.Placeholder = "YYYY-MM-DD"
	di.SetWidth(12)
	di.CharLimit = 10

	xi := newTextinput()
	xi.Placeholder = "1"
	xi.SetWidth(6)
	xi.CharLimit = 6

	ci := newTextinput()
	ci.Placeholder = "ADIF Contest-ID"
	ci.SetWidth(20)
	ci.CharLimit = 30

	ei := newTextinput()
	ei.SetWidth(40)
	ei.CharLimit = 60

	ri := newTextinput()
	ri.SetWidth(40)
	ri.CharLimit = 60

	cc := &ContestChooser{
		app:           a,
		mode:          contestList,
		nameInput:     ni,
		dateInput:     di,
		nextInput:     xi,
		contInput:     ci,
		exchSentInput: ei,
		exchRcvdInput: ri,
		toasts:        tq,
	}
	cc.rebuildNames()
	return cc
}

// rebuildNames refreshes the names and ids slices from config.
func (c *ContestChooser) rebuildNames() {
	c.names = []string{"None"}
	c.ids = []string{""}
	sorted := config.SortedContestIDs(c.app.Config, c.app.LogbookName)
	for _, id := range sorted {
		contest := c.app.Config.Contests[id]
		c.names = append(c.names, config.ContestDisplayName(&contest))
		c.ids = append(c.ids, id)
	}
	// Keep cursor on active contest.
	c.cursor = 0
	if c.app.Logbook.ActiveContest != "" {
		for i, id := range c.ids {
			if id == c.app.Logbook.ActiveContest {
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
				d, ok := updated.(DialogModel)
				if !ok {
					return c, nil
				}
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

		case c.mode == contestList && k.String() == "enter":
			if c.cursor == 0 {
				return c, c.handleActivate()
			}
			if c.cursor < len(c.ids) {
				c.startEdit(c.ids[c.cursor])
				return c, c.nameInput.Focus()
			}

		case c.mode == contestList && (k.String() == " " || msg.Code == ' '):
			return c, c.handleActivate()

		case c.mode == contestList && k.String() == "insert":
			c.startCreate()
			return c, c.nameInput.Focus()

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

		case c.mode == contestList && (k.String() == "pgup" || k.String() == "pgdown" || k.String() == "home" || k.String() == "end"):
			c.vp, _ = c.vp.Update(msg)
			return c, nil

		case c.mode == contestList && (msg.Code == tea.KeyUp || k.String() == "up" || k.String() == "k"):
			if c.cursor == 0 {
				c.cursor = len(c.names) - 1
			} else {
				c.cursor--
			}
			scrollVpToLine(&c.vp, c.cursor)

		case c.mode == contestList && (msg.Code == tea.KeyDown || k.String() == "down" || k.String() == "j"):
			if c.cursor == len(c.names)-1 {
				c.cursor = 0
			} else {
				c.cursor++
			}
			scrollVpToLine(&c.vp, c.cursor)

		case c.mode == contestEdit || c.mode == contestCreate:
			switch {
			case k.String() == "ctrl+s":
				return c, c.saveContest()
			case k.String() == "esc":
				c.mode = contestList
				return c, nil
			case c.focus == 2 && (k.String() == " " || msg.Code == ' ' || k.String() == "enter"):
				// Toggle In Use checkbox.
				c.inUse = !c.inUse
				return c, nil
			case c.focus == 5 && (k.String() == " " || msg.Code == ' ' || k.String() == "enter"):
				// Toggle prefill exchange sent checkbox.
				c.prefillExchange = !c.prefillExchange
				if !c.prefillExchange {
					c.exchSentInput.Blur()
					c.exchSentInput.SetValue("")
				}
				return c, nil
			case c.focus == 7 && (k.String() == " " || msg.Code == ' ' || k.String() == "enter"):
				// Toggle prefill exchange rcvd checkbox.
				c.prefillExchangeRcvd = !c.prefillExchangeRcvd
				if !c.prefillExchangeRcvd {
					c.exchRcvdInput.Blur()
					c.exchRcvdInput.SetValue("")
				}
				return c, nil
			case c.focus == 4 && (k.String() == " " || msg.Code == ' '):
				// Space cycles forward through ADIF Contest IDs.
				cur := strings.TrimSpace(c.contInput.Value())
				nxt := nextContestID(cur)
				c.contInput.SetValue(nxt)
				c.adifIdx = 0
				return c, nil
			case c.focus == 4 && (k.String() == "pgdown" || k.String() == "pgup"):
				// PgDn/PgUp cycle forward/backward through ADIF Contest IDs.
				cur := strings.TrimSpace(c.contInput.Value())
				var nxt string
				if k.String() == "pgup" {
					nxt = prevContestID(cur)
				} else {
					nxt = nextContestID(cur)
				}
				c.contInput.SetValue(nxt)
				c.adifIdx = 0
				return c, nil
			case k.String() == "tab" || k.String() == "down":
				if c.focus == 4 {
					c.validateContestID()
				}
				c.blurAll()
				c.focus = (c.focus + 1) % c.visibleItems()
				scrollVpToLine(&c.vp, c.focus)
				return c, c.focusField()
			case k.String() == "shift+tab" || k.String() == "up":
				if c.focus == 4 {
					c.validateContestID()
				}
				c.blurAll()
				c.focus = (c.focus - 1 + c.visibleItems()) % c.visibleItems()
				scrollVpToLine(&c.vp, c.focus)
				return c, c.focusField()
			case c.focus == 2 || c.focus == 5 || c.focus == 7:
				// Checkboxes handle Space/Enter; ignore other keys.
				return c, nil
			case k.String() == "pgup", k.String() == "pgdown", k.String() == "home", k.String() == "end":
				c.vp, _ = c.vp.Update(msg)
				return c, nil
			default:
				var cmd tea.Cmd
				ti := c.focusedInput()
				*ti, cmd = ti.Update(msg)
				return c, cmd
			}
		}
	}
	return c, nil
}

func (c *ContestChooser) handleActivate() tea.Cmd {
	if c.cursor == 0 {
		c.app.SetActiveContest("")
		c.toasts.Success("Contest: None (no contest active)")
		return nil
	}
	id := c.ids[c.cursor]
	ct := c.app.Config.Contests[id]
	c.app.SetActiveContest(id)
	c.toasts.Success(fmt.Sprintf("Contest activated: %s", config.ContestDisplayName(&ct)))
	return nil
}

func (c *ContestChooser) focusedInput() *textinput.Model {
	switch c.focus {
	case 1:
		return &c.dateInput
	case 3:
		return &c.nextInput
	case 4:
		return &c.contInput
	case 6:
		return &c.exchSentInput
	case 8:
		return &c.exchRcvdInput
	default:
		return &c.nameInput
	}
}

// visibleItems returns the number of focusable items (used for tab wrapping).
func (c *ContestChooser) visibleItems() int {
	n := 5 // name, date, inUse, nextQSO, contestID
	n += 2 // two checkboxes always visible (prefill sent, prefill rcvd)
	if c.prefillExchange {
		n++ // exchange sent field
	}
	if c.prefillExchangeRcvd {
		n++ // exchange rcvd field
	}
	return n
}

func (c *ContestChooser) blurAll() {
	c.nameInput.Blur()
	c.dateInput.Blur()
	c.nextInput.Blur()
	c.contInput.Blur()
	c.exchSentInput.Blur()
	c.exchRcvdInput.Blur()
}

func (c *ContestChooser) focusField() tea.Cmd {
	if c.focus == 2 || c.focus == 5 || c.focus == 7 {
		// Checkboxes — no textinput to focus.
		return nil
	}
	return c.focusedInput().Focus()
}

func (c *ContestChooser) validateContestID() {
	cid := strings.TrimSpace(c.contInput.Value())
	if cid != "" && !isValidContestID(cid) {
		c.toasts.Warn("Contest ID not found in the ADIF spec")
	}
}

func (c *ContestChooser) startEdit(id string) {
	c.editID = id
	c.focus = 0
	ct := c.app.Config.Contests[id]
	c.nameInput.SetValue(ct.Name)
	c.dateInput.SetValue(ct.Date)
	c.inUse = ct.InUse == nil || *ct.InUse
	c.nextInput.SetValue(fmt.Sprintf("%d", ct.NextQSO))
	c.contInput.SetValue(ct.ContestID)
	c.prefillExchange = ct.PrefillExchange
	c.exchSentInput.SetValue(ct.ExchangeSent)
	c.prefillExchangeRcvd = ct.PrefillExchangeRcvd
	c.exchRcvdInput.SetValue(ct.ExchangeRcvd)
	c.blurAll()
	c.mode = contestEdit
	c.nameInput.Focus()
}

func (c *ContestChooser) startCreate() {
	c.editID = ""
	c.focus = 0
	c.nameInput.SetValue("")
	c.dateInput.SetValue(time.Now().Format("2006-01-02"))
	c.inUse = true
	c.nextInput.SetValue("1")
	c.contInput.SetValue("")
	c.prefillExchange = false
	c.exchSentInput.SetValue("@rst @serial")
	c.prefillExchangeRcvd = false
	c.exchRcvdInput.SetValue("@rst")
	c.blurAll()
	c.mode = contestCreate
	c.nameInput.Focus()
}

func (c *ContestChooser) saveContest() tea.Cmd {
	name := strings.TrimSpace(c.nameInput.Value())
	if name == "" {
		c.toasts.Warn("Contest name cannot be empty")
		return nil
	}
	dateStr := strings.TrimSpace(c.dateInput.Value())
	nextStr := strings.TrimSpace(c.nextInput.Value())
	if nextStr == "" {
		c.toasts.Warn("Next QSO / Rcvd serial is required")
		return nil
	}
	nextQSO, err := strconv.Atoi(nextStr)
	if err != nil || nextQSO < 1 {
		c.toasts.Warn("Next QSO / Rcvd serial must be a positive integer")
		return nil
	}
	contestID := strings.TrimSpace(c.contInput.Value())
	if contestID == "" {
		c.toasts.Warn("Contest ADIF ID is required — press Space to cycle through known IDs")
		return nil
	}
	exchangeSent := strings.TrimSpace(c.exchSentInput.Value())
	exchangeRcvd := strings.TrimSpace(c.exchRcvdInput.Value())

	if c.mode == contestCreate {
		id := config.NewID(name)
		if c.app.Config.Contests == nil {
			c.app.Config.Contests = make(map[string]config.Contest)
		}
		c.app.Config.Contests[id] = config.Contest{
			ID:                  id,
			LogbookID:           c.app.LogbookName,
			Name:                name,
			Date:                dateStr,
			NextQSO:             nextQSO,
			ContestID:           contestID,
			ContestIDName:       contestIDDesc(contestID),
			PrefillExchange:     c.prefillExchange,
			ExchangeSent:        exchangeSent,
			PrefillExchangeRcvd: c.prefillExchangeRcvd,
			ExchangeRcvd:        exchangeRcvd,
			InUse:               &c.inUse,
		}
		c.app.SetActiveContest(id)
		c.toasts.Success(fmt.Sprintf("Contest created: %s", name))
	} else {
		ct := c.app.Config.Contests[c.editID]
		ct.Name = name
		ct.Date = dateStr
		ct.NextQSO = nextQSO
		ct.ContestID = contestID
		ct.ContestIDName = contestIDDesc(contestID)
		ct.PrefillExchange = c.prefillExchange
		ct.ExchangeSent = exchangeSent
		ct.PrefillExchangeRcvd = c.prefillExchangeRcvd
		ct.ExchangeRcvd = exchangeRcvd
		ct.InUse = &c.inUse
		c.app.Config.Contests[c.editID] = ct
		// If the contest was just marked "not in use" and it is the
		// active contest, clear the active contest so the user isn't
		// stuck inside a deactivated contest.
		if !c.inUse && c.app.Logbook.ActiveContest == c.editID {
			c.app.SetActiveContest("")
			c.toasts.Info(fmt.Sprintf("Contest %q deactivated — active contest cleared", name))
		}
		c.toasts.Success(fmt.Sprintf("Contest saved: %s", name))
	}

	c.rebuildNames()
	c.mode = contestList
	c.needsSave = true
	return nil
}

func (c *ContestChooser) deleteContest() tea.Cmd {
	id := c.ids[c.cursor]
	ct := c.app.Config.Contests[id]
	name := config.ContestDisplayName(&ct)
	delete(c.app.Config.Contests, id)
	if c.app.Logbook.ActiveContest == id {
		c.app.SetActiveContest("")
	}
	c.rebuildNames()
	c.toasts.Success(fmt.Sprintf("Contest deleted: %s", name))
	c.mode = contestList
	c.dialog = nil
	return nil
}

func (c *ContestChooser) View() tea.View {
	if c.done {
		return tea.NewView("")
	}

	switch c.mode {
	case contestList:
		return tea.NewView(c.viewList())
	case contestEdit, contestCreate:
		return tea.NewView(c.viewForm())
	case contestConfirmDelete:
		body := c.viewList()
		if c.dialog != nil {
			body = RenderDialogOverlay(body, *c.dialog, c.width, c.height)
		}
		return tea.NewView(body)
	}
	return tea.NewView("")
}

func (c *ContestChooser) viewList() string {
	w := c.width
	if w < 40 {
		w = 80
	}
	h := c.height
	if h < 10 {
		h = 24
	}

	// Render cache — skip expensive work on every frame.
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(w))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(h))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(c.cursor))
	sb.WriteByte('|')
	sb.WriteString(c.app.Logbook.ActiveContest)
	for _, n := range c.names {
		sb.WriteByte(',')
		sb.WriteString(n)
	}
	sig := sb.String()

	// Cache the expensive content building, but always re-render
	// the viewport layout (scroll position may have changed).
	listContent := c.cachedList
	if c.listSig != sig || listContent == "" {
		var b strings.Builder
		contentH := contentHeight(h)
		if contentH < 3 {
			contentH = 3
		}

		if len(c.names) == 0 {
			b.WriteString("No contests configured.\n")
		} else {
			contentW := w - 8
			if contentW > partnerMapMaxW-8 {
				contentW = partnerMapMaxW - 8
			}
			if contentW < 20 {
				contentW = 20
			}

			for i, name := range c.names {
				prefix := "  "
				if i == c.cursor {
					prefix = S.FormPrefixOn.Render("> ")
				}

				activeBadge := padOrTrunc("[      ]", 10)
				if c.ids[i] == c.app.Logbook.ActiveContest && c.ids[i] != "" {
					activeBadge = S.ToastSuccess.Render(padOrTrunc("[Active]", 10))
				} else if i == 0 && c.app.Logbook.ActiveContest == "" {
					activeBadge = S.ToastSuccess.Render(padOrTrunc("[Active]", 10))
				}

				dateStr := ""
				if i > 0 {
					ct := c.app.Config.Contests[c.ids[i]]
					dateStr = c.formatDate(ct.Date)
				}

				// Truncate/pad raw values before styling.
				dateVal := padOrTrunc(dateStr, 12)
				nameVal := padOrTrunc(name, 40)

				if i == c.cursor {
					dateVal = CursorStyle.Render(dateVal)
					nameVal = CursorStyle.Render(nameVal)
				}

				line := prefix + activeBadge + dateVal + nameVal
				b.WriteString(padOrTrunc(line, contentW))
				if i < len(c.names)-1 {
					b.WriteString("\n")
				}
			}
		}

		listContent = b.String()
		c.cachedList = listContent
		c.listSig = sig
	}
	return renderScrollableMenu("Configuration \u2014 Contests", listContent, &c.vp, &c.lastListContent, w, h)
}

func (c *ContestChooser) viewForm() string {
	w := c.width
	if w < 40 {
		w = 80
	}
	h := c.height
	if h < 10 {
		h = 24
	}

	// Render cache — skip expensive work on every frame.
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(w))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(h))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(c.focus))
	sb.WriteByte('|')
	sb.WriteString(c.nameInput.Value())
	sb.WriteByte('|')
	sb.WriteString(c.dateInput.Value())
	sb.WriteByte('|')
	sb.WriteString(c.nextInput.Value())
	sb.WriteByte('|')
	sb.WriteString(c.contInput.Value())
	sb.WriteByte('|')
	sb.WriteString(c.exchSentInput.Value())
	sb.WriteByte('|')
	sb.WriteString(c.exchRcvdInput.Value())
	sb.WriteString(strconv.FormatBool(c.inUse))
	sb.WriteString(strconv.FormatBool(c.prefillExchange))
	sb.WriteString(strconv.FormatBool(c.prefillExchangeRcvd))
	sb.WriteString(strconv.Itoa(int(c.mode)))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(c.vp.YOffset()))
	sig := sb.String()
	if c.formSig == sig && c.cachedForm != "" {
		return c.cachedForm
	}

	var b strings.Builder
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	title := "Create Contest"
	if c.mode == contestEdit {
		title = "Edit Contest"
	}

	// Use wider labels throughout this submenu.
	lbl := S.FormLabelCtx
	lblF := S.FormFocusedCtx

	// Name field.
	nl := lbl
	if c.focus == 0 {
		nl = lblF
	}
	b.WriteString("  ")
	b.WriteString(nl.Render("Name:"))
	b.WriteString(c.nameInput.View())
	b.WriteString("\n")

	// Date field.
	dl := lbl
	if c.focus == 1 {
		dl = lblF
	}
	b.WriteString("  ")
	b.WriteString(dl.Render("Date:"))
	b.WriteString(c.dateInput.View())
	b.WriteString("\n")

	// In Use checkbox — just below Date.
	c.renderCheckbox(&b, w, 2, "In use:", c.inUse)
	b.WriteString("\n")

	// Next QSO ID field.
	xl := lbl
	if c.focus == 3 {
		xl = lblF
	}
	b.WriteString("  ")
	b.WriteString(xl.Render("Next QSO / Rcvd serial:"))
	b.WriteString(c.nextInput.View())
	b.WriteString("\n")

	// Contest-ID field (ADIF).
	cid := strings.TrimSpace(c.contInput.Value())
	cidValid := cid != "" && isValidContestID(cid)
	cl := lbl
	cs := S.Input
	if c.focus == 4 {
		cl = lblF
	}
	if cidValid {
		cs = S.Success
	} else if cid != "" {
		cs = S.Warning
	}
	// Show ADIF description as trailing note, cycling with the ID.
	extra := ""
	if cidValid {
		extra = contestIDDesc(cid)
	}
	line := lipgloss.JoinHorizontal(lipgloss.Center, "  ", cl.Render("Contest ADIF ID:"), cs.Render(c.contInput.View()))
	if extra != "" {
		line = line + " " + DimStyle.Render(extra)
	}
	if c.focus == 4 {
		line = line + " " + DimStyle.Render("(Space)")
	}
	maxW := c.width - 4
	if maxW < 40 {
		maxW = 40
	}
	b.WriteString(padOrTrunc(line, maxW))
	b.WriteString("\n")

	// Prefill Exchange Sent checkbox.
	c.renderCheckbox(&b, w, 5, "Prefill Exchange Sent:", c.prefillExchange)

	// Indented exchange sent field.
	if c.prefillExchange {
		b.WriteString("\n")
		c.renderIndentedField(&b, 6, "  Exchange Sent:", &c.exchSentInput, "")
	}

	// Prefill Exchange Rcvd checkbox.
	b.WriteString("\n")
	c.renderCheckbox(&b, w, 7, "Prefill Exchange Rcvd:", c.prefillExchangeRcvd)

	// Indented exchange rcvd field.
	if c.prefillExchangeRcvd {
		b.WriteString("\n")
		c.renderIndentedField(&b, 8, "  Exchange Rcvd:", &c.exchRcvdInput, "")
	}

	// Marker reference section — shown below the form fields.
	b.WriteString("\n\n")
	b.WriteString("  Exchange markers")
	b.WriteString("\n\n")

	markers := [][2]string{
		{"@rst", "RST sent or received"},
		{"@serial", "Sent serial / rcvd serial placeholder"},
		{"@cqz", "DX station CQ zone"},
		{"@mycqz", "Your station CQ zone"},
		{"@itu", "DX station ITU zone"},
		{"@myitu", "Your station ITU zone"},
		{"@grid", "DX station grid square"},
		{"@mygrid", "Your station grid square"},
	}

	// Pre-built marker lines — avoids fmt.Sprintf on every frame.
	for _, m := range markers {
		b.WriteString("  ")
		b.WriteString(m[0])
		for i := len(m[0]); i < 12; i++ {
			b.WriteByte(' ')
		}
		b.WriteString(DimStyle.Render(m[1]))
		b.WriteByte('\n')
	}

	b.WriteString("\n")
	b.WriteString(DimStyle.Render("  Example: @rst @serial will generate 59 023"))

	// Use viewport for scrollable form body on small terminals.
	boxW := w
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	vpW := boxW - 4 // account for menu box left+right padding
	if vpW < 20 {
		vpW = 20
	}
	// Overhead: header(1) + blank row(1) + scroll hint(1) = 3 lines.
	vpH := contentH - 3
	if vpH < 4 {
		vpH = 4
	}
	c.vp.SetWidth(vpW)
	c.vp.SetHeight(vpH)
	bodyStr := b.String()
	if c.vp.TotalLineCount() == 0 || bodyStr != c.lastFormContent {
		c.vp.SetContent(bodyStr)
		c.lastFormContent = bodyStr
	}
	if c.vp.PastBottom() {
		c.vp.SetYOffset(c.vp.TotalLineCount() - c.vp.VisibleLineCount())
	}
	header := S.Title.Width(boxW).Render("Configuration \u2014 Contests \u2014 " + title)
	vpContent := c.vp.View()
	hintLine := DimStyle.Width(vpW).Render(scrollHint(c.vp))
	if hintLine == "" {
		hintLine = strings.Repeat(" ", vpW)
	}
	vpContent = lipgloss.JoinVertical(lipgloss.Left, vpContent, hintLine)
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	result := lipgloss.JoinVertical(lipgloss.Left, header, "", box)
	c.cachedForm = result
	c.formSig = sig
	return result
}

func (c *ContestChooser) renderCheckbox(b *strings.Builder, w, focusIdx int, label string, checked bool) {
	cb := "[ ]"
	if checked {
		cb = "[x]"
	}
	prefix := "  "
	lbl := S.FormLabelCtx.Align(lipgloss.Left).Render(label)
	if c.focus == focusIdx {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedCtx.Align(lipgloss.Left).Render(label)
		cb = CursorStyle.Render(cb) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, cb),
		w-4))
}

func (c *ContestChooser) renderIndentedField(b *strings.Builder, focusIdx int, label string, ti *textinput.Model, extra string) {
	prefix := "  "
	lbl := S.FormLabelCtx.Align(lipgloss.Left).Render(label)
	val := ti.View()
	if c.focus == focusIdx {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedCtx.Align(lipgloss.Left).Render(label)
	}
	// Show "See reference below" when the field is empty and not focused.
	if strings.TrimSpace(ti.Value()) == "" && c.focus != focusIdx {
		val = DimStyle.Render("See reference below")
	}
	line := lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, val)
	if extra != "" {
		line = line + " " + DimStyle.Render(extra)
	}
	maxW := c.width - 4
	if maxW < 40 {
		maxW = 40
	}
	b.WriteString(padOrTrunc(line, maxW))
}
