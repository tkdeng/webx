package webx

import (
	"math"
	"os"

	"github.com/tkdeng/goutil"
	"github.com/tkdeng/regex"
)

type ThemeConfig struct {
	FontSize string
	Font     map[string]string

	Scheme string
	ForceScheme bool
	Theme  map[string]*struct {
		Scheme string

		BGChroma    float64
		TextChroma  float64
		FGChroma    float64
		ColorChroma float64

		BGDark  uint8
		BG      uint8
		BGLight uint8
		FG      uint8

		Text      uint8
		TextMuted uint8
	}

	Colors map[string]*struct {
		Hue   int16
		Dark  uint8
		Light uint8
	}

	Vars map[string]string
}

func (comp *compiler) compTheme() {
	config := ThemeConfig{}
	err := goutil.ReadConfig(comp.config.Root+"/theme/theme.yml", &config)
	if err != nil {
		return
	}

	// fix theme value constraints
	for name, theme := range config.Theme {
		if theme.BGChroma >= 1 {
			config.Theme[name].BGChroma = math.Round(theme.BGChroma*400) / 100000
		}
		if theme.TextChroma >= 1 {
			config.Theme[name].TextChroma = math.Round(theme.TextChroma*400) / 100000
		}
		if theme.FGChroma >= 1 {
			config.Theme[name].FGChroma = math.Round(theme.FGChroma*400) / 100000
		}
		if theme.ColorChroma >= 1 {
			config.Theme[name].ColorChroma = math.Round(theme.ColorChroma*400) / 100000
		}

		config.Theme[name].BGChroma = math.Max(0, math.Min(0.4, config.Theme[name].BGChroma))
		config.Theme[name].TextChroma = math.Max(0, math.Min(0.4, config.Theme[name].TextChroma))
		config.Theme[name].FGChroma = math.Max(0, math.Min(0.4, config.Theme[name].FGChroma))
		config.Theme[name].ColorChroma = math.Max(0, math.Min(0.4, config.Theme[name].ColorChroma))

		config.Theme[name].BG = uint8(math.Max(5, math.Min(95, float64(theme.BG))))
		config.Theme[name].BGDark = uint8(math.Max(0, math.Max(float64(config.Theme[name].BG)-50, math.Min(float64(config.Theme[name].BG-5), float64(theme.BGDark)))))
		config.Theme[name].BGLight = uint8(math.Min(100, math.Max(float64(config.Theme[name].BG+5), math.Min(float64(config.Theme[name].BG)+50, float64(theme.BGLight)))))

		if config.Theme[name].BG < 50 {
			config.Theme[name].FG = uint8(math.Min(100, math.Max(float64(config.Theme[name].BGLight+10), float64(theme.FG))))
		} else {
			config.Theme[name].FG = uint8(math.Min(float64(config.Theme[name].BGDark-10), math.Max(0, float64(theme.FG))))
		}

		textMin := float64(0)
		textMax := float64(100)
		if config.Theme[name].BG < 35 {
			textMin = 65
		} else if config.Theme[name].BG > 65 {
			textMax = 35
		}

		config.Theme[name].Text = uint8(math.Max(textMin, math.Min(textMax, float64(theme.Text))))
		config.Theme[name].TextMuted = uint8(math.Max(textMin, math.Min(textMax, float64(theme.TextMuted))))
	}

	var primaryHue *goutil.Degree
	if color, ok := config.Colors["primary"]; ok {
		primaryHue = goutil.Deg(color.Hue)
	}

	// fix color value constraints
	for name, color := range config.Colors {
		config.Colors[name].Light = uint8(math.Max(10, math.Min(100, float64(color.Light))))
		config.Colors[name].Dark = uint8(math.Max(0, math.Min(float64(config.Colors[name].Light)-10, float64(color.Dark))))

		hue := goutil.Deg(color.Hue)

		if primaryHue != nil {
			if name == "accent" && hue.Distance(primaryHue) < 10 {
				hue.Set(primaryHue.Get())
			} else if name == "link" && hue.Distance(primaryHue) < 20 {
				hue.Set(primaryHue.Get())
			} else if name == "confirm" || name == "warn" {
				var accentHue *goutil.Degree
				if color, ok := config.Colors["accent"]; ok {
					accentHue = goutil.Deg(color.Hue)
					if accentHue.Distance(primaryHue) < 10 {
						accentHue.Set(primaryHue.Get())
					}
				} else {
					accentHue = goutil.Deg(primaryHue.Get())
				}

				if name == "confirm" {
					hue.SetClamp(125, 175)
				} else if name == "warn" {
					hue.SetClamp(345, 70)
				}

				if hue.Distance(primaryHue) < 15 || hue.Distance(accentHue) < 15 {
					if hue.Get() < int16(math.Min(float64(primaryHue.Get()), float64(accentHue.Get()))) {
						hue.Rotate(-25)
					} else if hue.Get() > int16(math.Max(float64(primaryHue.Get()), float64(accentHue.Get()))) {
						hue.Rotate(25)
					} else {
						hue.Rotate(15)
						if hue.Distance(primaryHue) < 15 || hue.Distance(accentHue) < 15 {
							hue.Rotate(-30)
						}
					}
				}

				if hue.Distance(primaryHue) < 15 || hue.Distance(accentHue) < 15 {
					if name == "confirm" {
						if h := goutil.Deg(150); h.Distance(primaryHue) < 15 && h.Distance(accentHue) < 15 {
							hue.Set(h.Get())
						} else if h := goutil.Deg(165); h.Distance(primaryHue) < 15 && h.Distance(accentHue) < 15 {
							hue.Set(h.Get())
						} else if h := goutil.Deg(135); h.Distance(primaryHue) < 15 && h.Distance(accentHue) < 15 {
							hue.Set(h.Get())
						}
					} else if name == "warn" {
						if h := goutil.Deg(25); h.Distance(primaryHue) < 15 && h.Distance(accentHue) < 15 {
							hue.Set(h.Get())
						} else if h := goutil.Deg(0); h.Distance(primaryHue) < 15 && h.Distance(accentHue) < 15 {
							hue.Set(h.Get())
						} else if h := goutil.Deg(50); h.Distance(primaryHue) < 15 && h.Distance(accentHue) < 15 {
							hue.Set(h.Get())
						}
					}
				}
			}
		}

		config.Colors[name].Hue = hue.Get()
	}

	cleanName := func(str string) []byte {
		return regex.Comp(`[^\w_\-]`).RepLit([]byte(str), []byte{})
	}

	// generate config.css theme file
	buf := []byte(":root {\n")

	if config.FontSize != "" {
		buf = append(buf, regex.JoinBytes(`  font-size: `, config.FontSize, ';', '\n')...)
	}

	buf = append(buf, '\n')

	for name, font := range config.Font {
		buf = append(buf, regex.JoinBytes(`  --ff-`, cleanName(name), `: `, font, ';', '\n')...)
	}

	buf = append(buf, '\n')

	for name, color := range config.Colors {
		buf = append(buf, regex.JoinBytes(`  --h-`, cleanName(name), `: `, int(color.Hue), ';', '\n')...)
	}

	buf = append(buf, '\n')

	if theme, ok := config.Theme[config.Scheme]; ok {
		/* if theme.Scheme == "dark" {
			buf = append(buf, []byte("  color-scheme: dark light;\n")...)
		} else if theme.Scheme == "light" {
			buf = append(buf, []byte("  color-scheme: light dark;\n")...)
		} else if theme.Scheme != "" {
			buf = append(buf, regex.JoinBytes(`  color-scheme: `, theme.Scheme, ';', '\n')...)
		} */

		buf = append(buf, regex.JoinBytes(`  color-scheme: `, theme.Scheme, ';', '\n')...)

		buf = append(buf, '\n')

		buf = append(buf, regex.JoinBytes(`  --c-bg: `, theme.BGChroma, ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --c-text: `, theme.TextChroma, ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --c-fg: `, theme.FGChroma, ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --c-color: `, theme.ColorChroma, ';', '\n')...)

		buf = append(buf, '\n')

		buf = append(buf, regex.JoinBytes(`  --l-bg-dark: `, int(theme.BGDark), '%', ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --l-bg: `, int(theme.BG), '%', ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --l-bg-light: `, int(theme.BGLight), '%', ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --l-fg: `, int(theme.FG), '%', ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --l-text: `, int(theme.Text), '%', ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --l-text-muted: `, int(theme.TextMuted), '%', ';', '\n')...)

		buf = append(buf, '\n')

		for name, color := range config.Colors {
			if theme.Scheme == "dark" {
				buf = append(buf, regex.JoinBytes(`  --l-`, cleanName(name), `: `, int(color.Light), '%', ';', '\n')...)
			} else {
				buf = append(buf, regex.JoinBytes(`  --l-`, cleanName(name), `: `, int(color.Dark), '%', ';', '\n')...)
			}
		}
	}

	buf = append(buf, '\n')

	for key, val := range config.Vars {
		buf = append(buf, regex.JoinBytes(`  --`, key, `: `, val, ';', '\n')...)
	}

	buf = append(buf, []byte("}\n")...)

	if !config.ForceScheme {
		for name, theme := range config.Theme {
			if name == config.Scheme {
				continue
			}
	
			buf = append(buf, regex.JoinBytes(
				'\n', `@media (prefers-color-scheme: `, cleanName(name), `) {`, '\n',
				`  :root {`, '\n',
			)...)
	
			/* if theme.Scheme == "dark" {
				buf = append(buf, []byte("    color-scheme: dark light;\n")...)
			} else if theme.Scheme == "light" {
				buf = append(buf, []byte("    color-scheme: light dark;\n")...)
			} else if theme.Scheme != "" {
				buf = append(buf, regex.JoinBytes(`    color-scheme: `, theme.Scheme, ';', '\n')...)
			} */
	
			buf = append(buf, regex.JoinBytes(`    color-scheme: `, theme.Scheme, ';', '\n')...)
	
			buf = append(buf, '\n')
	
			buf = append(buf, regex.JoinBytes(`    --c-bg: `, theme.BGChroma, ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --c-text: `, theme.TextChroma, ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --c-fg: `, theme.FGChroma, ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --c-color: `, theme.ColorChroma, ';', '\n')...)
	
			buf = append(buf, '\n')
	
			buf = append(buf, regex.JoinBytes(`    --l-bg-dark: `, int(theme.BGDark), '%', ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --l-bg: `, int(theme.BG), '%', ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --l-bg-light: `, int(theme.BGLight), '%', ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --l-fg: `, int(theme.FG), '%', ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --l-text: `, int(theme.Text), '%', ';', '\n')...)
			buf = append(buf, regex.JoinBytes(`    --l-text-muted: `, int(theme.TextMuted), '%', ';', '\n')...)
	
			buf = append(buf, '\n')
	
			for name, color := range config.Colors {
				if theme.Scheme == "dark" {
					buf = append(buf, regex.JoinBytes(`    --l-`, cleanName(name), `: `, int(color.Light), '%', ';', '\n')...)
				} else {
					buf = append(buf, regex.JoinBytes(`    --l-`, cleanName(name), `: `, int(color.Dark), '%', ';', '\n')...)
				}
			}
	
			buf = append(buf, []byte("  }\n}\n")...)
		}
	}


	// add basic css color vars
	buf = append(buf, []byte("\n:root {\n")...)

	buf = append(buf, []byte("  --h-color: var(--h-primary);\n")...)
	buf = append(buf, []byte("  --l-color: var(--l-primary);\n")...)

	buf = append(buf, '\n')

	buf = append(buf, regex.JoinBytes(`  --bg-dark: oklch(var(--l-bg-dark) var(--c-bg) var(--h-color))`, ';', '\n')...)
	buf = append(buf, regex.JoinBytes(`  --bg: oklch(var(--l-bg) var(--c-bg) var(--h-color))`, ';', '\n')...)
	buf = append(buf, regex.JoinBytes(`  --bg-light: oklch(var(--l-bg-light) var(--c-bg) var(--h-color))`, ';', '\n')...)
	buf = append(buf, regex.JoinBytes(`  --fg: oklch(var(--l-fg) var(--c-bg) var(--h-color))`, ';', '\n')...)
	buf = append(buf, regex.JoinBytes(`  --text: oklch(var(--l-text) var(--c-text) var(--h-color))`, ';', '\n')...)
	buf = append(buf, regex.JoinBytes(`  --text-muted: oklch(var(--l-text-muted) var(--c-text) var(--h-color))`, ';', '\n')...)

	buf = append(buf, '\n')

	buf = append(buf, regex.JoinBytes(`  --color: oklch(var(--l-color) var(--c-color) var(--h-color))`, ';', '\n')...)
	buf = append(buf, regex.JoinBytes(`  --color-fg: oklch(var(--l-color) var(--c-fg) var(--h-color))`, ';', '\n')...)

	for name := range config.Colors {
		buf = append(buf, '\n')

		buf = append(buf, regex.JoinBytes(`  --`, cleanName(name), `: oklch(var(--l-`, cleanName(name), `) var(--c-color) var(--h-`, cleanName(name), `))`, ';', '\n')...)
		buf = append(buf, regex.JoinBytes(`  --`, cleanName(name), `-fg: oklch(var(--l-`, cleanName(name), `) var(--c-fg) var(--h-`, cleanName(name), `))`, ';', '\n')...)
	}

	buf = append(buf, []byte("}\n")...)

	if !comp.config.DebugMode {
		minifyCSS(&buf, "config")
	}

	os.WriteFile(comp.config.Root+"/theme/config.css", buf, 0755)
}
