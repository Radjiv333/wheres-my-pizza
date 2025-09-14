package utils

import (
	"fmt"
	"strings"
)

func GetStringArray(str string) []string {
	var arr []string
	str = strings.ReplaceAll(str, " ", "")
	fmt.Println(str)
	arr = strings.Split(str, ",")
	fmt.Println(arr)
	return arr
}
