package forth

import (
	"testing"
)

func TestRunForth(t *testing.T) {
	var tests = []struct {
		prog string
		want string
	}{
		{"3 8 + .", "11 "},
		{": add + ; 4 8 add .", "12 "},
		{": count 0 do i . loop ; 10 count", "0 1 2 3 4 5 6 7 8 9 "},
		{"5 6 * . cr bl emit", "30 \n "},
		{`variable a
      variable b
      20 a ! 80 b !
      a @ b @ * .`, "1600 "},
		{`variable a
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
		got := RunForth(test.prog)
		if got != test.want {
			t.Errorf("RunForth(%q) = %q; wanted %q", test.prog, got, test.want)
		}
	}
}
