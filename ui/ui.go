package ui

import (
	"fmt"

	"github.com/fatih/color"
)

func SayOK() {
	c := color.New(color.FgGreen).Add(color.Bold)
	c.Println(OK)
}

func SayFailed() {
	c := color.New(color.FgRed).Add(color.Bold)
	c.Println(FAILED)
}

func SayMessage(message string, args ...interface{}) {
	fmt.Printf(message+"\n", args...)
}

func SayWarningMessage(message string, args ...interface{}) {
	c := color.New(color.FgYellow).Add(color.Bold)
	c.Printf(message+"\n", args...)
}
