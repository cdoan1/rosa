// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package externalauthprovider

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var (
	hyperfleetEnabled = hyperfleet.Enabled
	runV1             = run
)

func dispatch(cmd *cobra.Command, args []string) {
	if hyperfleetEnabled() {
		hfDeleteExternalAuthProvider(cmd, args)
		return
	}
	runV1(cmd, args)
}
