package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type eventSinkCtxKey struct{}

func withEventSink(ctx context.Context, sink EventSink) context.Context {
	return context.WithValue(ctx, eventSinkCtxKey{}, sink)
}

func eventSinkFromContext(ctx context.Context) EventSink {
	sink, _ := ctx.Value(eventSinkCtxKey{}).(EventSink)
	return sink
}

func (e *Executor) runExecutionPlan(ctx context.Context, sink EventSink, st *state.SessionState, plan ExecutionPlan, message string) (specialists.Result, error) {
	if len(plan.Domains) == 0 {
		return specialists.Result{}, fmt.Errorf("execution plan requires at least one domain")
	}

	ctx = withEventSink(ctx, sink)
	results := make([]specialists.Result, len(plan.Domains))
	for _, domain := range plan.Domains {
		domainPlan := plan
		domainPlan.Domains = []string{domain}
		domainPlan.RequiredArtifacts = selectRequiredArtifacts(domainPlan.Domains)
		if err := validatePlanArtifacts(st, domainPlan); err != nil {
			return specialists.Result{}, err
		}
	}

	var (
		wg       sync.WaitGroup
		firstErr error
		errMu    sync.Mutex
	)

	for i, domain := range plan.Domains {
		wg.Add(1)
		go func(idx int, domain string) {
			defer wg.Done()

			runner, ok := e.specialistRegistry.RunnerFor(domain)
			if !ok {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("no specialist runner registered for %s", domain)
				}
				errMu.Unlock()
				return
			}

			route := plan.Route
			route.PrimaryDomain = domain
			route.SecondaryDomains = secondaryDomainsForPlan(plan.Domains, domain)
			result, err := runner.Run(ctx, specialists.Request{
				SessionID:      st.SessionID,
				UserMessage:    message,
				Route:          route,
				ManagerContext: st.ManagerContext,
				DomainContext:  *domainContextFor(st, domain),
				Session:        st,
			})
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			results[idx] = result
		}(i, domain)
	}

	wg.Wait()
	if firstErr != nil {
		return specialists.Result{}, firstErr
	}

	summaries := make([]string, 0, len(results))
	domainNames := make([]string, 0, len(results))
	for i, result := range results {
		if summary := strings.TrimSpace(result.NormalizedSummary()); summary != "" {
			summaries = append(summaries, summary)
		}
		name := strings.TrimSpace(result.Domain)
		if name == "" {
			name = plan.Domains[i]
		}
		domainNames = append(domainNames, name)
	}

	aggregated := specialists.Result{
		Domain: strings.Join(domainNames, "+"),
	}
	if len(summaries) > 0 {
		aggregated.Summary = strings.Join(summaries, "\n\n")
	}
	if len(results) == 1 {
		if aggregated.Summary == "" {
			aggregated.Summary = results[0].NormalizedSummary()
		}
		if aggregated.Domain == "" {
			aggregated.Domain = results[0].Domain
		}
		aggregated.ManagerBrief = results[0].ManagerBrief
		aggregated.MissingSlots = results[0].MissingSlots
		aggregated.DomainContextPatch = results[0].DomainContextPatch
	}
	return aggregated, nil
}

func secondaryDomainsForPlan(domains []string, primary string) []string {
	if len(domains) <= 1 {
		return nil
	}
	secondary := make([]string, 0, len(domains)-1)
	for _, domain := range domains {
		if domain == "" || domain == primary {
			continue
		}
		secondary = append(secondary, domain)
	}
	return secondary
}
