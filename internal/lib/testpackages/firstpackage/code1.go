package firstpackage

import "fmt"

var (
	UsedVar   = "UsedVar"
	UnusedVar = "UnusedVar"
)

const (
	UsedConst   = "UsedConst"
	UnusedConst = "UnusedConst"
)

func UsedFunction() {
	fmt.Println("UsedFunction")
}

func UnusedFunction() {
	fmt.Println("UnusedFunction")
}

type MyType struct {
	UsedField   string
	UnusedField string
}

func (MyType) UsedMethod() {
	fmt.Println("UsedMethod")
}

func (MyType) UnusedMethod() {
	fmt.Println("UnusedMethod")
}

type UnusedInterfaceWithUsedAndUnusedMethod interface {
	UsedInterfaceMethodReturningInt() int
	UnusedInterfaceMethodReturningInt() int
}

type UnusedInterface interface {
	UnusedInterfaceReturningInt() int
}

type UsedInterface interface {
	UsedInterfaceReturningInt() int
}

type UsedInterface2 interface {
	UsedInterface2ReturningInt() int
}
