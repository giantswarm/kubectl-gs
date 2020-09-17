package azure

func AvailableAZs(region string) int {
	azs, ok := infrastructure[region]
	if !ok {
		return 0
	}

	return len(azs)
}

func GetAvailabilityZones(num int, region string) []string {
	var azs []string
	for i := 0; i < num; i++ {
		azs = append(azs, infrastructure[region][i])
	}

	return azs
}

func ValidateRegion(region string) bool {
	for r := range infrastructure {
		if r == region {
			return true
		}
	}

	return false
}

func ValidateAZ(region, availabilityZone string) bool {
	for _, az := range infrastructure[region] {
		if az == availabilityZone {
			return true
		}
	}

	return false
}
