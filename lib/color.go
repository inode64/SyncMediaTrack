package syncmediatrack

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	ColorBlue   = color.New(color.FgBlue).SprintFunc()
	ColorGreen  = color.New(color.FgGreen).SprintFunc()
	ColorRed    = color.New(color.FgRed).SprintFunc()
	ColorYellow = color.New(color.FgYellow).SprintFunc()
)

func Error(s string) {
	fmt.Println(Colorize(s, color.FgRed))
}

func Warning(s string) {
	fmt.Println(Colorize(s, color.FgRed))
}

func Notice(s string) {
	fmt.Println(Colorize(s, color.FgYellow))
}

func Info(s string) {
	fmt.Println(Colorize(s, color.FgGreen))
}

func Pass(s string) {
	fmt.Println(Colorize(s, color.FgBlue))
}

func Colorize(s string, c color.Attribute) string {
	return color.New(c).SprintFunc()(s)
}
