package modifier

import "github.com/giantswarm/microerror"

// My idea for now was to create some sort of modifiers
// that consume []byte input, modify it accordingly to the
// provided configuration, and return []byte that Creator can
// write to a file.
type Modifier interface {
	Execute([]byte) ([]byte, error)
}

// IfExists runs its modifier only on a file that exists. The creator leaves
// the file out when it doesn't, where it fails for a plain modifier. It is for
// files that are optional in the structure, such as a workload cluster's apps
// Kustomization.
type IfExists struct {
	Modifier
}

// Chain runs its modifiers one after the other on the same file.
type Chain []Modifier

// Execute is the interface used by the creator to execute post modifier.
func (c Chain) Execute(data []byte) ([]byte, error) {
	var err error
	for _, m := range c {
		data, err = m.Execute(data)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return data, nil
}
