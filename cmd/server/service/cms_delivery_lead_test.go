package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"kbank-ecms/internal/delivery/http/dto"
	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/domain/entity/enums"
	domainrepo "kbank-ecms/internal/domain/repository"
)

// captureLeadRepo is a LeadRepository test double that records the last
// LeadQuery it received so tests can assert on the channel/placements
// forwarded by fetchLeadsForRequest.
type captureLeadRepo struct {
	leads     []entity.Lead
	err       error
	lastQuery domainrepo.LeadQuery
	calls     int
}

func (f *captureLeadRepo) GetLeads(_ context.Context, q domainrepo.LeadQuery) ([]entity.Lead, error) {
	f.calls++
	f.lastQuery = q
	return f.leads, f.err
}

func TestFetchLeadsForRequest_SkipsWhenNoSalesTargetRule(t *testing.T) {
	repo := &captureLeadRepo{leads: []entity.Lead{{LeadID: "x"}}}
	svc := &CMSDeliveryService{leadRepo: repo}

	rules := []*entity.DecisionRule{
		{Type: enums.DecisionTypeMass},
		{Type: enums.DecisionTypeAudience},
	}
	got := svc.fetchLeadsForRequest(context.Background(),
		&dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "1234567890"},
		"salesforce",
		[]string{"wsaHomeBanner"},
		rules,
	)

	assert.Nil(t, got)
	assert.Equal(t, 0, repo.calls, "leadRepo must not be called when no SALES_TARGET rule")
}

func TestFetchLeadsForRequest_SkipsForNonCISCustomer(t *testing.T) {
	repo := &captureLeadRepo{leads: []entity.Lead{{LeadID: "x"}}}
	svc := &CMSDeliveryService{leadRepo: repo}

	got := svc.fetchLeadsForRequest(context.Background(),
		&dto.CustomerRequest{Type: dto.CustomerIdTypeIPID, IP_ID: "999"},
		"salesforce",
		[]string{"wsaHomeBanner"},
		[]*entity.DecisionRule{{Type: enums.DecisionTypeSalesTarget}},
	)
	assert.Nil(t, got)
	assert.Equal(t, 0, repo.calls)
}

func TestFetchLeadsForRequest_SkipsWhenLeadRepoNil(t *testing.T) {
	svc := &CMSDeliveryService{leadRepo: nil}
	got := svc.fetchLeadsForRequest(context.Background(),
		&dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "1234567890"},
		"salesforce",
		[]string{"wsaHomeBanner"},
		[]*entity.DecisionRule{{Type: enums.DecisionTypeSalesTarget}},
	)
	assert.Nil(t, got)
}

func TestFetchLeadsForRequest_CallsRepoWithCorrectQuery(t *testing.T) {
	wantLeads := []entity.Lead{{LeadID: "lead-1", ContentID: "/dam/a"}}
	repo := &captureLeadRepo{leads: wantLeads}
	svc := &CMSDeliveryService{leadRepo: repo}

	got := svc.fetchLeadsForRequest(context.Background(),
		&dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "1234567890"},
		"salesforce",
		[]string{"wsaHomeBanner", "wsaSplash"},
		[]*entity.DecisionRule{
			{Type: enums.DecisionTypeMass},
			{Type: enums.DecisionTypeSalesTarget},
		},
	)

	assert.Equal(t, wantLeads, got)
	assert.Equal(t, 1, repo.calls)
	assert.Equal(t, "1234567890", repo.lastQuery.CisID)
	assert.Equal(t, "salesforce", repo.lastQuery.Channel)
	assert.Equal(t, []string{"wsaHomeBanner", "wsaSplash"}, repo.lastQuery.Placements)
}

func TestFetchLeadsForRequest_SwallowsRepoError(t *testing.T) {
	repo := &captureLeadRepo{err: errors.New("CLEN down")}
	svc := &CMSDeliveryService{leadRepo: repo}

	got := svc.fetchLeadsForRequest(context.Background(),
		&dto.CustomerRequest{Type: dto.CustomerIdTypeCISID, CIS_ID: "1234567890"},
		"salesforce",
		[]string{"wsaHomeBanner"},
		[]*entity.DecisionRule{{Type: enums.DecisionTypeSalesTarget}},
	)
	assert.Nil(t, got, "errors must be swallowed so non-fatal CLEN outage degrades gracefully")
	assert.Equal(t, 1, repo.calls)
}
