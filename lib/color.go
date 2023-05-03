package syncmediatrack

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	ColorGreen  = color.New(color.FgGreen).SprintFunc()
	ColorRed    = color.New(color.FgRed).SprintFunc()
	ColorYellow = color.New(color.FgYellow).SprintFunc()
)

func Red(s string) {
	fmt.Println(Colorize(s, color.FgRed))
}

func Colorize(s string, c color.Attribute) string {
	return color.New(c).SprintFunc()(s)
}
