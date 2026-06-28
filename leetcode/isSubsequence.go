package leetcode

func isSubsequence(s string, t string) bool {
	sIndex, tIndex := 0, 0
	for sIndex < len(s) && tIndex < len(t) {
		if t[tIndex] == s[sIndex] {
			sIndex++
		}
		tIndex++
	}

	return sIndex >= len(s)
}

// Time complexity: O(n), n = len(t)
// Space complexity: O(1)

// Follow-up question: haven't solved yet
