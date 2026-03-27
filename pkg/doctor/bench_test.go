package doctor_test

import (
	"io"
	"testing"

	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
)

func BenchmarkRun(b *testing.B) {
	noop := func() pkgdoctor.CheckResult {
		return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "ok"}
	}
	checks := make([]pkgdoctor.CheckFunc, 10)
	for i := range checks {
		checks[i] = noop
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pkgdoctor.Run(checks, io.Discard)
	}
}
