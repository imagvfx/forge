package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/imagvfx/forge"
)

func addAdmin(server *forge.Server, ctx context.Context) (*forge.User, error) {
	admin := "admin@imagvfx.com"
	u := &forge.User{
		Name:   admin,
		Called: "admin of tests",
	}
	err := server.AddUser(ctx, u)
	if err != nil {
		return nil, err
	}
	got, err := server.GetUser(ctx, admin)
	if err != nil {
		return nil, err
	}
	if got != nil {
		u.ID = got.ID
	}
	if !reflect.DeepEqual(got, u) {
		return nil, fmt.Errorf("got %v, want %v", got, u)
	}
	return got, nil
}

func TestAddAdmin(t *testing.T) {
	db, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server, err := testServer(t, db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, err = addAdmin(server, ctx)
	if err != nil {
		t.Fatal(err)
	}
}
