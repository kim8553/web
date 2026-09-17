package exchangebinding

import "testing"

func TestPreviewFromRuntimeBindStatusDisabledHides(t *testing.T) {
	for _, bindStatus := range []int32{-1, 0} {
		got := PreviewFromRuntime(bindStatus, 1, 1)
		if got.Visible || got.Bound {
			t.Fatalf("bindStatus=%d preview=%+v want hidden zero state", bindStatus, got)
		}
	}
}

func TestPreviewFromRuntimeShowBindZeroHides(t *testing.T) {
	got := PreviewFromRuntime(1, 0, 1)
	if got.Visible || got.Bound {
		t.Fatalf("preview=%+v want hidden zero state", got)
	}
}

func TestPreviewFromRuntimeDefaultsToVisibleUnbound(t *testing.T) {
	got := PreviewFromRuntime(1, 1, 0)
	if !got.Visible || got.Bound {
		t.Fatalf("preview=%+v want visible unbound", got)
	}
}

func TestPreviewFromRuntimeExchangeBindPositiveShowsBound(t *testing.T) {
	got := PreviewFromRuntime(1, 1, 1)
	if !got.Visible || !got.Bound {
		t.Fatalf("preview=%+v want visible bound", got)
	}
}

func TestPreviewFromRuntimeUsesExactComparisonDirections(t *testing.T) {
	// Exact current Lua tests BindStatus > 0, ExchangeBind > 0, and only
	// ShowBind == 0 for suppression. Preserve those comparison directions.
	got := PreviewFromRuntime(1, -1, -1)
	if !got.Visible || got.Bound {
		t.Fatalf("preview=%+v want visible unbound for nonzero ShowBind and nonpositive ExchangeBind", got)
	}
}
