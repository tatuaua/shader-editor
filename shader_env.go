package main

import "math"

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
