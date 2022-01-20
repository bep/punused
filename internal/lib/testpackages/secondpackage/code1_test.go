package secondpackage

import (
	"fmt"
	"testing"

	"github.com/bep/punused/internal/lib/testpackages/firstpackage"
)

func TestUseTestlib1(t *testing.T) {
	fmt.Println(firstpackage.OnlyUsedInTestConst)
}
