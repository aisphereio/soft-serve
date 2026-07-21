package store

import (
	"context"
	"reflect"
	"testing"

	"github.com/aisphereio/soft-serve/pkg/db"
	"github.com/aisphereio/soft-serve/pkg/db/models"
)

func TestRepositoryCreateExtensionsPreserveOrder(t *testing.T) {
	var got []int
	first := func(context.Context, db.Handler, models.Repo) error {
		got = append(got, 1)
		return nil
	}
	second := func(context.Context, db.Handler, models.Repo) error {
		got = append(got, 2)
		return nil
	}

	ctx := WithRepositoryCreateExtension(context.Background(), first)
	ctx = WithRepositoryCreateExtension(ctx, second)
	for _, extension := range RepositoryCreateExtensionsFromContext(ctx) {
		if err := extension(ctx, nil, models.Repo{}); err != nil {
			t.Fatal(err)
		}
	}

	if want := []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("extensions ran in order %v, want %v", got, want)
	}
}
