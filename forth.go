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

func (f *Forth) InitWords() {
	compileStepsUntil := func(end string) []string {
		var steps []string
		for {
			w := f.nextWord()
			if w == end {
				break
			}
			if imm, ok := f.immediate[w]; ok {
				steps = append(steps, imm())
			} else {
				steps = append(steps, w)
			}
		}
		return steps
	}

	f.immediate = idict{
		"do": func() string {
			// Use f.here for deterministic, unique name.
			name := fmt.Sprintf("do_%d", f.here)
			f.here++
			steps := compileStepsUntil("loop")
			fn := func() {
				i := f.d.pop()
				limit := f.d.pop()
				for ; i < limit; i++ {
					for _, w := range steps {
						f.r.push(i)
						f.runWord(w)
						f.r.pop()
					}
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
			steps := compileStepsUntil(";")
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
		"@": func() {
			i := f.d.pop()
			f.d.push(f.mem[i])
		},
		"dup": func() {
			x := f.d.pop()
			f.d.push(x)
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
