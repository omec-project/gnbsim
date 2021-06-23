package fatal

import (
	"fmt"
	"os"
)

func Exit(code int) {
	os.Exit(code)
}
func Fatalf(format string, err error) {
	fmt.Println(fmt.Errorf(format, err))
	Exit(1)
}
