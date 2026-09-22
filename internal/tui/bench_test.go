package tui

import "testing"

// renderStatusBar is deliberately uncached and runs on every frame, so its
// cost is paid at the render rate. This benchmark exists to decide whether
// caching it is worth the staleness risk that caching previously caused.
func BenchmarkRenderStatusBar(b *testing.B) {
	m := newTestModel()
	m.width = 120
	m.height = 30

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if s := m.renderStatusBar(); s == "" {
			b.Fatal("renderStatusBar returned empty")
		}
	}
}

// viewForm is the QSO form render path; it is signature-cached, so this
// measures the cache-hit cost that dominates normal typing.
func BenchmarkViewFormCached(b *testing.B) {
	m := newTestModel()
	m.width = 120
	m.height = 30
	_ = m.viewForm(110) // prime the cache

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = m.viewForm(110)
	}
}
