package tests

// mockChecker реализует urlcheck.Checker без реального HTTP.
// Используется во всех тестах этого пакета.
type mockChecker struct {
	code int
	err  error
}

func (m mockChecker) Check(_ string) (int, error) {
	return m.code, m.err
}