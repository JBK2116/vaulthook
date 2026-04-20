package auth

import (
	"context"
	"testing"
)

func TestDelete(t *testing.T) {
	ctx := context.Background()
	beforeEach(t)
	// create and delete the same token
	token := createRefreshToken(t)
	err := testRepo.Delete(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	afterEach(t)
}

func TestExists(t *testing.T) {
	ctx := context.Background()
	beforeEach(t)
	// create and insert the token
	token := createRefreshToken(t)
	// token exists in the database
	exists, err := testRepo.Exists(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected token %s, found nothing", token)
	}
	// delete the token
	if err := testRepo.Delete(ctx, token); err != nil {
		t.Fatal(err)
	}
	// token should not exist in the database anymore
	exists, err = testRepo.Exists(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("expected no token, found something ...")
	}
	afterEach(t)
}
