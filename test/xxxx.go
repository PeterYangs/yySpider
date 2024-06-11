package main

import (
	"fmt"
	"path/filepath"
)

func main() {

	str := "xx/yy/asdaskjdas.xlsx"

	fmt.Println(filepath.Dir(str))

}
