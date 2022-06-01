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

func addGroup(server *forge.Server, ctx context.Context, group string) error {
	g := &forge.Group{
		Name: group,
	}
	err := server.AddGroup(ctx, g)
	if err != nil {
		return err
	}
	got, err := server.GetGroup(ctx, group)
	if err != nil {
		return err
	}
	if group != got.Name {
		return fmt.Errorf("got %v, want %v", got, g)
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
	ctx = forge.ContextWithUserName(ctx, "admin@imagvfx.com")
	err = addGroup(server, ctx, "pms")
	if err != nil {
		t.Fatal(err)
	}
}
