package claims

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var claimReverseOpenFailTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "claim_reverse_open_fail_total",
	Help: "Sync reverse-logistics OpenFromClaim failures after claim file (G12); async consumer retries.",
})

func incClaimReverseOpenFail() {
	claimReverseOpenFailTotal.Inc()
}
