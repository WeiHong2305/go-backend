package leetcode

func longestSubarray(nums []int) int {
	left, maxLen, zeroCount := 0, 0, 0

	for right := range nums {
		if nums[right] == 0 {
			zeroCount++
		}

		for zeroCount > 1 {
			if nums[left] == 0 {
				zeroCount--
			}
			left++
		}
		if right-left > maxLen {
			maxLen = right - left
		}
	}
	return maxLen
}
