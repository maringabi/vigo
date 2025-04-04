package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-isatty"

	"io"
	"log"
	"os"
)

const (
	tabSize = 4
)

func main() {
	var input []byte
	var filename string

	if len(os.Args) > 1 {
		filename = os.Args[1]
		var err error
		input, err = os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
	} else if !isatty.IsTerminal(os.Stdin.Fd()) {
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		input = bytes
	}

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}
	s.EnableMouse()

	v := newViewFromBuffer(newBuffer(string(input), filename), s)

	// Initially everything needs to be drawn
	redraw := 2
	for {
		if redraw == 2 {
			s.Clear()
			v.display()
			v.cursor.display()
			s.Show()
		} else if redraw == 1 {
			v.cursor.display()
			s.Show()
		}

		event := s.PollEvent()

		switch e := event.(type) {
		case *tcell.EventKey:
			if e.Key() == tcell.KeyCtrlQ {
				s.Fini()
				os.Exit(0)
			}
		}
		redraw = v.handleEvent(event)
	}
}

