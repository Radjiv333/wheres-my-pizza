package utils

import (
	"strings"
)

func GetStringArray(str string) []string {
	var arr []string
	str = strings.ReplaceAll(str, " ", "")
	arr = strings.Split(str, ",")
	return arr
}
