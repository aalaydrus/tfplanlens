package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/aalaydrus/tfplanlens/internal/analyze"
)

// parseKeys splits raw terminal input into key presses. Escape sequences
// (arrows, page keys) arrive in one read; a lone ESC is the esc key.
func parseKeys(b []byte) []Key {
	var keys []Key
	for len(b) > 0 {
		switch {
		case b[0] == 0x1b && len(b) == 1:
			keys, b = append(keys, "esc"), b[1:]
		case b[0] == 0x1b && (b[1] == '[' || b[1] == 'O'):
			i := 2
			for i < len(b) && (b[i] >= '0' && b[i] <= '9' || b[i] == ';') {
				i++
			}
			if i >= len(b) {
				return keys
			}
			name := map[string]Key{
				"A": "up", "B": "down", "C": "right", "D": "left", "H": "home", "F": "end",
				"5~": "pgup", "6~": "pgdown", "1~": "home", "7~": "home", "4~": "end", "8~": "end",
			}[string(b[2:i+1])]
			if name != "" {
				keys = append(keys, name)
			}
			b = b[i+1:]
		case b[0] == 0x1b:
			keys, b = append(keys, "esc"), b[1:]
		case b[0] == '\r' || b[0] == '\n':
			keys, b = append(keys, "enter"), b[1:]
		case b[0] == 0x7f || b[0] == 0x08:
			keys, b = append(keys, "backspace"), b[1:]
		case b[0] == 3:
			keys, b = append(keys, "ctrl+c"), b[1:]
		case b[0] < ' ':
			b = b[1:]
		default:
			r := []rune(string(b))
			keys, b = append(keys, Key(string(r[0]))), b[len(string(r[0])):]
		}
	}
	return keys
}

// Run shows a on the terminal until the user quits. It needs a terminal on
// both stdin and stdout and never queries the terminal for its colors.
func Run(a *analyze.Analysis) error {
	in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	if !term.IsTerminal(in) || !term.IsTerminal(out) {
		return errors.New("tui needs an interactive terminal; use `report` for pipes and files")
	}
	old, err := term.MakeRaw(in)
	if err != nil {
		return err
	}
	// Alternate screen, hidden cursor; always put the terminal back.
	io.WriteString(os.Stdout, "\x1b[?1049h\x1b[?25l")
	defer func() {
		io.WriteString(os.Stdout, "\x1b[?25h\x1b[?1049l")
		term.Restore(in, old)
	}()

	m := New(a)
	m.Color = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	keys := make(chan []Key)
	readErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				readErr <- err
				return
			}
			keys <- parseKeys(buf[:n])
		}
	}()

	lastW, lastH := -1, -1
	draw := func() {
		w, h, err := term.GetSize(out)
		if err == nil {
			m.Resize(w, h)
		}
		frame := strings.ReplaceAll(m.View(), "\n", "\x1b[K\r\n")
		fmt.Fprint(os.Stdout, "\x1b[H"+frame+"\x1b[K\x1b[J")
		lastW, lastH = w, h
	}
	draw()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case ks := <-keys:
			for _, k := range ks {
				if m.Update(k) {
					return nil
				}
			}
			draw()
		case err := <-readErr:
			if err == io.EOF {
				return nil
			}
			return err
		case <-tick.C:
			if w, h, err := term.GetSize(out); err == nil && (w != lastW || h != lastH) {
				draw()
			}
		}
	}
}
