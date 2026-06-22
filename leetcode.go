package main

func moveZeroes(nums []int) {
	writer := 0

	for reader := range nums {
		if nums[reader] != 0 {
			nums[writer], nums[reader] = nums[reader], nums[writer]
			nums[writer] = nums[reader]
			writer++
		}
	}
}
