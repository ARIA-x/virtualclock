package main

import (
	"fmt"
	"sync"
)

type Person struct{ id, x, y int }

func (p *Person) run() {
	p.x += 1
	p.y += 1
}

func ready(fn func(), id int, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("ready")
	fmt.Println("run")
	fn()
	fmt.Println("done")
}

func main() {
	person := make([]Person, 3)
	var wg sync.WaitGroup

	for i := range person {
		person[i].id = i
		person[i].x = 0
		person[i].y = 0
	}

	duration := 10
	for duration > 0 {
		for i := range person {
			wg.Add(1)
			go ready(person[i].run, person[i].id, &wg)
		}
		wg.Wait()
		duration--
	}

	fmt.Print(person)
}
