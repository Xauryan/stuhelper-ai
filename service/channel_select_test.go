package service

import "testing"

func TestRetryParamExcludeChannel(t *testing.T) {
	p := &RetryParam{}

	// Ignores non-positive IDs and lazily allocates the set.
	p.ExcludeChannel(0)
	p.ExcludeChannel(-3)
	if p.ExcludeChannelIDs != nil {
		t.Fatalf("expected no allocation for non-positive IDs, got %v", p.ExcludeChannelIDs)
	}

	p.ExcludeChannel(7)
	p.ExcludeChannel(7)
	p.ExcludeChannel(9)
	if len(p.ExcludeChannelIDs) != 2 {
		t.Fatalf("expected 2 excluded channels, got %d", len(p.ExcludeChannelIDs))
	}
	if _, ok := p.ExcludeChannelIDs[7]; !ok {
		t.Fatalf("expected channel 7 to be excluded")
	}
	if _, ok := p.ExcludeChannelIDs[9]; !ok {
		t.Fatalf("expected channel 9 to be excluded")
	}
}

func TestMergeExcludeChannelIDs(t *testing.T) {
	a := map[int]struct{}{1: {}, 2: {}}
	b := map[int]struct{}{2: {}, 3: {}}

	merged := mergeExcludeChannelIDs(a, b)
	if len(merged) != 3 {
		t.Fatalf("expected union of 3 ids, got %d (%v)", len(merged), merged)
	}
	for _, id := range []int{1, 2, 3} {
		if _, ok := merged[id]; !ok {
			t.Fatalf("expected merged set to contain %d", id)
		}
	}

	// Inputs must not be mutated.
	if len(a) != 2 || len(b) != 2 {
		t.Fatalf("merge must not mutate inputs: a=%v b=%v", a, b)
	}

	// Empty cases return the other set directly.
	if got := mergeExcludeChannelIDs(nil, b); len(got) != 2 {
		t.Fatalf("expected b returned when a empty, got %v", got)
	}
	if got := mergeExcludeChannelIDs(a, nil); len(got) != 2 {
		t.Fatalf("expected a returned when b empty, got %v", got)
	}
}
