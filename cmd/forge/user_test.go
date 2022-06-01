package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/imagvfx/forge"
)

func addUser(server *forge.Server, ctx context.Context, user string) error {
	u := &forge.User{
		Name: user,
	}
	err := server.AddUser(ctx, u)
	if err != nil {
		return err
	}
	got, err := server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	if user != got.Name {
		return fmt.Errorf("got %v, want %v", got, u)
	}
	return nil
}

func TestAddUser(t *testing.T) {
	db, server, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	err = addUser(server, ctx, "admin@imagvfx.com")
	if err != nil {
		t.Fatal(err)
	}
}
