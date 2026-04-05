package traverse

import (
	"strings"
	"testing"
)

func TestApply_IncludedInURL(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Sales",
		urlDirty:  true,
	}
	qb.Apply("groupby((Region),aggregate(Amount with sum as Total))")
	u := qb.buildURL()
	if !strings.Contains(u, "$apply=") {
		t.Errorf("buildURL() missing $apply, got: %s", u)
	}
	if !strings.Contains(u, "groupby") {
		t.Errorf("buildURL() $apply value missing, got: %s", u)
	}
}

func TestApply_Empty_NotIncluded(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Sales",
		urlDirty:  true,
	}
	u := qb.buildURL()
	if strings.Contains(u, "$apply") {
		t.Errorf("buildURL() should not include $apply when not set, got: %s", u)
	}
}

func TestApply_Chaining(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	result := qb.Apply("aggregate(Amount with sum as Total)")
	if result != qb {
		t.Error("Apply() should return the same QueryBuilder for chaining")
	}
	if qb.apply != "aggregate(Amount with sum as Total)" {
		t.Errorf("Apply() did not set apply field, got: %q", qb.apply)
	}
	if !qb.urlDirty {
		t.Error("Apply() should set urlDirty=true")
	}
}
