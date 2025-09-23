package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

const (
	NORMAL_MODE  = iota
	INSERT_MODE  = iota
	COMMAND_MODE = iota
)

// Structure for holding cursor position
type cursor struct {
	x            int
	y            int
	scrolloffset int
}

// Structure for holding editor state.
type editor struct {
	screen    tcell.Screen
	style     tcell.Style
	mode      int8
	lines     []string
	cursor    cursor
	cmdbuf    string
	statusmsg string
}

// Function for initializing cursor state based on file size
func initCursor(c *cursor, lines []string) {
	if c.y+c.scrolloffset >= len(lines) {
		c.y = len(lines) - 1 - c.scrolloffset
		if c.y < 0 {
			c.y = 0
		}
	}
}

// Function for initializing editor state. As parameters it needs
// an editor structure variable and an array of lines
func initEditor(e *editor, lines []string) {
	var err error
	e.screen, err = tcell.NewScreen()
	if err != nil {
		log.Fatalf("Failed to create screen: %v", err)
	}
	if err := e.screen.Init(); err != nil {
		log.Fatalf("Failed to initialize screen: %v", err)
	}
	e.style = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	e.mode = NORMAL_MODE
	e.lines = lines
	e.cmdbuf = ""
	e.statusmsg = ""

	initCursor(&e.cursor, e.lines)

	// Draw initial lines
	draw(e)
}

// Draw lines that are stored inside editor structure to screen
func draw(e *editor) {
	e.screen.Clear()
	width, height := e.screen.Size()

	// Draw lines
	for y := 0; y < height && (e.cursor.scrolloffset+y) < len(e.lines); y++ {
		line := e.lines[e.cursor.scrolloffset+y]
		for x, r := range line {
			if x >= width {
				break
			}
			e.screen.SetContent(x, y, r, nil, e.style)
		}
	}

	// Draw status line
	var status string
	if e.mode == COMMAND_MODE {
		status = ":" + e.cmdbuf
	} else if e.statusmsg != "" {
		status = e.statusmsg
	} else if e.mode == INSERT_MODE {
		status = "-- INSERT --"
	}

	for i := 0; i < width; i++ {
		ch := ' '
		if i < len(status) {
			ch = rune(status[i])
		}
		e.screen.SetContent(i, height-1, ch, nil, e.style)
	}

	// Show cursor position
	currline := ""
	if e.cursor.y+e.cursor.scrolloffset < len(e.lines) {
		currline = e.lines[e.cursor.y+e.cursor.scrolloffset]
	}
	if e.cursor.x > len(currline) {
		e.cursor.x = len(currline)
	}
	e.screen.ShowCursor(e.cursor.x, e.cursor.y)

	e.screen.Show()
}

// Write lines to a file
func wlines(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

func main() {
	// TODO: Handle what to do when no file is provided
	if len(os.Args) == 1 {
	}
	// TODO: Check if file exists, if not create it
	if len(os.Args) == 2 {
	}

	// Open file provided from stdin as first argument
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Read file line by line
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Init editor structure
	var e editor
	initEditor(&e, lines)
	defer e.screen.Fini() // NOTE: doesn't work when added in 'initEditor()'

	// This is the main application loop. Here we handle all keys
	for {
		ev := e.screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch e.mode {
			case NORMAL_MODE:
				switch ev.Key() {
				case tcell.KeyCtrlQ, tcell.KeyEscape:
					return
				default:
					switch ev.Rune() {
					case 'q':
						return
					case 'i':
						e.mode = INSERT_MODE
					case 'j': // scroll down
						// NOTE: if we are at the last line displayed on screen then
						// change scrolloffset, else just change the cursor position
						if e.cursor.scrolloffset+e.cursor.y+1 < len(e.lines) {
							_, h := e.screen.Size() // Screen height
							if e.cursor.y < h-1 {
								e.cursor.y++
							} else {
								e.cursor.scrolloffset++
							}
						}
					case 'k': // scroll up
						if e.cursor.y > 0 {
							e.cursor.y--
						} else if e.cursor.scrolloffset > 0 {
							e.cursor.scrolloffset--
						}
					case 'h': // move left
						if e.cursor.x > 0 {
							e.cursor.x--
						}
					case 'l': // move right
						e.cursor.x++
					case ':':
						e.mode = COMMAND_MODE
						e.cmdbuf = ""
						e.statusmsg = ""
					}
				}
			case INSERT_MODE:
				lineIdx := e.cursor.y + e.cursor.scrolloffset
				switch ev.Key() {
				case tcell.KeyESC:
					e.mode = NORMAL_MODE
					// Here we handle deleting in insert mode
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					if lineIdx < len(e.lines) && e.cursor.x > 0 {
						line := e.lines[lineIdx]
						e.lines[lineIdx] = line[:e.cursor.x-1] + line[e.cursor.x:]
						e.cursor.x--
					}
				case tcell.KeyEnter:
					if lineIdx < len(e.lines) {
						line := e.lines[lineIdx]
						// Split the line at cursorX
						before := line[:e.cursor.x]
						after := line[e.cursor.x:]
						// Insert new line with 'after' part
						e.lines = append(e.lines[:lineIdx+1], append([]string{after}, e.lines[lineIdx+1:]...)...)
						e.lines[lineIdx] = before
						e.cursor.y++
						e.cursor.x = 0

						// If cursor is at bottom of screen, scroll
						_, h := e.screen.Size()
						if e.cursor.y >= h-1 {
							e.cursor.scrolloffset++
							e.cursor.y = h - 2
						}
					}
					// Here we handle writing characters in insert mode
				case tcell.KeyRune:
					lineIdx := e.cursor.y + e.cursor.scrolloffset
					if lineIdx < len(e.lines) {
						line := e.lines[lineIdx]
						r := ev.Rune()
						if e.cursor.x > len(line) {
							e.cursor.x = len(line)
						}
						e.lines[lineIdx] = line[:e.cursor.x] + string(r) + line[e.cursor.x:]
						e.cursor.x++
					}
				}
			case COMMAND_MODE:
				switch ev.Key() {
				case tcell.KeyEsc:
					e.mode = NORMAL_MODE
					e.cmdbuf = ""
					e.statusmsg = ""
				case tcell.KeyEnter:
					cmd := strings.TrimSpace(e.cmdbuf)

					// NOTE: Here we handle commands in COMMAND_MODE
					switch cmd {
					case "w":
						err := wlines(os.Args[1], e.lines)
						if err != nil {
							e.statusmsg = fmt.Sprintf("Error writing: %v", err)
						} else {
							e.statusmsg = fmt.Sprintf("Wrote to %s", os.Args[1])
						}
					case "q":
						e.screen.Fini()
						os.Exit(0)
					case "wq":
						err := wlines(os.Args[1], e.lines)
						if err != nil {
							e.statusmsg = fmt.Sprintf("Error writing: %v", err)
							e.mode = NORMAL_MODE
						} else {
							e.screen.Fini()
							os.Exit(0)
						}
					default:
						e.statusmsg = "Unknown command: " + cmd
						e.mode = NORMAL_MODE
					}
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					if len(e.cmdbuf) > 0 {
						e.cmdbuf = e.cmdbuf[:len(e.cmdbuf)-1]
					}
				case tcell.KeyRune:
					e.cmdbuf += string(ev.Rune())
				}
			}
			draw(&e)
		case *tcell.EventResize:
			draw(&e)
		}
	}
}
