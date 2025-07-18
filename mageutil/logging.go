package mageutil

import (
	"fmt"
	"os"
	"time"
)

const (
	ColorBlue   = "\033[0;34m"
	ColorGreen  = "\033[0;32m"
	ColorRed    = "\033[0;31m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
)

// Generic print function
func printWithColor(color, message string, withTime bool) {
	if withTime {
		currentTime := time.Now().Format("[2006-01-02 15:04:05 MST]")
		fmt.Printf("%s %s%s%s\n", currentTime, color, message, ColorReset)
	} else {
		fmt.Printf("%s%s%s\n", color, message, ColorReset)
	}
}

// Colored print with timestamp
func PrintBlue(message string) {
	printWithColor(ColorBlue, message, true)
}

func PrintGreen(message string) {
	printWithColor(ColorGreen, message, true)
}

func PrintRed(message string) {
	printWithColor(ColorRed, message, true)
}

func PrintYellow(message string) {
	printWithColor(ColorYellow, message, true)
}

// Two-line format (timestamp on separate line)
func PrintBlueTwoLine(message string) {
	fmt.Println(time.Now().Format("[2006-01-02 15:04:05 MST]"))
	fmt.Printf("%s%s%s\n", ColorBlue, message, ColorReset)
}

func PrintGreenTwoLine(message string) {
	fmt.Println(time.Now().Format("[2006-01-02 15:04:05 MST]"))
	fmt.Printf("%s%s%s\n", ColorGreen, message, ColorReset)
}

// Version without timestamp
func PrintRedNoTimeStamp(message string) {
	printWithColor(ColorRed, message, false)
}

func PrintGreenNoTimeStamp(message string) {
	printWithColor(ColorGreen, message, false)
}

// Output to specific streams
func PrintRedToStdErr(a ...any) (n int, err error) {
	return fmt.Fprint(os.Stderr, ColorRed, fmt.Sprint(a...), ColorReset)
}

func PrintGreenToStdOut(a ...any) (n int, err error) {
	return fmt.Fprint(os.Stdout, ColorGreen, fmt.Sprint(a...), ColorReset)
}
