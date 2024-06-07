package main

import (
	"bufio"
	"flag"
	"os"
	"os/signal"
	"syscall"

	//"strings"
	"sync"
	"time"
	"unicode"

	"github.com/hhhhhhhhhn/hexes"
	"github.com/hhhhhhhhhn/hexes/input"
	"github.com/mattn/go-runewidth"
)

var text []rune

var chars [][]rune
var times [][]time.Time

var cursorRow int
var cursorCol int

var bg = hexes.TrueColorBg(0, 0, 0)

var writer *bufio.Writer

var colors = []hexes.Attribute {
	hexes.Join(bg, hexes.TrueColor(255, 255, 255)),
	hexes.Join(bg, hexes.TrueColor(200, 200, 200)),
	hexes.Join(bg, hexes.TrueColor(150, 150, 150)),
	hexes.Join(bg, hexes.TrueColor(80, 80, 80)),
	hexes.Join(bg, hexes.TrueColor(20, 20, 20)),
	hexes.Join(bg, hexes.TrueColor(0, 0, 0)),
}

var timeLimit time.Duration

func timeToAttribute(t, now time.Time) hexes.Attribute {
	delta := now.Sub(t)

	if delta < timeLimit/16 {
		return colors[0]
	} else if delta < timeLimit/8 {
		return colors[1]
	} else if delta < timeLimit/4 {
		return colors[2]
	} else if delta < timeLimit/2 {
		return colors[3]
	} else if delta < timeLimit {
		return colors[4]
	} else {
		return colors[5]
	}
}


var renderMutex sync.Mutex
func render(renderer *hexes.Renderer) {
	renderMutex.Lock()
	defer renderMutex.Unlock()
	renderer.SetAttribute(hexes.NORMAL)
	now := time.Now()
	for r := 0; r < len(chars); r++ {
		for c := 0; c < len(chars[0]); c++ {
			renderer.MoveCursor(r, c)
			renderer.SetAttribute(timeToAttribute(times[r][c], now))
			if r == cursorRow && c == cursorCol {
				renderer.SetAttribute(hexes.Join(hexes.NORMAL, hexes.BG_MAGENTA))
				renderer.Set(r, c, ' ')
			} else if now.Sub(times[r][c]) > timeLimit && chars[r][c] > 1 {
				renderer.Set(r, c, ' ')
			} else if chars[r][c] > 1 {
				renderer.Set(r, c, chars[r][c])
			} else if chars[r][c] == 0{
				renderer.Set(r, c, ' ')
			}
		}
	}
	writer.Flush()
}

func writeCharacter(chr rune) {
	text = append(text, chr)
	width := runewidth.RuneWidth(chr)

	chars[cursorRow][cursorCol] = chr
	if width == 2 && cursorCol < len(chars[0])-1 {
		chars[cursorRow][cursorCol+1] = 1
	}
	times[cursorRow][cursorCol] = time.Now()
	cursorCol += width

	if cursorCol >= len(chars[0]) { // Column Amount
		cursorCol = 0
		cursorRow++
		if cursorRow >= len(chars) { // Row Amount
			cursorRow = 0
		}
	}
}

func moveCursorBack() {
	if cursorCol > 0 {
		cursorCol--
	} else {
		cursorCol = len(chars[0]) - 1
		cursorRow--
		if cursorRow < 0 {
			cursorRow = len(chars)-1
		}
	}
}

func tryRemoveCharacter() {
	if len(text) == 0 {
		return
	}
	originalCursorCol := cursorCol
	originalCursorRow := cursorRow

	moveCursorBack()
	if chars[cursorRow][cursorCol] == 1 { // Handles wide characters
		moveCursorBack()
	}

	if time.Now().Sub(times[cursorRow][cursorCol]) > timeLimit {
		cursorCol = originalCursorCol
		cursorRow = originalCursorRow
		return
	}
	chars[cursorRow][cursorCol] = 0

	text = text[:len(text)-1]
}

func handleInput(listener *input.Listener, renderer *hexes.Renderer) {
	for {
		event := listener.GetEvent()
		switch(event.EventType) {
		case input.KeyPressed:
			if event.Chr == '\n' {
				text = append(text, '\n')
				cursorCol = 0
				cursorRow++
				if cursorRow >= len(chars) {
					cursorRow = 0
				}
			} else if unicode.IsGraphic(event.Chr) {
				writeCharacter(event.Chr)
			} else if event.Chr == input.BACKSPACE {
				tryRemoveCharacter()
			}
		}
		render(renderer)
	}
}

func refresh(r *hexes.Renderer) {
	for {
		render(r)
		time.Sleep(time.Second/10)
	}
}

func main() {
	timeLimitSecs := flag.Float64("timelimit", 2, "Time limit for visual time")
	flag.Parse()
	timeLimit = time.Millisecond * time.Duration(*timeLimitSecs*1000)

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer tty.Close()

	writer = bufio.NewWriter(tty)
	r := hexes.New(tty, writer)
	r.SetDefaultAttribute(hexes.NORMAL)
	r.Start()

	listener := input.New(os.Stdin)

	chars = make([][]rune, r.Rows)
	for i := range chars {
		chars[i] = make([]rune, r.Cols)
		for j := range chars[i] {
			chars[i][j] = 0
		}
	}

	now := time.Unix(0, 0)
	times = make([][]time.Time, r.Rows)
	for i := range chars {
		times[i] = make([]time.Time, r.Cols)
		for j := range times[i] {
			times[i][j] = now
		}
	}

	go handleInput(listener, r)
	go refresh(r)

    channel := make(chan os.Signal, 1)
    signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)
    <-channel

	r.End()
	writer.Flush()

	os.Stdout.Write([]byte(string(text)))
}
