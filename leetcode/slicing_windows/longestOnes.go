package leetcode

// Sliding windows (Medium)
func longestOnes(nums []int, k int) int {
	left, maxLen, zeroCount := 0, 0, 0

	for right := range nums {
		if nums[right] == 0 {
			zeroCount++
		}
		for zeroCount > k {
			if nums[left] == 0 {
				zeroCount--
			}
			left++
		}

		if right-left+1 > maxLen {
			maxLen = right - left + 1
		}
	}
	return maxLen
}
