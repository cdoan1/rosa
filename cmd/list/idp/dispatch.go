// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package idp

import (
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
)

var (
	hfEnabled = hyperfleet.Enabled
	runV1     = run
)

func dispatch(cmd *cobra.Command, args []string) {
	if hfEnabled() {
		runV2()
		return
	}
	runV1(cmd, args)
}
