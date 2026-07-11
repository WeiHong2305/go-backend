package leetcode

func longestOnes(nums []int, k int) int {
	i, j, longestOnes := 0, 0, 0
	zeros := []int{}

	for j < len(nums) {
		if nums[j] == 1 {
			j++
			continue
		}

		zeros = append(zeros, j)

		if k > 0 {
			k--
			j++
			continue
		}

		length := j - i
		if length > longestOnes {
			longestOnes = length
		}

		currentFirstZeroPosition := zeros[0]
		zeros = zeros[1:]

		i = currentFirstZeroPosition + 1
		j++
	}

	length := j - i
	if length > longestOnes {
		longestOnes = length
	}

	return longestOnes
}
