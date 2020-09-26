package forth

import (
	"log"
	"testing"
)

func TestRunForth(t *testing.T) {
	var tests = []struct {
		prog string
		want string
	}{
		{"3 8 + .", "11 "},
		{"3 8 swap - .", "5 "},
		{"3 8 - .", "-5 "},
		{": add + ; 4 8 add .", "12 "},
		{": count 0 do i . loop ; 10 count", "0 1 2 3 4 5 6 7 8 9 "},
		{"5 6 * . cr bl emit", "30 \n "},
		{": tmp 1 if 100 else 33 then . ; tmp", "100 "},
		{": tmp 0 if 100 else 33 then . ; tmp", "33 "},
		{": tmp 88 1 if 100 then . ; tmp", "100 "},
		{": tmp 88 0 if 100 then . ; tmp", "88 "},
		{`
                     variable a
                     variable b
                     20 a ! 80 b !
                     a @ b @ * .`, "1600 "},
		{`
                     variable a
                     variable b
                     : fib
                         1 b !  0 a !
                         0 do
                           a @ b @ +
                           dup .
                             b @ a !
                           b !
                         loop ;
                     10 fib`, "1 2 3 5 8 13 21 34 55 89 "},
	}
	for _, test := range tests {
		log.Printf("RunForth(%q)...", test.prog)
		got := RunForth(test.prog)
		if got != test.want {
			t.Errorf("RunForth(%q) = %q; wanted %q", test.prog, got, test.want)
		}
	}
}
