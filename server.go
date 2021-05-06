package forge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"github.com/imagvfx/forge/service"
	"golang.org/x/image/draw"
)

type Server struct {
	svc service.Service
	cfg *Config
}

func NewServer(svc service.Service, cfg *Config) *Server {
	s := &Server{
		svc: svc,
		cfg: cfg,
	}
	return s
}

func (s *Server) GetEntry(ctx context.Context, path string) (*Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("path emtpy")
	}
	e, err := s.svc.GetEntry(ctx, path)
	if err != nil {
		return nil, err
	}
	ent := &Entry{
		ID:           e.ID,
		Path:         e.Path,
		Type:         e.Type,
		HasThumbnail: e.HasThumbnail,
	}
	return ent, nil
}

func (s *Server) SubEntries(ctx context.Context, path string) ([]*Entry, error) {
	es, err := s.svc.FindEntries(ctx, service.EntryFinder{
		ParentPath: &path,
	})
	if err != nil {
		return nil, err
	}
	ents := make([]*Entry, 0)
	for _, e := range es {
		ent := &Entry{
			ID:           e.ID,
			Path:         e.Path,
			Type:         e.Type,
			HasThumbnail: e.HasThumbnail,
		}
		ents = append(ents, ent)
	}
	return ents, nil
}

func (s *Server) AddEntry(ctx context.Context, path, typ string) error {
	path = filepath.ToSlash(path)
	parent := filepath.Dir(path)
	p, err := s.svc.GetEntry(ctx, parent)
	if err != nil {
		return fmt.Errorf("error on parent check: %v", err)
	}
	allow := false
	subtyps := s.cfg.Struct[p.Type].SubEntryTypes
	for _, subtyp := range subtyps {
		if subtyp == typ {
			allow = true
			break
		}
	}
	if !allow {
		return fmt.Errorf("cannot create a child of type %q from %q", typ, p.Type)
	}
	e := &service.Entry{
		Path: path,
		Type: typ,
	}
	props := make([]*service.Property, 0)
	for _, ktv := range s.cfg.Struct[typ].Properties {
		p := &Property{
			EntryPath: path,
			Name:      ktv.Key,
			Type:      ktv.Type,
			Value:     ktv.Value,
		}
		err := p.Validate()
		if err != nil {
			return err
		}
		props = append(props, p.ServiceProperty())
	}
	envs := make([]*service.Property, 0)
	for _, ktv := range s.cfg.Struct[typ].Environs {
		e := &Property{
			EntryPath: path,
			Name:      ktv.Key,
			Type:      ktv.Type,
			Value:     ktv.Value,
		}
		err := e.Validate()
		if err != nil {
			return err
		}
		envs = append(envs, e.ServiceProperty())
	}
	err = s.svc.AddEntry(ctx, e, props, envs)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) RenameEntry(ctx context.Context, path, newName string) error {
	err := s.svc.RenameEntry(ctx, path, newName)
	if err != nil {
		return err
	}
	// Move the thumbnail also.
	newPath := filepath.Dir(path) + "/" + newName
	thumbnailRoot := filepath.Join(s.cfg.UserdataRoot, "thumbnail")
	thumbnailDir := filepath.Join(thumbnailRoot, path)
	newThumbnailDir := filepath.Join(thumbnailRoot, newPath)
	_, err = os.Stat(thumbnailDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	err = os.Rename(thumbnailDir, newThumbnailDir)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteEntry(ctx context.Context, path string) error {
	err := s.svc.DeleteEntry(ctx, path)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryProperties(ctx context.Context, path string) ([]*Property, error) {
	ps, err := s.svc.EntryProperties(ctx, path)
	if err != nil {
		return nil, err
	}
	props := make([]*Property, 0)
	for _, p := range ps {
		prop := &Property{
			ID:        p.ID,
			EntryPath: p.EntryPath,
			Name:      p.Name,
			Type:      p.Type,
			Value:     p.Value,
		}
		props = append(props, prop)
	}
	return props, nil
}

func (s *Server) getProperty(ctx context.Context, path string, name string) (*Property, error) {
	p, err := s.svc.GetProperty(ctx, path, name)
	if err != nil {
		return nil, err
	}
	prop := &Property{
		ID:        p.ID,
		EntryPath: p.EntryPath,
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.Value,
	}
	return prop, nil
}

func (s *Server) AddProperty(ctx context.Context, path string, name, typ, value string) error {
	ent, err := s.svc.GetEntry(ctx, path)
	if err != nil {
		return err
	}
	env := &Property{
		EntryPath: ent.Path,
		Name:      name,
		Type:      typ,
		Value:     value,
	}
	err = env.Validate()
	if err != nil {
		return err
	}
	err = s.svc.AddProperty(ctx, env.ServiceProperty())
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetProperty(ctx context.Context, path string, name, value string) error {
	prop, err := s.getProperty(ctx, path, name)
	if err != nil {
		return err
	}
	prop.Value = value // validate the given value
	err = prop.Validate()
	if err != nil {
		return err
	}
	err = s.svc.UpdateProperty(ctx, service.PropertyUpdater{
		EntryPath: path,
		Name:      name,
		Value:     &value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteProperty(ctx context.Context, path string, name string) error {
	err := s.svc.DeleteProperty(ctx, path, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryEnvirons(ctx context.Context, path string) ([]*Property, error) {
	ps, err := s.svc.EntryEnvirons(ctx, path)
	if err != nil {
		return nil, err
	}
	props := make([]*Property, 0)
	for _, p := range ps {
		prop := &Property{
			ID:        p.ID,
			EntryPath: p.EntryPath,
			Name:      p.Name,
			Type:      p.Type,
			Value:     p.Value,
		}
		props = append(props, prop)
	}
	return props, nil
}

func (s *Server) getEnviron(ctx context.Context, path, name string) (*Property, error) {
	e, err := s.svc.GetEnviron(ctx, path, name)
	if err != nil {
		return nil, err
	}
	env := &Property{
		ID:        e.ID,
		EntryPath: e.EntryPath,
		Name:      e.Name,
		Type:      e.Type,
		Value:     e.Value,
	}
	return env, nil
}

func (s *Server) AddEnviron(ctx context.Context, path string, name, typ, value string) error {
	env := &Property{
		EntryPath: path,
		Name:      name,
		Type:      typ,
		Value:     value,
	}
	err := env.Validate()
	if err != nil {
		return err
	}
	err = s.svc.AddEnviron(ctx, env.ServiceProperty())
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetEnviron(ctx context.Context, path string, name, value string) error {
	env, err := s.getEnviron(ctx, path, name)
	if err != nil {
		return err
	}
	env.Value = value // validate the given value
	err = env.Validate()
	if err != nil {
		return err
	}
	err = s.svc.UpdateEnviron(ctx, service.PropertyUpdater{
		EntryPath: path,
		Name:      name,
		Value:     &value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteEnviron(ctx context.Context, path string, name string) error {
	err := s.svc.DeleteEnviron(ctx, path, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryAccessControls(ctx context.Context, path string) ([]*AccessControl, error) {
	as, err := s.svc.EntryAccessControls(ctx, path)
	if err != nil {
		return nil, err
	}
	acs := make([]*AccessControl, 0, len(as))
	for _, a := range as {
		ac := &AccessControl{
			ID:           a.ID,
			EntryPath:    a.EntryPath,
			Accessor:     a.Accessor,
			AccessorType: AccessorType(a.AccessorType),
			Mode:         AccessMode(a.Mode),
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

func (s *Server) AddAccessControl(ctx context.Context, path string, accessor, accessor_type, mode string) error {
	ac := &service.AccessControl{
		EntryPath: path,
		Accessor:  accessor,
	}
	switch accessor_type {
	case "user":
		ac.AccessorType = 0
	case "group":
		ac.AccessorType = 1
	default:
		return fmt.Errorf("unknown accessor type")
	}
	switch mode {
	case "r":
		ac.Mode = int(ReadAccess)
	case "rw":
		ac.Mode = int(ReadWriteAccess)
	default:
		return fmt.Errorf("unknown access type")
	}
	err := s.svc.AddAccessControl(ctx, ac)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetAccessControl(ctx context.Context, path, accessor, mode string) error {
	ac := service.AccessControlUpdater{
		EntryPath: path,
		Accessor:  accessor,
	}
	switch mode {
	case "r":
		m := int(ReadAccess)
		ac.Mode = &m
	case "rw":
		m := int(ReadWriteAccess)
		ac.Mode = &m
	default:
		return fmt.Errorf("unknown access type")
	}
	err := s.svc.UpdateAccessControl(ctx, ac)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteAccessControl(ctx context.Context, path string, name string) error {
	err := s.svc.DeleteAccessControl(ctx, path, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryLogs(ctx context.Context, path string) ([]*Log, error) {
	ls, err := s.svc.FindLogs(ctx, service.LogFinder{
		EntryPath: path,
	})
	if err != nil {
		return nil, err
	}
	logs := make([]*Log, 0)
	for _, l := range ls {
		log := &Log{
			ID:        l.ID,
			EntryPath: l.EntryPath,
			User:      l.User,
			Action:    l.Action,
			Category:  l.Category,
			Name:      l.Name,
			Type:      l.Type,
			Value:     l.Value,
			When:      l.When,
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (s *Server) GetUser(ctx context.Context, user string) (*User, error) {
	su, err := s.svc.GetUserByName(ctx, user)
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:   su.ID,
		Name: su.Name,
	}
	return u, nil
}

func (s *Server) AddUser(ctx context.Context, user string) error {
	u := &service.User{Name: user}
	err := s.svc.AddUser(ctx, u)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) FindAllGroups(ctx context.Context) ([]*Group, error) {
	sgroups, err := s.svc.FindGroups(ctx, service.GroupFinder{})
	if err != nil {
		return nil, err
	}
	groups := make([]*Group, 0, len(sgroups))
	for _, sg := range sgroups {
		g := &Group{
			ID:   sg.ID,
			Name: sg.Name,
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (s *Server) GetGroup(ctx context.Context, name string) (*Group, error) {
	sgroups, err := s.svc.FindGroups(ctx, service.GroupFinder{Name: &name})
	if err != nil {
		return nil, err
	}
	if len(sgroups) == 0 {
		return nil, fmt.Errorf("group not exist: %v", name)
	}
	sg := sgroups[0]
	g := &Group{
		ID:   sg.ID,
		Name: sg.Name,
	}
	return g, nil
}

func (s *Server) AddGroup(ctx context.Context, group string) error {
	g := &service.Group{Name: group}
	err := s.svc.AddGroup(ctx, g)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetGroup(ctx context.Context, groupID string, group string) error {
	id, err := strconv.Atoi(groupID)
	if err != nil {
		return fmt.Errorf("invalid group id: %v", groupID)
	}
	g := service.GroupUpdater{ID: id, Name: &group}
	err = s.svc.UpdateGroup(ctx, g)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) FindGroupMembers(ctx context.Context, group string) ([]*Member, error) {
	svcMembers, err := s.svc.FindGroupMembers(ctx, service.MemberFinder{Group: &group})
	if err != nil {
		return nil, err
	}
	members := make([]*Member, 0, len(svcMembers))
	for _, sm := range svcMembers {
		m := &Member{
			Group:  sm.Group,
			Member: sm.Member,
		}
		members = append(members, m)
	}
	return members, nil
}

func (s *Server) AddGroupMember(ctx context.Context, group, member string) error {
	m := &service.Member{Group: group, Member: member}
	err := s.svc.AddGroupMember(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteGroupMember(ctx context.Context, group, member string) error {
	err := s.svc.DeleteGroupMember(ctx, group, member)
	if err != nil {
		return err
	}
	return nil
}

// GetThumbnail gets a thumbnail image of a entry.
func (s *Server) GetThumbnail(ctx context.Context, path string) (*Thumbnail, error) {
	svcThumb, err := s.svc.GetThumbnail(ctx, path)
	if err != nil {
		return nil, err
	}
	thumb := &Thumbnail{
		ID:        svcThumb.ID,
		Data:      svcThumb.Data,
		EntryPath: svcThumb.EntryPath,
	}
	return thumb, nil
}

func thumbnail(img image.Image, width, height int) image.Image {
	thumb := image.NewRGBA(image.Rect(0, 0, 192, 108))
	thumbBounds := thumb.Bounds()
	imgWidth := float64(img.Bounds().Dx())
	imgHeight := float64(img.Bounds().Dy())
	xs := imgWidth / 192
	ys := imgHeight / 108
	if xs > ys {
		scaledHeight := int(imgHeight / xs)
		marginY := (108 - scaledHeight) / 2
		thumbBounds.Min.Y += marginY
		thumbBounds.Max.Y -= marginY
	} else {
		scaledWidth := int(imgWidth / ys)
		marginX := (192 - scaledWidth) / 2
		thumbBounds.Min.X += marginX
		thumbBounds.Max.X -= marginX
	}
	draw.CatmullRom.Scale(thumb, thumbBounds, img, img.Bounds(), draw.Over, nil)
	return thumb
}

// AddThumbnail adds a thumbnail image to a entry.
func (s *Server) AddThumbnail(ctx context.Context, path string, img image.Image) error {
	thumb := thumbnail(img, 192, 108)
	buf := new(bytes.Buffer)
	err := png.Encode(buf, thumb)
	if err != nil {
		return err
	}
	err = s.svc.AddThumbnail(ctx, &service.Thumbnail{
		EntryPath: path,
		Data:      buf.Bytes(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateThumbnail(ctx context.Context, path string, img image.Image) error {
	thumb := thumbnail(img, 192, 108)
	buf := new(bytes.Buffer)
	err := png.Encode(buf, thumb)
	if err != nil {
		return err
	}
	err = s.svc.UpdateThumbnail(ctx, service.ThumbnailUpdater{
		EntryPath: path,
		Data:      buf.Bytes(),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteThumbnail(ctx context.Context, path string) error {
	err := s.svc.DeleteThumbnail(ctx, path)
	if err != nil {
		return err
	}
	return nil
}
