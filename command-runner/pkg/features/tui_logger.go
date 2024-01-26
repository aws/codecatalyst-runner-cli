package features

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var tuiApp tuiApplication

// TUILogger is a Feature to display a UI
func TUILogger(planID string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		planExecution := tuiApp.Add(ctx, planID)
		ctx = planExecution.Start(ctx)
		err := e(ctx)
		if err == nil {
			planExecution.Success()
		} else if errors.Is(err, common.ErrDefer) {
			planExecution.Defer()
		} else {
			planExecution.Failure(err)
		}
		return err
	}
}

type tuiApplication struct {
	executions  map[string]*planExecution
	app         *tview.Application
	actionsView *tview.Table
	pagesView   *tview.Pages
	done        chan bool
	mu          sync.Mutex
	pending     int
}

func (t *tuiApplication) Add(ctx context.Context, planID string) *planExecution {
	tuiApp.Start(ctx)
	t.mu.Lock()
	defer t.mu.Unlock()
	if pe, ok := t.executions[planID]; ok {
		return pe
	}
	pe := newPlanExecution(planID, t)
	t.executions[planID] = pe
	i := len(t.executions) - 1
	t.actionsView.SetCell(i, 0, pe.cell)
	pe.logView.SetChangedFunc(func() {
		pe.logView.ScrollToEnd()
		t.app.Draw()
	})
	t.pagesView.AddPage(strconv.Itoa(i), pe.logView, true, false)
	return pe
}

func (t *tuiApplication) Start(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.app != nil {
		return
	}
	t.app = tview.NewApplication()
	t.pagesView = tview.NewPages()
	t.actionsView = tview.NewTable().SetBorders(false).SetSelectable(true, true)
	t.actionsView.SetBorder(true).SetTitle("Actions")

	t.executions = make(map[string]*planExecution)

	priorRow := 0
	t.actionsView.SetSelectionChangedFunc(func(row int, column int) {
		t.pagesView.HidePage(strconv.Itoa(priorRow))
		t.pagesView.ShowPage(strconv.Itoa(row))
		if row >= 0 && row < t.actionsView.GetRowCount() {
			pe := t.actionsView.GetCell(row, column).GetReference().(*planExecution)
			log.Logger = log.Logger.Output(pe.logWriter)
		}
		priorRow = row
	})

	flex := tview.NewFlex().
		AddItem(t.actionsView, 0, 1, true).
		AddItem(t.pagesView, 0, 5, false)

	t.app = t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC && t.pending > 0 {
			process, _ := os.FindProcess(os.Getpid())
			if err := process.Signal(syscall.SIGINT); err != nil {
				log.Error().Err(err).Msg("failed to send SIGINT to process")
			}
		}
		return event
	})
	t.done = make(chan bool, 1)
	go func() {
		if err := t.app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
			log.Error().Err(err).Msg("failed to start TUI app")
		}
		t.done <- true
	}()
	go t.refreshLoadingIcons(ctx)
}

var loadingIcons = []rune("🕛🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚")

func (t *tuiApplication) refreshLoadingIcons(ctx context.Context) {
	index := 0
	for ctx.Err() == nil {
		icon := string(loadingIcons[index%len(loadingIcons)])
		redraw := false
		for _, pe := range t.executions {
			if pe.running {
				pe.cell.SetText(fmt.Sprintf("%s %s", icon, pe.id))
				redraw = true
			}
		}
		index++
		if redraw {
			t.app.Draw()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (t *tuiApplication) Close() {
	if t.app == nil {
		return
	}
	<-t.done
	t.app.Stop()
}

func (t *tuiApplication) HandleStart(planID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending == 0 {
		for row := 0; row < t.actionsView.GetRowCount(); row++ {
			if t.actionsView.GetCell(row, 0).GetReference().(*planExecution).id == planID {
				t.actionsView.Select(row, 0)
				break
			}
		}
	}
	t.pending++
}
func (t *tuiApplication) HandleSuccess(_ string) {
	t.decrementPending()
}
func (t *tuiApplication) HandleFailure(_ string) {
	t.decrementPending()
}

func (t *tuiApplication) decrementPending() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending == 0 {
		return
	}
	t.pending--
	if t.pending == 0 {
		t.Close()
	}
}

type planExecutionEventHandler interface {
	HandleStart(planID string)
	HandleSuccess(planID string)
	HandleFailure(planID string)
}

type planExecution struct {
	id           string
	started      bool
	running      bool
	cell         *tview.TableCell
	logView      *tview.TextView
	logWriter    io.Writer
	eventHandler planExecutionEventHandler
}

func newPlanExecution(planID string, eventHandler planExecutionEventHandler) *planExecution {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)
	textView.SetBorder(true).SetTitle(fmt.Sprintf("%s Logs", planID))

	cell := tview.NewTableCell(planID)
	cell.SetText(fmt.Sprintf("⏸️ %s", planID))
	cell.SetTextColor(tcell.ColorYellow)

	pe := &planExecution{
		id:           planID,
		cell:         cell,
		logView:      textView,
		logWriter:    tview.ANSIWriter(textView),
		eventHandler: eventHandler,
	}
	cell.SetReference(pe)
	return pe
}

func (pe *planExecution) Start(ctx context.Context) context.Context {
	ctx = log.Logger.Output(zerolog.ConsoleWriter{Out: pe.logWriter}).WithContext(ctx)
	pe.cell.SetText(fmt.Sprintf("%s %s", string(loadingIcons[0]), pe.id))
	pe.cell.SetTextColor(tcell.ColorWhite)
	pe.running = true
	if pe.started {
		return ctx
	}
	pe.started = true
	pe.eventHandler.HandleStart(pe.id)
	return ctx
}

func (pe *planExecution) Success() {
	pe.running = false
	pe.cell.SetText(fmt.Sprintf("✅ %s", pe.id))
	pe.cell.SetTextColor(tcell.ColorGreen)
	pe.eventHandler.HandleSuccess(pe.id)
}

func (pe *planExecution) Defer() {
	pe.running = false
	pe.cell.SetText(fmt.Sprintf("⏸️ %s", pe.id))
	pe.cell.SetTextColor(tcell.ColorYellow)
}

func (pe *planExecution) Failure(err error) {
	pe.running = false
	pe.cell.SetText(fmt.Sprintf("❌ %s", pe.id))
	pe.cell.SetTextColor(tcell.ColorRed)
	pe.eventHandler.HandleFailure(pe.id)
	logger := log.Logger.Output(zerolog.ConsoleWriter{Out: pe.logWriter})
	logger.Error().Msg(err.Error())
}
