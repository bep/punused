package secondpackage

import (
	"fmt"

	"github.com/bep/unused/internal/lib/testpackages/firstpackage"
)

func UseStuffInFirstPackage() {
	firstpackage.UsedFunction()
	fmt.Println(firstpackage.UsedVar, firstpackage.UsedConst)

	mt := firstpackage.MyType{
		UsedField: "UsedField",
	}
	mt.UsedMethod()
	fmt.Println(mt.UsedField)
}

func UseStuffInThisPackage() {
	UseStuffInFirstPackage()

	i1 := new(UsedInterfaceInterfaceImpl)
	_ = i1.UsedInterfaceMethodReturningInt()
	_ = i1.UsedInterfaceReturningInt()

	i2 := new(UsedInterfaceInterface2Impl)
	_ = i2.UsedInterface2ReturningInt()

	GetInterface2Implementation()
}

func GetInterfaceImplementation() *UsedInterfaceInterfaceImpl {
	UseStuffInThisPackage()
	return new(UsedInterfaceInterfaceImpl)
}

func GetInterface2Implementation() firstpackage.UsedInterface2 {
	return new(UsedInterfaceInterface2Impl)
}

type UsedInterfaceInterfaceImpl int

func (i UsedInterfaceInterfaceImpl) UsedInterfaceReturningInt() int {
	return 22
}

func (i UsedInterfaceInterfaceImpl) UsedInterfaceMethodReturningInt() int {
	GetInterfaceImplementation()
	return 32
}

type UsedInterfaceInterface2Impl int

func (i UsedInterfaceInterface2Impl) UsedInterface2ReturningInt() int {
	return 42
}
