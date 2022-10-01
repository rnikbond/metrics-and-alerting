package main

import (
	"fmt"
	"os"
)

func main() {

	fmt.Println("Five wine experts jokingly quizzed sample chablis")
	os.Exit(0) // want "you call Exit from main, but you do it without respect"
}
