package util_test

import (
	"reflect"
	"testing"

	"kbank-ecms/pkg/util"
)

func TestUniqueStringsSlice(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, []string{}},
		{"no-dupes", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"dupes-preserve-order", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all-same", []string{"x", "x", "x"}, []string{"x"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := util.UniqueStringsSlice(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
