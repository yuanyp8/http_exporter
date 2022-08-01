package http

import (
	"fmt"
	"testing"
)

func TestAdjustTarget(t *testing.T) {
	b := "accc.com"
	AdjustTarget(&b)
	fmt.Println(b)
}
