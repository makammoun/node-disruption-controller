package controller

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nodedisruptionv1alpha1 "github.com/criteo/node-disruption-controller/api/v1alpha1"
)

type fixedNoticeBudget struct {
	Budget
	notice time.Duration
}

func (b fixedNoticeBudget) NoticePeriodFor(_ string) time.Duration { return b.notice }

func TestNoticeOpening(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	start := metav1.NewTime(now.Add(2 * time.Hour))
	deadline := metav1.NewTime(now.Add(3 * time.Hour))
	earlyDeadline := metav1.NewTime(now.Add(30 * time.Minute))
	openingDeadline := metav1.NewTime(start.Add(-time.Hour))
	zero := metav1.Time{}

	cases := []struct {
		name     string
		notice   time.Duration
		start    metav1.Time
		retry    bool
		deadline metav1.Time
		want     time.Time // zero = open now
	}{
		{"zero notice never gates", 0, start, true, deadline, time.Time{}},
		{"startDate anchor wins", time.Hour, start, true, deadline, start.Add(-time.Hour)},
		{"deadline alone does not gate", time.Hour, zero, true, deadline, time.Time{}},
		{"no anchor no gating", time.Hour, zero, true, zero, time.Time{}},
		{"past opening is open", 4 * time.Hour, start, true, deadline, time.Time{}},
		{"opening after deadline is open", time.Hour, start, true, earlyDeadline, time.Time{}},
		{"opening at deadline is open", time.Hour, start, true, openingDeadline, time.Time{}},
		{"deadline ignored without retry", time.Hour, start, false, earlyDeadline, start.Add(-time.Hour)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ndr := &SingleNodeDisruptionReconciler{
				NodeDisruption: nodedisruptionv1alpha1.NodeDisruption{
					Spec: nodedisruptionv1alpha1.NodeDisruptionSpec{
						Type:      "maintenance",
						StartDate: c.start,
						Retry:     nodedisruptionv1alpha1.RetrySpec{Enabled: c.retry, Deadline: c.deadline},
					},
				},
			}
			got := ndr.noticeOpening(fixedNoticeBudget{notice: c.notice}, now)
			if got == nil {
				if !c.want.IsZero() {
					t.Fatalf("got open, want %v", c.want)
				}
				return
			}
			if !got.Equal(c.want) {
				t.Fatalf("got %v, want %v", *got, c.want)
			}
		})
	}
}
