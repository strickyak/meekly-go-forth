package forth

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
)

var V = flag.Bool("v", false, "verbosity")
var E = flag.Bool("e", false, "show error stack")

type (
	Dict  map[string]func()
	Idict map[string]func() string
	Stack struct {
		v []int
	}
	Forth struct {
		Words     Dict
		Immediate Idict
		D         Stack
		R         Stack
		Input     string
		Mem       [100000]int
		Here      int
		Emit      EmitFn
		RecentDef string
	}
	EmitFn                func(ch byte)
	CompileThisWordSignal string
)

func Panicf(format string, args ...interface{}) {
	log.Printf("==========================================")
	debug.PrintStack()
	log.Printf("==========================================")
	log.Panicf(format, args...)
}

func RunForth(prog string) string {
	var output []byte
	f := NewForth(func(ch byte) {
		output = append(output, ch)
	})
	f.RunProgram(prog)
	return string(output)
}

func NewForth(emit EmitFn) *Forth {
	f := &Forth{
		Emit: emit,
	}
	f.InitWords()
	return f
}

func (o *Stack) Push(x int) {
	o.v = append(o.v, x)
}

func (o *Stack) Pop() int {
	z := o.v[len(o.v)-1]
	o.v = o.v[:len(o.v)-1]
	return z
}

func (o *Stack) String() string {
	return fmt.Sprintf("%#v", o.v)
}

func (f *Forth) NextChar() (rune, bool) {
	if len(f.Input) == 0 {
		return 0, false
	}
	// TODO strings.NewReader. but for now, bytes.
	var z rune = rune(f.Input[0])
	f.Input = f.Input[1:]
	return z, true
}
func (f *Forth) NextWord() (string, bool) {
	var r rune
	var ok bool
	// Consume white space.  Leave first non-white in r.
	for {
		r, ok = f.NextChar()
		if !ok {
			return "", false
		}
		if r > 32 {
			break
		}
		// consume all control chars as white space
	}

	var buf strings.Builder
	buf.WriteRune(r)
	for {
		r, ok = f.NextChar()
		if !ok {
			return strings.ToLower(buf.String()), true
		}
		if r <= 32 {
			break
		}
		buf.WriteRune(r)
	}
	return strings.ToLower(buf.String()), true
}

func (f *Forth) runWords(word []string) {
	for _, w := range word {
		f.runWord(w)
	}
}
func (f *Forth) runWord(word string) {
	word = strings.ToLower(word)
	if *V {
		log.Printf("RW: %q", word)
	}
	// First see if it is an integer.
	i, err := strconv.ParseInt(word, 10, 64)
	if err == nil {
		f.D.Push(int(i))
		return
	}
	// Or else it could be a normal word.
	fn, ok := f.Words[word]
	if ok {
		if fn == nil {
			Panicf("runWord: word %q is nil", word)
		}
		fn()
		return // normal word normal return.
	}
	// Or else it could be an immediate word.
	imm, ok2 := f.Immediate[word]
	if !ok2 {
		Panicf("forth: word not found: %q", word)
	}
	word = imm() // Change the word to generated word.

	fn, ok = f.Words[word]
	if ok {
		fn()
		return // generated word normal return.
	}
	Panicf("forth: genrated word not found: %q", word)
}
func (f *Forth) RunProgram(prog string) {
	f.Input = prog
	for {
		w, ok := f.NextWord()
		if !ok {
			break
		}
		if *V {
			log.Printf("RP: %q", w)
		}
		f.runWord(w)
	}
}

