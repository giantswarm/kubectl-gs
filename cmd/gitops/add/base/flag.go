package app

import (
	"github.com/spf13/cobra"
)

//const ()

type flag struct {
}

func (f *flag) Init(cmd *cobra.Command) {

}

func (f *flag) Validate() error {
	return nil
}
