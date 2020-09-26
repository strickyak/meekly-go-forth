package forth

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

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
		input     []string
		Mem       [100]int
		Here      int
		Emit      EmitFn
	}
	EmitFn func(ch byte)
)

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

func (f *Forth) nextWord() string {
	word := f.input[0]
	f.input = f.input[1:]
	return word
}

func (f *Forth) runWords(word []string) {
	for _, w := range word {
		f.runWord(w)
	}
}
func (f *Forth) runWord(word string) {
	// First see if it is an integer.
	i, err := strconv.ParseInt(word, 10, 64)
	if err == nil {
		f.D.Push(int(i))
		return
	}
	// Or else it should be a defined word.
	fn, ok := f.Words[word]
	if !ok {
		log.Panicf("forth: word not found: %q", word)
	}
	fn()
}
func (f *Forth) RunProgram(prog string) {
	f.input = strings.Fields(prog)
	for len(f.input) > 0 {
		f.runWord(f.nextWord())
	}
}

func (f *Forth) compileStepsUntil(enders ...string) (end string, steps []string) {
	for {
		w := f.nextWord()
		for _, end = range enders {
			if w == end {
				return
			}
		}
		// First look to see if it is an Immediate word.
		if imm, ok := f.Immediate[w]; ok {
			// If so, execute the Immediate function, and append the result.
			steps = append(steps, imm())
		} else {
			// Or append the normal word.
			steps = append(steps, w)
		}
	}
}

func (f *Forth) InitWords() {
	f.Immediate = Idict{
		"if": func() string {
			name := fmt.Sprintf("if_%d_", f.Here)
			f.Here++ // we needed a unique name.
			var then_steps, else_steps []string
			ender, then_steps := f.compileStepsUntil("else", "then")
			if ender == "else" {
				_, else_steps = f.compileStepsUntil("then")
			}
			fn := func() {
				if f.D.Pop() != 0 {
					f.runWords(then_steps)
				} else {
					f.runWords(else_steps)
				}
			}
			f.Words[name] = fn
			return name
		},
		"do": func() string {
			name := fmt.Sprintf("do_%d_", f.Here) // Use Here to create a unique name,
			f.Here++                              // then waste a slot.
			_, steps := f.compileStepsUntil("loop")
			fn := func() {
				i := f.D.Pop()
				limit := f.D.Pop()
				for ; i < limit; i++ {
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
				}
			}
			f.Words[name] = fn
			return name
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
		"+": binop(func(x, y int) int { return x + y }),
		"-": binop(func(x, y int) int { return x - y }),
		"*": binop(func(x, y int) int { return x * y }),
		"/": binop(func(x, y int) int { return x / y }),
		"%": binop(func(x, y int) int { return x % y }),
		"variable": func() {
			name := f.nextWord()
			i := f.Here
			f.Here++
			fn := func() {
				f.D.Push(i)
			}
			f.Words[name] = fn
		},
		":": func() {
			name := f.nextWord()
			_, steps := f.compileStepsUntil(";")
			fn := func() {
				for _, w := range steps {
					f.runWord(w)
				}
			}
			f.Words[name] = fn
		},
		"!": func() {
			i := f.D.Pop()
			x := f.D.Pop()
			f.Mem[i] = x
		},
		"@": func() { f.D.Push(f.Mem[f.D.Pop()]) },
		"dup": func() {
			x := f.D.Pop()
			f.D.Push(x)
			f.D.Push(x)
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
		"?": func() {
			log.Printf("----------------")
			log.Printf("D Stack: %v", f.D)
			log.Printf("R Stack: %v", f.R)
			for i, e := range f.Mem {
				if e != 0 {
					log.Printf("Mem[%d] = %d", i, e)
				}
			}
			log.Printf("================")
		},
	}
}
