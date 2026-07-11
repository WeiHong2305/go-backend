package leetcode

func maxVowels(s string, k int) int {
	var isVowel [256]bool
	isVowel['a'], isVowel['e'], isVowel['i'], isVowel['o'], isVowel['u'] = true, true, true, true, true

	var currentVowelCount int

	for i := 0; i < k; i++ {
		if isVowel[s[i]] {
			currentVowelCount++
		}
	}

	maxVowelCount := currentVowelCount
	if maxVowelCount == k {
		return k
	}

	for i := k; i < len(s); i++ {
		if isVowel[s[i]] {
			currentVowelCount++
		}
		if isVowel[s[i-k]] {
			currentVowelCount--
		}

		if currentVowelCount > maxVowelCount {
			maxVowelCount = currentVowelCount
			if maxVowelCount == k {
				return k
			}
		}
	}

	return maxVowelCount
}
