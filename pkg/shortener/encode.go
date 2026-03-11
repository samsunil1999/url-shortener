package shortener

const base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Encode converts a uint64 (DB row ID) into a base62 short code
func Encode(num uint64) string {
	if num == 0 {
		return string(base62Chars[0])
	}
	result := []byte{}
	for num > 0 {
		result = append([]byte{base62Chars[num%62]}, result...)
		num /= 62
	}
	return string(result)
}

// Decode converts a base62 short code back to its uint64
func Decode(s string) uint64 {
	var num uint64
	for _, c := range s {
		num = num*62 + uint64(indexOf(byte(c)))
	}
	return num
}

func indexOf(c byte) int {
	for i := 0; i < len(base62Chars); i++ {
		if base62Chars[i] == c {
			return i
		}
	}
	return 0
}
