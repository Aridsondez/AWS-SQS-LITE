
package main

import (
	"fmt"
)

func main() {

	fmt.Println("Hello, World!")
	var a int 
	a = 5
	fmt.Println(a)

	//different dataytpes in go 

	var keyword string = "var name then type then value or you don't have to define it"

	fmt.Println(keyword)

	var b int16 = 10 //There are different types of integers in go int8, int16, int32, int64 int will default to int32 or int64 depending on the system
	fmt.Println(b)
	var check float32 = 10.99 //There are different types of floats in go float32, float64 float will default to float64
	fmt.Println(check)
	var something = 20.99
	something2 := 30.99 //If you don't define the type it will default to float64
	fmt.Println(something)

	var array [3]int

	

}