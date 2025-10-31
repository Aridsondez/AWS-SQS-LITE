package main

import (
	"fmt"
	"time"
	"math"
)

type testStruct struct {
	value int
}
// i defined it as a var and not a actual type 
func main(){
	timeNow := time.Now()
	test := [4]int{1,2,3,4}

	for i := 0 ; i <len(test); i++{
		test[i] = int(math.Pow(float64(test[i]),2))
	}

	fmt.Println(test)
	time.Sleep(2*time.Second)
	fmt.Println(time.Since(timeNow))
	go func() {
		fmt.Println("Hello from goroutine")
	}()
	time.Sleep(2 * time.Second)

}