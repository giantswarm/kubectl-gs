package feature

import (
	"strings"

	semver "github.com/blang/semver/v4"
)

// Service is used to compute different provider capabilities.
type Service struct {
	provider string
	features map[string]Feature
}

func New(provider string) *Service {
	s := &Service{
		provider: provider,
		features: map[string]Feature{
			Autoscaling:        autoscaling,
			NodePoolConditions: nodePoolConditions,
			ClientCert:         clientCert,
		},
	}

	return s
}

// Supports checks if a certain feature is supported or not on a given
// workload cluster release version.
func (s *Service) Supports(featureName string, releaseVersion string) bool {
	feature, exists := s.features[featureName]
	if !exists {
		return false
	}

	capability, exists := feature[s.provider]
	if !exists {
		return false
	}

	releaseVersion = strings.TrimPrefix(releaseVersion, "v")
	inputVersion := semver.MustParse(releaseVersion)
	featureMinVersion := semver.MustParse(capability.MinVersion)

	return inputVersion.GE(featureMinVersion)
}
