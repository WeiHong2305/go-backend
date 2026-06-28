func isSubsequence(s string, t string) bool {
	sIndex, tIndex := 0, 0
	for sIndex < len(s) && tIndex < len(t) {
		if t[tIndex] == s[sIndex] {
			tIndex++
			sIndex++
		} else {
			tIndex++
		}
	}

	if sIndex >= len(s) {
		return true
	} else {
		return false
	}
}