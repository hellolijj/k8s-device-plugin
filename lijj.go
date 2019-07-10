package main

import (
	"fmt"
)

func main() {
	test := make([]int, 12)
	
	test[1] = 23
	
	fmt.Println(test)
	
	f := make([][]int, 12)
	for i := 0; i < 12; i++ {
		f[i] = make([]int, 12)
	}
	fmt.Println(f)
}
