package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	n := nums[1] - nums[0]
	stats := Stats{
		Count: 1,
		Sum:   n,
		Min:   n,
		Max:   n,
	}

	for i := 2; i < len(nums); i++	{
		n = nums[i] - nums[i-1]
		stats.Count++
		stats.Sum += n
		if n < stats.Min {
			stats.Min = n
		}
		if n > stats.Max {
			stats.Max = n
		}
	}
	return stats
}
