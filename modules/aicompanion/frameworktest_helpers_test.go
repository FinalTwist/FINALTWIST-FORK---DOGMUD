package aicompanion

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
)

// The server's budget and breaker are apiframework's, shared by every
// feature on a server. In this test binary each module gets its own set
// (isolateBooks, m.fw()), so a call one test left in flight can never land
// in another test's figures. The binary starts with no key (a developer's
// OPENAI_API_KEY is cleared in TestMain, so no test can ever make a real
// call or print a real key), and freshServer sets the server settings a test
// weighs its budget and breaker against.

// frameworkForTests is what apiframework.Server returns in this binary
// unless a test says otherwise: no key, the official endpoint, the default
// budget and breaker.
func frameworkForTests() {
	isolateBooks = true
	_ = os.Unsetenv(`OPENAI_API_KEY`)
	apiframework.SetServerForTest(apiframework.ServerSettings{
		Endpoint:         apiframework.Endpoint{BaseURL: apiframework.DefaultBaseURL},
		DailyTokenBudget: 2000000,
		BreakerErrors:    5,
		BreakerSeconds:   60,
	})
	apiframework.ResetBudgetForTest(``)
	apiframework.ResetBreaker()
}

// freshServer gives one test the server budget and breaker settings it
// names, and puts the defaults back after. budget 0 is no cap. Each module's
// own books start with nothing spent and the breaker closed.
func freshServer(t *testing.T, budget int, breakerErrors int, breakerSeconds int) {
	t.Helper()
	restore := apiframework.SetServerForTest(apiframework.ServerSettings{
		Endpoint:         apiframework.Endpoint{BaseURL: apiframework.DefaultBaseURL},
		DailyTokenBudget: budget,
		BreakerErrors:    breakerErrors,
		BreakerSeconds:   breakerSeconds,
	})
	apiframework.ResetBudgetForTest(``)
	apiframework.ResetBreaker()
	t.Cleanup(func() {
		restore()
		apiframework.ResetBudgetForTest(``)
		apiframework.ResetBreaker()
	})
}

// serverSpent and serverHeld are the day's server tokens and what calls in
// flight still hold.
func serverSpent(m *AICompanionModule) int { return m.fw().Today().Tokens }
func serverHeld(m *AICompanionModule) int  { return m.fw().Today().Outstanding }

// pointAt sends this module's server-key calls to baseURL with key.
func pointAt(m *AICompanionModule, baseURL string, key string) {
	m.endpoint = &apiframework.Endpoint{BaseURL: baseURL, APIKey: key}
}

// The day's per-user allowances, as the ledger keeps them.
func ownerSpent(m *AICompanionModule, ownerId int) int {
	return m.fw().Allowance(apiframework.DimCompanionOwner, ownerId)
}
func strangerSpent(m *AICompanionModule, askerId int) int {
	return m.fw().Allowance(apiframework.DimCompanionStranger, askerId)
}
func strangersForSpent(m *AICompanionModule, ownerId int) int {
	return m.fw().Allowance(apiframework.DimCompanionStrangersFor, ownerId)
}
func setOwnerSpent(m *AICompanionModule, ownerId int, tokens int) {
	m.fw().SetAllowanceForTest(apiframework.DimCompanionOwner, ownerId, tokens)
}
func setStrangerSpent(m *AICompanionModule, askerId int, tokens int) {
	m.fw().SetAllowanceForTest(apiframework.DimCompanionStranger, askerId, tokens)
}
func setStrangersForSpent(m *AICompanionModule, ownerId int, tokens int) {
	m.fw().SetAllowanceForTest(apiframework.DimCompanionStrangersFor, ownerId, tokens)
}
