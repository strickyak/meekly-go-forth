// +build main

// echo 3 4 + . cr | go run main.go
package main

import (
	forth "github.com/strickyak/meekly-go-forth"
	"io/ioutil"
	"os"
)

func main() {
	bb, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	f := forth.NewForth(putchar)
	f.RunProgram(string(bb))
}

func putchar(ch byte) {
	os.Stdout.Write([]byte{ch})
}
