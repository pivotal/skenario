package serve

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestServePkg(t *testing.T) {
	spec.Run(t, "RunHandler", testRunHandler, spec.Report(report.Terminal{}), spec.Sequential())

	var server *SkenarioServer
	server = &SkenarioServer{IndexRoot: "."}
	server.Serve()

	spec.Run(t, "Acceptance test", testAcceptance, spec.Report(report.Terminal{}), spec.Sequential())

	server.Shutdown()
}
