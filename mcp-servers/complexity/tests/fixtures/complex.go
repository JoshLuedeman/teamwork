package main

func complexFunction(x int, y int, z int) int {
	result := 0
	if x > 0 {
		if y > 0 {
			if z > 0 {
				result = x + y + z
			} else if z == 0 {
				result = x + y
			} else {
				result = x - y
			}
		} else {
			for i := 0; i < x; i++ {
				if i%2 == 0 {
					result += i
				} else {
					result -= i
				}
			}
		}
	} else if x == 0 {
		switch y {
		case 1:
			result = 10
		case 2:
			result = 20
		case 3:
			result = 30
		default:
			result = -1
		}
	}
	return result
}
