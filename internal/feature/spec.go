package feature

type Capability struct {
	MinVersion string
}

type Feature map[string]Capability

type Map = map[string]Feature
