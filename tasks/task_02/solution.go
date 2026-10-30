package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}
	shift = shift % n
	if shift < 0 {
		shift = n + shift
	}
	runes = append(runes[shift:], runes[:shift]...)
	return string(runes)
}