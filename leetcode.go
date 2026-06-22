package main

func moveZeroes(nums []int) {
	reader, writer := 0, 0

	for reader < len(nums) {
		if nums[reader] != 0 {
			found := false
			for writer < reader && !found {
				if nums[writer] == 0 {
					nums[writer], nums[reader] = nums[reader], nums[writer]
					found = true
				}
				writer++
			}
		}
		reader++
	}
}
