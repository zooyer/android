package log

import (
	"fmt"
	"log"
	"os"
	"time"
)

var debug = false

func output(level string, v ...any) {
	if debug {
		log.Println(append([]any{"[KEYD " + level + "]:"}, v...)...)
	}
}

func cyan(s string) string {
	return fmt.Sprintf("\033[1;36m%s\033[0m", s)
}

func green(s string) string {
	return fmt.Sprintf("\033[1;32m%s\033[0m", s)
}

func blue(s string) string {
	return fmt.Sprintf("\033[1;34m%s\033[0m", s)
}

func purple(s string) string {
	return fmt.Sprintf("\033[1;35m%s\033[0m", s)
}

func red(s string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", s)
}

func Debug(v ...any) {
	output(green("D"), v...)
}

func Info(v ...any) {
	output(blue("I"), v...)
}

func Warn(v ...any) {
	output(purple("W"), v...)
}

func Error(v ...any) {
	output(red("E"), v...)
}

func setDebug(d bool) {
	debug = d
}

func Init(debug bool) {
	setDebug(debug)
	log.SetOutput(os.Stdout)
	time.Local = time.FixedZone("CST", 8*3600)
}
