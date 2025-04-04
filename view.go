package main

import (
	"github.com/gdamore/tcell/v2"
	"strings"
)

type View struct {
	cursor  Cursor
	topline int
	linesN  int
	colsN   int

	buf           *Buffer
	mouseReleased bool

	s tcell.Screen
}

func newViewFromBuffer(buf *Buffer, s tcell.Screen) *View {
	v := new(View)

	v.buf = buf
	v.s = s
	w, h := s.Size()

	v.topline = 0
	v.linesN = h
	v.colsN = w
	v.cursor = *NewCursor(v)

	return v
}

// Returns an int describing how the screen needs to be redrawn
// 0: Screen does not need to be redrawn
// 1: Only the cursor needs to be redrawn
// 2: Everything needs to be redrawn
func (v *View) handleEvent(event tcell.Event) int {
	var ret int
	switch e := event.(type) {
	case *tcell.EventKey:
		switch e.Key() {
		case tcell.KeyEsc:
			v.cursor.mode = normal
		case tcell.KeyUp:
			v.cursor.up()
			ret = 1
		case tcell.KeyDown:
			v.cursor.down()
			ret = 1
		case tcell.KeyLeft:
			v.cursor.left()
			ret = 1
		case tcell.KeyRight:
			v.cursor.right()
			ret = 1
		case tcell.KeyEnter:
			v.buf.insert(v.cursor.loc, "\n")
			v.cursor.right()
			ret = 2
		case tcell.KeyBackspace2:
			if v.cursor.loc > 0 {
				v.cursor.left()
				v.buf.remove(v.cursor.loc, v.cursor.loc+1)
				ret = 2
			}
		case tcell.KeyTab:
			v.buf.insert(v.cursor.loc, "\t")
			v.cursor.right()
			ret = 2
		case tcell.KeyRune:
			r := e.Rune()
			if v.cursor.mode == insert {
				v.buf.insert(v.cursor.loc, string(r))
				v.cursor.right()
				ret = 2
			} else if v.cursor.mode == command {
				switch r {
				case 'w':
					v.buf.save()
				}
			} else {
				switch r {
				case 'i':
					v.cursor.mode = insert
				case 'j':
					v.cursor.down()
					ret = 1
				case 'k':
					v.cursor.up()
					ret = 1
				case 'l':
					v.cursor.right()
					ret = 1
				case 'h':
					v.cursor.left()
					ret = 1
				case ':':
					v.cursor.mode = command
				}
			}
		}
	case *tcell.EventMouse:
		x, y := e.Position()
		y += v.topline
		// Position always seems to be off by one
		x--
		y--

		button := e.Buttons()

		switch button {
		case tcell.Button1:
			if y-v.topline > v.linesN-1 {
				y = v.linesN + v.topline - 1
			}
			if y > len(v.buf.lines) {
				y = len(v.buf.lines) - 1
			}
			if x > count(v.buf.lines[y]) {
				x = count(v.buf.lines[y])
			}

			x = v.cursor.getCharPos(y, x)
			d := v.cursor.distance(x, y)
			v.cursor.loc += d
			v.cursor.x = x
			v.cursor.y = y

			if v.mouseReleased {
				v.cursor.selectionStart = v.cursor.loc
			}
			v.cursor.selectionEnd = v.cursor.loc
			v.mouseReleased = false
			ret = 2
		case tcell.ButtonNone:
			v.mouseReleased = true
		case tcell.WheelUp:
			if v.topline > 0 {
				v.topline--
				return 2
			} else {
				return 0
			}
		case tcell.WheelDown:
			if v.topline < len(v.buf.lines)-v.linesN {
				v.topline++
				return 2
			} else {
				return 0
			}
		}
	}

	cy := v.cursor.y
	if cy < v.topline {
		v.topline = cy
		ret = 2
	}
	if cy > v.topline+v.linesN-1 {
		v.topline = cy - v.linesN + 1
		ret = 2
	}

	return ret
}

func (v *View) display() {
	var charNum int
	for l := range v.linesN {
		if l+v.topline >= len(v.buf.lines) {
			break
		}
		line := strings.Replace(v.buf.lines[l+v.topline], "\t", emptyString(tabSize), -1)
		for colN, ch := range line {
			st := tcell.StyleDefault
			if v.cursor.hasSelection() && charNum >= v.cursor.selectionStart && charNum <= v.cursor.selectionEnd {
				st = st.Reverse(true)
			}

			v.s.SetContent(colN, l, ch, nil, st)
			charNum++
		}
		charNum++
	}
}
