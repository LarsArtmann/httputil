package httputil

import "strings"

func join(ss []string) string {
	return strings.Join(ss, ", ")
}

const decimalBase = 10

func itoa(num int) string {
	if num == 0 {
		return "0"
	}

	neg := false
	if num < 0 {
		neg = true
		num = -num
	}

	var buf [20]byte

	i := len(buf)
	for num > 0 {
		i--
		buf[i] = byte('0') + byte(num%decimalBase)
		num /= decimalBase
	}

	if neg {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}
