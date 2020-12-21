package feature

import (
	"github.com/blang/semver"
)

type Service struct {
	provider string
	features Map
}

func New(provider string) *Service {
	s := &Service{
		provider: provider,
		features: Map{
			Autoscaling: autoscaling,
			Conditions:  conditions,
		},
	}

	return s
}

func (s *Service) Supports(featureName string, releaseVersion string) bool {
	feature, exists := s.features[featureName]
	if !exists {
		return false
	}

	capability, exists := feature[s.provider]
	if !exists {
		return false
	}

	inputVersion := semver.MustParse(releaseVersion)
	featureMinVersion := semver.MustParse(capability.MinVersion)

	return inputVersion.GE(featureMinVersion)
}
