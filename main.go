package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var (
	savelog = flag.Bool("savelogs", false, "save logs to shader.log on exit")
	profile = flag.Bool("profile", false, "write CPU profile to cpu.prof")
)

func main() {
	flag.IntVar(&Height, "h", Height, "grid height in pixels")
	flag.IntVar(&Width, "w", Width, "grid width in pixels")
	flag.Parse()

	if *profile {
		f, _ := os.Create("cpu.prof")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log(err.Error(), nil)
	}

	if *savelog {
		saveLogs()
	}
}

var (
	Height = 50
	Width  = 50
)

type errMsg error

type channelExpr struct {
	program    *vm.Program
	compileErr error
}

type model struct {
	frameBytes []byte
	textarea   textarea.Model
	err        error
	startTime  int64
	redExpr    channelExpr
	greenExpr  channelExpr
	blueExpr   channelExpr
	hint       string
}

func initialModel() model {
	ti := textarea.New()
	ti.SetVirtualCursor(false)
	ti.SetWidth(100)
	ti.Focus()

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("ctrl+s to compile, ctrl+d to clear, ctrl+c to quit")

	defaultShader := "sin(t * 0.001 + x * 6.0) * 127.0 + 128.0\n" +
		"sin(t * 0.001 + y * 6.0) * 127.0 + 128.0\n" +
		"sin(t * 0.001 + (x + y) * 3.0) * 127.0 + 128.0"
	ti.SetValue(defaultShader)

	m := model{
		hint:      hint,
		textarea:  ti,
		err:       nil,
		startTime: time.Now().UnixMilli(),
	}
	m.CompilePrograms()
	return m
}

type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(time.Second/60, func(tick time.Time) tea.Msg {
		return TickMsg(tick)
	})
}

func (m model) Init() tea.Cmd {
	return doTick()
}

func compileOpts() []expr.Option {
	shaderEnv["t"] = 0.0
	shaderEnv["x"] = 0.0
	shaderEnv["y"] = 0.0
	return []expr.Option{
		expr.Env(shaderEnv),
		expr.AsFloat64(),
		expr.Function("fmod", func(params ...any) (any, error) {
			return math.Mod(params[0].(float64), params[1].(float64)), nil
		}, new(func(float64, float64) float64)),
		expr.Operator("%", "fmod"),
	}
}

func compileChannel(line string, opts []expr.Option) channelExpr {
	p, err := expr.Compile(line, opts...)
	if err != nil {
		return channelExpr{compileErr: err}
	}
	return channelExpr{program: p}
}

func (m *model) CompilePrograms() {
	lines := strings.SplitN(m.textarea.Value(), "\n", 3)
	opts := compileOpts()

	m.redExpr = channelExpr{}
	m.greenExpr = channelExpr{}
	m.blueExpr = channelExpr{}

	if len(lines) > 0 {
		m.redExpr = compileChannel(lines[0], opts)
	}
	if len(lines) > 1 {
		m.greenExpr = compileChannel(lines[1], opts)
	}
	if len(lines) > 2 {
		m.blueExpr = compileChannel(lines[2], opts)
	}
}

func clampColor(v float64) int {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return int(math.Max(0, math.Min(255, v)))
}

func evalChannel(ch channelExpr, env map[string]any) int {
	if ch.program == nil {
		return 0
	}
	output, err := expr.Run(ch.program, env)
	if err != nil {
		return 0
	}
	return clampColor(output.(float64))
}

func (m *model) DoMath(t, x, y float64) (int, int, int) {
	shaderEnv["t"] = t
	shaderEnv["x"] = x
	shaderEnv["y"] = y
	return evalChannel(m.redExpr, shaderEnv), evalChannel(m.greenExpr, shaderEnv), evalChannel(m.blueExpr, shaderEnv)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+s":
			m.CompilePrograms()
			log("%s", m.textarea.Value())
		case "ctrl+d":
			m.textarea.SetValue("")
		case "ctrl+c":
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	case TickMsg:
		if m.redExpr.program == nil && m.greenExpr.program == nil && m.blueExpr.program == nil {
			return m, doTick()
		}
		t := float64(time.Now().UnixMilli() - m.startTime)

		m.frameBytes = m.frameBytes[:0]

		for y := range Height {
			for x := range Width {
				r, g, b := m.DoMath(t, float64(x)/float64(Width-1), float64(Height-1-y)/float64(Height-1))
				m.frameBytes = append(m.frameBytes, "\033[48;2;"...)
				m.frameBytes = strconv.AppendInt(m.frameBytes, int64(r), 10)
				m.frameBytes = append(m.frameBytes, ';')
				m.frameBytes = strconv.AppendInt(m.frameBytes, int64(g), 10)
				m.frameBytes = append(m.frameBytes, ';')
				m.frameBytes = strconv.AppendInt(m.frameBytes, int64(b), 10)
				m.frameBytes = append(m.frameBytes, "m  \033[0m"...)
			}
			m.frameBytes = append(m.frameBytes, '\n')
		}

		return m, doTick()

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	var c *tea.Cursor
	if !m.textarea.VirtualCursor() {
		c = m.textarea.Cursor()
	}

	content := ""
	var errLines []string
	for i, ch := range []channelExpr{m.redExpr, m.greenExpr, m.blueExpr} {
		if ch.compileErr != nil {
			errLines = append(errLines, fmt.Sprintf("Line %d: %s", i+1, ch.compileErr.Error()))
		}
	}
	if len(errLines) > 0 {
		content = m.textarea.View() + "\n" + m.hint + "\n" + strings.Join(errLines, "\n") + "\n" + string(m.frameBytes)
	} else {
		content = m.textarea.View() + "\n" + m.hint + "\n" + string(m.frameBytes)
	}

	v := tea.View{Content: content}
	v.Cursor = c
	v.AltScreen = true
	return v
}
