package leetcode

func maxVowels(s string, k int) int {
	// Idiomatic Go for character classification. O(1) lookup, no branching, no string comparisons
	var isVowel [256]bool
	isVowel['a'], isVowel['e'], isVowel['i'], isVowel['o'], isVowel['u'] = true, true, true, true, true

	var count int

	for i := range k {
		if isVowel[s[i]] {
			count++
		}
	}

	maxCount := count

	// Micro-optimization that adds branching complexity
	if maxCount == k {
		return k
	}

	for i := k; i < len(s); i++ {
		if isVowel[s[i]] {
			count++
		}
		if isVowel[s[i-k]] {
			count--
		}

		if count > maxCount {
			maxCount = count
			if maxCount == k {
				return k
			}
		}
	}

	return maxCount
}
