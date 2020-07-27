package forth

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type (
	dict  map[string]func()
	idict map[string]func() string
	stack struct {
		v []int
	}
	Forth struct {
		words     dict
		immediate idict
		d         stack
		r         stack
		input     []string
		mem       [100]int
		here      int
		emit      emitFn
	}
	emitFn func(ch byte)
)

func RunForth(prog string) string {
	var output []byte
	f := NewForth(func(ch byte) {
		output = append(output, ch)
	})
	f.RunProgram(prog)
	return string(output)
}

func NewForth(emit emitFn) *Forth {
	f := &Forth{
		emit: emit,
	}
	f.InitWords()
	return f
}

func (o *stack) push(x int) {
	o.v = append(o.v, x)
}

func (o *stack) pop() int {
	z := o.v[len(o.v)-1]
	o.v = o.v[:len(o.v)-1]
	return z
}

func (o *stack) String() string {
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
		f.d.push(int(i))
		return
	}
	// Or else it should be a defined word.
	fn, ok := f.words[word]
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
		if imm, ok := f.immediate[w]; ok {
			steps = append(steps, imm())
		} else {
			steps = append(steps, w)
		}
	}
}

func (f *Forth) InitWords() {
	f.immediate = idict{
		"if": func() string {
			name := fmt.Sprintf("if_%d_", f.here)
			f.here++ // we needed a unique name.
			var then_steps, else_steps []string
			ender, then_steps := f.compileStepsUntil("else", "then")
			if ender == "else" {
				_, else_steps = f.compileStepsUntil("then")
			}
			fn := func() {
				if f.d.pop() != 0 {
					f.runWords(then_steps)
				} else {
					f.runWords(else_steps)
				}
			}
			f.words[name] = fn
			return name
		},
		"do": func() string {
			name := fmt.Sprintf("do_%d_", f.here)
			f.here++ // we needed a unique name.
			_, steps := f.compileStepsUntil("loop")
			fn := func() {
				i := f.d.pop()
				limit := f.d.pop()
				for ; i < limit; i++ {
					// We don't have a Forth way to access `limit`
					// or change `i` (yet) but let's put them
					// both on the r-stack anyway.
					f.r.push(limit)
					f.r.push(i)
					for _, w := range steps {
						f.runWord(w)
					}
					i = f.r.pop()
					limit = f.r.pop()
				}
			}
			f.words[name] = fn
			return name
		},
	}
	binop := func(op func(int, int) int) func() {
		return func() {
			y := f.d.pop()
			x := f.d.pop()
			f.d.push(op(x, y))
		}
	}
	f.words = dict{
		"+": binop(func(x, y int) int { return x + y }),
		"-": binop(func(x, y int) int { return x - y }),
		"*": binop(func(x, y int) int { return x * y }),
		"/": binop(func(x, y int) int { return x / y }),
		"%": binop(func(x, y int) int { return x % y }),
		"variable": func() {
			name := f.nextWord()
			i := f.here
			f.here++
			fn := func() {
				f.d.push(i)
			}
			f.words[name] = fn
		},
		":": func() {
			name := f.nextWord()
			_, steps := f.compileStepsUntil(";")
			fn := func() {
				for _, w := range steps {
					f.runWord(w)
				}
			}
			f.words[name] = fn
		},
		"!": func() {
			i := f.d.pop()
			x := f.d.pop()
			f.mem[i] = x
		},
		"@": func() { f.d.push(f.mem[f.d.pop()]) },
		"dup": func() {
			x := f.d.pop()
			f.d.push(x)
			f.d.push(x)
		},
		"swap": func() {
			y := f.d.pop()
			x := f.d.pop()
			f.d.push(y)
			f.d.push(x)
		},
		".": func() {
			x := f.d.pop()
			s := fmt.Sprintf("%d ", x)
			for _, ch := range s {
				f.emit(byte(ch))
			}
		},
		"i": func() {
			x := f.r.pop()
			f.r.push(x)
			f.d.push(x)
		},
		">r": func() { f.r.push(f.d.pop()) },
		"r>": func() { f.d.push(f.r.pop()) },
		"emit": func() {
			x := f.d.pop()
			f.emit(byte(x))
		},
		"bl": func() { f.d.push(' ') },
		"cr": func() { f.emit('\n') },
		"?": func() {
			log.Printf("----------------")
			log.Printf("d stack: %v", f.d)
			log.Printf("r stack: %v", f.r)
			for i, e := range f.mem {
				if e != 0 {
					log.Printf("mem[%d] = %d", i, e)
				}
			}
			log.Printf("================")
		},
	}
}
