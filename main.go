package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log(err.Error(), nil)
	}

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "savelogs" {
		saveLogs()
	}
}

const (
	Height = 50
	Width  = 50
)

type errMsg error

type model struct {
	frameBuffer string
	textarea    textarea.Model
	err         error
	startTime   int64
	programs    [3]*vm.Program
	compileErrs [3]error
	hint        string
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

func shaderEnv(t, x, y float64) map[string]any {
	return map[string]any{
		"t": t, "x": x, "y": y,
		// Constants
		"PI": math.Pi, "TAU": math.Pi * 2, "E": math.E,
		// Trig
		"sin": math.Sin, "cos": math.Cos, "tan": math.Tan,
		"atan": math.Atan, "atan2": math.Atan2,
		// Power / exp / log
		"pow": math.Pow, "sqrt": math.Sqrt, "exp": math.Exp,
		"log": math.Log, "log2": math.Log2,
		// Rounding
		"floor": math.Floor, "ceil": math.Ceil, "round": math.Round,
		"abs": math.Abs,
		// Range
		"min": math.Min, "max": math.Max, "mod": math.Mod,
		// Shader-specific
		"fract": func(x float64) float64 { return x - math.Floor(x) },
		"clamp": func(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) },
		"mix":   func(a, b, t float64) float64 { return a*(1-t) + b*t },
		"step": func(edge, x float64) float64 {
			if x < edge {
				return 0
			}
			return 1
		},
		"smoothstep": func(e0, e1, x float64) float64 {
			t := math.Max(0, math.Min(1, (x-e0)/(e1-e0)))
			return t * t * (3 - 2*t)
		},
		"sign": func(x float64) float64 {
			if x < 0 {
				return -1
			}
			if x > 0 {
				return 1
			}
			return 0
		},
		"length": func(x, y float64) float64 { return math.Sqrt(x*x + y*y) },
		"fmod":   func(a, b float64) float64 { return math.Mod(a, b) },
	}
}

func compileOpts() []expr.Option {
	return []expr.Option{
		expr.Env(shaderEnv(0, 0, 0)),
		expr.AsFloat64(),
		expr.Function("fmod", func(params ...any) (any, error) {
			return math.Mod(params[0].(float64), params[1].(float64)), nil
		}, new(func(float64, float64) float64)),
		expr.Operator("%", "fmod"),
	}
}

func (m *model) CompilePrograms() {
	text := m.textarea.Value()
	lines := strings.Split(text, "\n")

	m.programs = [3]*vm.Program{}
	m.compileErrs = [3]error{}

	opts := compileOpts()

	log("CompilePrograms: %d lines", len(lines))
	for i := 0; i < 3 && i < len(lines); i++ {
		log("compiling line %d: %q", i, lines[i])
		p, err := expr.Compile(lines[i], opts...)
		if err != nil {
			log("compile error line %d: %s", i, err)
			m.programs[i] = nil
			m.compileErrs[i] = err
			continue
		}
		m.programs[i] = p
		m.compileErrs[i] = nil
		log("compiled line %d OK, program=%p", i, p)
	}
	log("programs after compile: [%p, %p, %p]", m.programs[0], m.programs[1], m.programs[2])
}

func clampColor(v float64) int {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return int(math.Max(0, math.Min(255, v)))
}

func (m model) DoMath(t, x, y float64) (int, int, int) {
	env := shaderEnv(t, x, y)

	var rgb [3]int
	for i, p := range m.programs {
		if p == nil {
			continue
		}
		output, err := expr.Run(p, env)
		if err != nil {
			continue
		}
		rgb[i] = clampColor(output.(float64))
	}

	return rgb[0], rgb[1], rgb[2]
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
		if m.programs[0] == nil && m.programs[1] == nil && m.programs[2] == nil {
			return m, doTick()
		}
		buf := make([]byte, 0, Height*Width*25)
		t := float64(time.Now().UnixMilli() - m.startTime)

		for y := range Height {
			for x := range Width {
				r, g, b := m.DoMath(t, float64(x)/float64(Width-1), float64(Height-1-y)/float64(Height-1))
				buf = append(buf, "\033[48;2;"...)
				buf = strconv.AppendInt(buf, int64(r), 10)
				buf = append(buf, ';')
				buf = strconv.AppendInt(buf, int64(g), 10)
				buf = append(buf, ';')
				buf = strconv.AppendInt(buf, int64(b), 10)
				buf = append(buf, "m  \033[0m"...)
			}
			buf = append(buf, '\n')
		}

		m.frameBuffer = string(buf)
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

	f := ""
	if m.compileErrs[0] != nil || m.compileErrs[1] != nil || m.compileErrs[2] != nil {
		errLines := []string{}
		for i, err := range m.compileErrs {
			if err != nil {
				errLines = append(errLines, fmt.Sprintf("Line %d: %s", i+1, err.Error()))
			}
		}
		f = m.textarea.View() + "\n" + m.hint + "\n" + strings.Join(errLines, "\n") + "\n" + m.frameBuffer
	} else {
		f = m.textarea.View() + "\n" + m.hint + "\n" + m.frameBuffer
	}

	v := tea.NewView(f)
	v.Cursor = c
	v.AltScreen = true
	return v
}
