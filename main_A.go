package main

import (
	"fmt"
	"time"
)

func sayA(s string) {
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Println(s)
	}
}

func mainA() {
	go sayA("world")
	sayA("hello")
}
