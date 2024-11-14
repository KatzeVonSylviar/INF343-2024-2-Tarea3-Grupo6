package main

import (
	"fmt"
	"time"
)

func sayB(s string) {
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Println(s)
	}
}

func mainB() {
	go sayB("world")
	sayB("hello")
}