func (f *Forth) compileStepsUntil(enders ...string) (end string, steps []string) {
	for {
		w, ok := f.NextWord()
		if !ok {
			Panicf("compileStepsUntil got EOF, wanted one of %v", enders)
		}
		for _, end = range enders {
			if w == end {
				return
			}
		}

		func() {
			defer func() {
				r := recover()
				switch t := r.(type) {
				case CompileThisWordSignal:
					steps = append(steps, string(t))
				case nil:
					return
				default:
					panic(r)
				}
			}()
			// First look to see if it is an Immediate word.
			if imm, ok := f.Immediate[w]; ok {
				// If so, execute the Immediate function, and append the result.
				steps = append(steps, imm())
			} else {
				// Or append the normal word.
				steps = append(steps, w)
			}
		}()
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func DumpInfo(f *Forth) {
	log.Printf("<<<<<<<<<<<<<<<<")
	log.Printf("D Stack: %v", f.D)
	log.Printf("R Stack: %v", f.R)
	for i, e := range f.Mem {
		if e != 0 {
			fmt.Fprintf(os.Stderr, "[%d]=%d ", i, e)
		}
	}
	log.Printf(">>>>>>>>>>>>>>>")
}
func (f *Forth) InitWords() {
	f.Immediate = Idict{
		"(": func() string {
			for {
				r, ok := f.NextChar()
				if !ok || r == ')' {
					break
				}
			}
			return "nop"
		},
		`\`: func() string {
			for {
				r, ok := f.NextChar()
				if !ok || r == '\n' || r == '\r' {
					break
				}
			}
			return "nop"
		},
		`."`: func() string {
			var buf strings.Builder
			for {
				r, ok := f.NextChar()
				if !ok {
					Panicf("`.\"` read EOF, wanted `\"`")
				}
				if r == '"' {
					break
				}
				buf.WriteRune(r)
			}
			str := buf.String()
			name := fmt.Sprintf("~emit~str~%d~", f.Here)
			f.Words[name] = func() {
				for _, r := range str {
					f.Emit(byte(r))
				}
			}
			return name
		},
		`s"`: func() string {
			var buf strings.Builder
			start := f.Here
			for {
				r, ok := f.NextChar()
				if !ok {
					Panicf("`s\"` read EOF, wanted `\"`")
				}
				if r == '"' {
					break
				}
				b := byte(r)
				buf.WriteByte(b)
				f.Mem[f.Here] = int(b)
				f.Here++
			}
			f.Mem[f.Here] = 0
			f.Here++
			name := fmt.Sprintf("~str~%d~", f.Here)
			f.Words[name] = func() {
				f.D.Push(start)
				f.D.Push(f.Here - start - 1)
			}
			return name
		},
		"begin": func() string {
			name := fmt.Sprintf("~if~%d~", f.Here)
			f.Here++ // we needed a unique name.
			var first_steps, later_steps []string
			var has_while bool
			ender, first_steps := f.compileStepsUntil("while", "repeat")
			if ender == "while" {
				_, later_steps = f.compileStepsUntil("repeat")
				has_while = true
			}
			f.Words[name] = func() {
				for {
					f.runWords(first_steps)
					if has_while {
						cond := f.D.Pop()
						if cond == 0 {
							return
						}
						f.runWords(later_steps)
					}
				}
			}
			return name
		},
		"if": func() string {
			name := fmt.Sprintf("~if~%d~", f.Here)
			f.Here++ // we needed a unique name.
			var then_steps, else_steps []string
			ender, then_steps := f.compileStepsUntil("else", "then")
			if ender == "else" {
				_, else_steps = f.compileStepsUntil("then")
			}
			f.Words[name] = func() {
				if f.D.Pop() != 0 {
					f.runWords(then_steps)
				} else {
					f.runWords(else_steps)
				}
			}
			return name
		},
		"do": func() string {
			name := fmt.Sprintf("~do~%d~", f.Here) // Use Here to create a unique name,
			f.Here++                               // then waste a slot.
			ender, steps := f.compileStepsUntil("loop", "+loop")
			plus_loop := ender == "+loop"
			fn := func() {
				i := f.D.Pop()
				limit := f.D.Pop()
				for i < limit {
					// We don't have a Forth way to access `limit`
					// or change `i` (yet) but let's put them
					// both on the R-Stack anyway.
					f.R.Push(limit)
					f.R.Push(i)
					for _, w := range steps {
						f.runWord(w)
					}
					i = f.R.Pop()
					limit = f.R.Pop()
					if plus_loop {
						i += f.D.Pop()
					} else {
						i++
					}
				}
			}
			f.Words[name] = fn
			return name
		},
		"postpone": func() string {
			name, ok := f.NextWord()
			if !ok {
				Panicf("`variable` got EOF, wanted a word")
			}
			fn, ok := f.Immediate[name]
			if ok {
				fn()
				return "nop"
			}
			_, ok = f.Words[name]
			if ok {
				panic(CompileThisWordSignal(name))
			}
			Panicf("`postpone` cannot find word %q", name)
			return ""
		},
	}
	binop := func(op func(int, int) int) func() {
		return func() {
			y := f.D.Pop()
			x := f.D.Pop()
			f.D.Push(op(x, y))
		}
	}
	f.Words = Dict{
		"and":   binop(func(x, y int) int { return x & y }),
		"or":    binop(func(x, y int) int { return x | y }),
		"xor":   binop(func(x, y int) int { return x ^ y }),
		"+":     binop(func(x, y int) int { return x + y }),
		"-":     binop(func(x, y int) int { return x - y }),
		"*":     binop(func(x, y int) int { return x * y }),
		"/":     binop(func(x, y int) int { return x / y }),
		"%":     binop(func(x, y int) int { return x % y }),
		"=":     binop(func(x, y int) int { return boolToInt(x == y) }),
		"==":    binop(func(x, y int) int { return boolToInt(x == y) }),
		"!=":    binop(func(x, y int) int { return boolToInt(x != y) }),
		"/=":    binop(func(x, y int) int { return boolToInt(x != y) }),
		"<>":    binop(func(x, y int) int { return boolToInt(x != y) }),
		"<":     binop(func(x, y int) int { return boolToInt(x < y) }),
		"<=":    binop(func(x, y int) int { return boolToInt(x <= y) }),
		">=":    binop(func(x, y int) int { return boolToInt(x >= y) }),
		">":     binop(func(x, y int) int { return boolToInt(x > y) }),
		"nop":   func() {},
		"cells": func() {},
		"immediate": func() {
			name := f.RecentDef
			fn, ok := f.Words[name]
			if !ok {
				Panicf("`immediate` cannot find word %q", name)
			}
			delete(f.Words, name)
			f.Immediate[name] = func() string {
				fn()
				return "nop"
			}
		},
		"constant": func() {
			value := f.D.Pop()
			name, ok := f.NextWord()
			if !ok {
				Panicf("`constant` got EOF, wanted a word")
			}
			f.Words[name] = func() {
				f.D.Push(value)
			}
			f.RecentDef = name
		},
		"variable": func() {
			name, ok := f.NextWord()
			if !ok {
				Panicf("`variable` got EOF, wanted a word")
			}
			i := f.Here
			f.Here++
			f.Words[name] = func() {
				f.D.Push(i)
			}
			f.RecentDef = name
		},
		"create": func() {
			name, ok := f.NextWord()
			if !ok {
				Panicf("`variable` got EOF, wanted a word")
			}
			i := f.Here
			f.Words[name] = func() {
				f.D.Push(i)
			}
			f.RecentDef = name
		},
		"allot": func() { f.Here += f.D.Pop() },
		":": func() {
			name, ok := f.NextWord()
			if !ok {
				Panicf("`:` got EOF, wanted a word")
			}
			_, steps := f.compileStepsUntil(";")
			fn := func() {
				for _, w := range steps {
					f.runWord(w)
				}
			}
			f.Words[name] = fn
			f.RecentDef = name
		},
		"!": func() {
			i := f.D.Pop()
			x := f.D.Pop()
			f.Mem[i] = x
		},
		"@":  func() { f.D.Push(f.Mem[f.D.Pop()]) },
		"c@": func() { f.D.Push(f.Mem[f.D.Pop()]) },
		"not": func() {
			f.D.Push(boolToInt(f.D.Pop() == 0))
		},
		"0=": func() {
			f.D.Push(boolToInt(f.D.Pop() == 0))
		},
		"1+": func() {
			f.D.Push(f.D.Pop() + 1)
		},
		"1-": func() {
			f.D.Push(f.D.Pop() - 1)
		},
		"negate": func() {
			f.D.Push(0 - f.D.Pop())
		},
		"drop": func() {
			f.D.Pop()
		},
		"dup": func() {
			x := f.D.Pop()
			f.D.Push(x)
			f.D.Push(x)
		},
		"2dup": func() {
			y := f.D.Pop()
			x := f.D.Pop()
			f.D.Push(x)
			f.D.Push(y)
			f.D.Push(x)
			f.D.Push(y)
		},
		"swap": func() {
			y := f.D.Pop()
			x := f.D.Pop()
			f.D.Push(y)
			f.D.Push(x)
		},
		".": func() {
			x := f.D.Pop()
			s := fmt.Sprintf("%d ", x)
			for _, ch := range s {
				f.Emit(byte(ch))
			}
		},
		"j": func() {
			i := f.R.Pop()
			x := f.R.Pop()
			f.R.Push(x)
			f.R.Push(i)
			f.D.Push(x)
		},
		"i": func() {
			x := f.R.Pop()
			f.R.Push(x)
			f.D.Push(x)
		},
		">r": func() { f.R.Push(f.D.Pop()) },
		"r>": func() { f.D.Push(f.R.Pop()) },
		"emit": func() {
			x := f.D.Pop()
			f.Emit(byte(x))
		},
		"bl": func() { f.D.Push(' ') },
		"cr": func() { f.Emit('\n') },
		".s": func() {
			n := f.D.Pop()
			ptr := f.D.Pop()
			for i := 0; i < n; i++ {
				f.Emit(byte(f.Mem[ptr+i]))
			}
		},
		"???": func() {
			DumpInfo(f)
		},
		"words": func() {
			var words []string
			for k := range f.Words {
				words = append(words, k)
			}
			sort.Strings(words)
			for _, w := range words {
				fmt.Printf("%s ", w)
			}
			fmt.Printf("\n")
			words = nil
			for k := range f.Immediate {
				words = append(words, k)
			}
			sort.Strings(words)
			for _, w := range words {
				fmt.Printf("%s ", w)
			}
			fmt.Printf("\n")
		},
	}
	f.Immediate["?do"] = f.Immediate["do"]
}
