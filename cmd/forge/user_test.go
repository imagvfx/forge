package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/imagvfx/forge"
)

func addAdmin(server *forge.Server, ctx context.Context) error {
	admin := "admin@imagvfx.com"
	u := &forge.User{
		Name:   admin,
		Called: "admin of tests",
	}
	err := server.AddUser(ctx, u)
	if err != nil {
		return err
	}
	got, err := server.GetUser(ctx, admin)
	if err != nil {
		return err
	}
	if got != nil {
		u.ID = got.ID
	}
	if !reflect.DeepEqual(got, u) {
		return fmt.Errorf("got %v, want %v", got, u)
	}
	return nil
}

func TestAddAdmin(t *testing.T) {
	db, server, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	err = addAdmin(server, ctx)
	if err != nil {
		t.Fatal(err)
	}
}
