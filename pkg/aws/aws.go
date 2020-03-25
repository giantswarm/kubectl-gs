package aws

import (
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

func ValidateRegion(region string) bool {
	resolver := endpoints.DefaultResolver()
	partitions := resolver.(endpoints.EnumPartitions).Partitions()

	for _, p := range partitions {
		if p.ID() == "aws" {
			for id := range p.Regions() {
				if region == id {
					return true
				}
			}
		}
	}

	return false
}
