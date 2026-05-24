package httputil

import "strings"

func join(ss []string) string {
	return strings.Join(ss, ", ")
}

const (
	decimalBase = 10
	intBufSize  = 20
)

func itoa(num int) string {
	if num == 0 {
		return "0"
	}

	neg := num < 0

	var buf [intBufSize]byte

	i := len(buf)
	for num != 0 {
		i--

		d := num % decimalBase
		if d < 0 {
			d = -d
		}

		buf[i] = "0123456789"[d]
		num /= decimalBase
	}

	if neg {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}
