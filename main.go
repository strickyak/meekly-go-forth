// +build main

// echo 3 4 + . cr | go run main.go
package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/chzyer/readline"

	forth "github.com/strickyak/meekly-go-forth"
)

var FlagI = flag.Bool("i", false, "run interactive shell even if command line scripts are given")

func main() {
	flag.Parse()

	f := forth.NewForth(emit)

	for _, filename := range flag.Args() {
		bb, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("Cannot read file %q: %v", filename, err)
		}
		f.RunProgram(string(bb))
	}

	if *FlagI || flag.NArg() == 0 {

		home := os.Getenv("HOME")
		if home == "" {
			home = "."
		}
		rl, err := readline.NewEx(&readline.Config{
			Prompt:          " ok ",
			HistoryFile:     filepath.Join(home, ".livy-apl.history"),
			InterruptPrompt: "*SIGINT*",
			EOFPrompt:       "*EOF*",
			// AutoComplete:    completer,
			// HistorySearchFold:   true,
			// FuncFilterInputRune: filterInput,
		})
		if err != nil {
			panic(err)
		}
		defer rl.Close()

		for {
			os.Stderr.Write([]byte{'\n'})
			line, err := rl.Readline()
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				} else {
					continue
				}
			} else if err == io.EOF {
				break
			}
			// f.RunProgram(line)
			TryRunProgram(f, line)
		}
	}
}

func TryRunProgram(f *forth.Forth, line string) {
	defer func() {
		r := recover()
		if r != nil {
			log.Printf("*** ERROR *** %v", r)
		}
	}()
	f.RunProgram(line)
}

func emit(ch byte) {
	//log.Printf("emit byte: %d", ch)
	_, err := os.Stdout.Write([]byte{ch})
	if err != nil {
		log.Fatalf("cannot emit: %v", err)
	}
}
