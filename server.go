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
		return nil, fmt.Errorf("entry path not specified")
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
	if path == "" {
		return nil, fmt.Errorf("entry path not specified")
	}
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
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	if typ == "" {
		return fmt.Errorf("entry type not specified")
	}
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
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	if newName == "" {
		return fmt.Errorf("new entry name not specified")
	}
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
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	err := s.svc.DeleteEntry(ctx, path)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryTypes(ctx context.Context) ([]string, error) {
	names, err := s.svc.EntryTypes(ctx)
	if err != nil {
		return nil, err
	}
	return names, nil
}

func (s *Server) AddEntryType(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("entry type name not specified")
	}
	err := s.svc.AddEntryType(ctx, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) RenameEntryType(ctx context.Context, name, newName string) error {
	if name == "" {
		return fmt.Errorf("current entry type name not specified")
	}
	if newName == "" {
		return fmt.Errorf("new entry type name not specified")
	}
	err := s.svc.RenameEntryType(ctx, name, newName)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteEntryType(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("entry type name not specified")
	}
	err := s.svc.DeleteEntryType(ctx, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryDefaults(ctx context.Context, entType string) ([]*EntryDefault, error) {
	if entType == "" {
		return nil, fmt.Errorf("entry type name not specified")
	}
	ds, err := s.svc.FindEntryDefaults(ctx, service.EntryDefaultFinder{EntryType: &entType})
	if err != nil {
		return nil, err
	}
	defaults := make([]*EntryDefault, 0)
	for _, d := range ds {
		def := &EntryDefault{
			EntryType: d.EntryType,
			Category:  d.Category,
			Type:      d.Type,
			Name:      d.Name,
			Value:     d.Value,
		}
		defaults = append(defaults, def)
	}
	return defaults, nil
}

func (s *Server) AddEntryDefault(ctx context.Context, entType, ctg, name, typ, value string) error {
	if entType == "" {
		return fmt.Errorf("entry default entry type not specified")
	}
	if ctg == "" {
		return fmt.Errorf("entry default category not specified")
	}
	if name == "" {
		return fmt.Errorf("entry default name not specified")
	}
	if typ == "" {
		return fmt.Errorf("entry default type not specified")
	}
	if value == "" {
		return fmt.Errorf("entry default value not specified")
	}
	d := &service.EntryDefault{
		EntryType: entType,
		Category:  ctg,
		Name:      name,
		Type:      typ,
		Value:     value,
	}
	err := s.svc.AddEntryDefault(ctx, d)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetEntryDefault(ctx context.Context, entType, ctg, name, typ, value string) error {
	if entType == "" {
		return fmt.Errorf("entry default entry type not specified")
	}
	if ctg == "" {
		return fmt.Errorf("entry default category not specified")
	}
	if name == "" {
		return fmt.Errorf("entry default name not specified")
	}
	if typ == "" {
		return fmt.Errorf("entry default type not specified")
	}
	if value == "" {
		return fmt.Errorf("entry default value not specified")
	}
	upd := service.EntryDefaultUpdater{
		EntryType: entType,
		Category:  ctg,
		Name:      name,
		Type:      &typ,
		Value:     &value,
	}
	err := s.svc.UpdateEntryDefault(ctx, upd)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteEntryDefault(ctx context.Context, entType, ctg, name string) error {
	if entType == "" {
		return fmt.Errorf("entry default entry type not specified")
	}
	if ctg == "" {
		return fmt.Errorf("entry default category not specified")
	}
	if name == "" {
		return fmt.Errorf("entry default name not specified")
	}
	err := s.svc.DeleteEntryDefault(ctx, entType, ctg, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryProperties(ctx context.Context, path string) ([]*Property, error) {
	if path == "" {
		return nil, fmt.Errorf("entry path not specified")
	}
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
	if path == "" {
		return nil, fmt.Errorf("property path not specified")
	}
	if name == "" {
		return nil, fmt.Errorf("property name not specified")
	}
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
	if path == "" {
		return fmt.Errorf("property path not specified")
	}
	if name == "" {
		return fmt.Errorf("property name not specified")
	}
	if typ == "" {
		return fmt.Errorf("property type not specified")
	}
	if value == "" {
		return fmt.Errorf("property value not specified")
	}
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
	if path == "" {
		return fmt.Errorf("property path not specified")
	}
	if name == "" {
		return fmt.Errorf("property name not specified")
	}
	if path == "" {
		return fmt.Errorf("property value not specified")
	}
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
	if path == "" {
		return fmt.Errorf("property path not specified")
	}
	if name == "" {
		return fmt.Errorf("property name not specified")
	}
	err := s.svc.DeleteProperty(ctx, path, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryEnvirons(ctx context.Context, path string) ([]*Property, error) {
	if path == "" {
		return nil, fmt.Errorf("environ path not specified")
	}
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
	if path == "" {
		return nil, fmt.Errorf("environ path not specified")
	}
	if name == "" {
		return nil, fmt.Errorf("environ name not specified")
	}
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
	if path == "" {
		return fmt.Errorf("environ path not specified")
	}
	if name == "" {
		return fmt.Errorf("environ name not specified")
	}
	if typ == "" {
		return fmt.Errorf("environ type not specified")
	}
	if value == "" {
		return fmt.Errorf("environ value not specified")
	}
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
	if path == "" {
		return fmt.Errorf("environ path not specified")
	}
	if name == "" {
		return fmt.Errorf("environ name not specified")
	}
	if value == "" {
		return fmt.Errorf("environ value not specified")
	}
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
	if path == "" {
		return fmt.Errorf("environ path not specified")
	}
	if name == "" {
		return fmt.Errorf("environ name not specified")
	}
	err := s.svc.DeleteEnviron(ctx, path, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryAccessControls(ctx context.Context, path string) ([]*AccessControl, error) {
	if path == "" {
		return nil, fmt.Errorf("access control path not specified")
	}
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
	if path == "" {
		return fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return fmt.Errorf("accessor not specified")
	}
	if accessor_type == "" {
		return fmt.Errorf("accessor type not specified")
	}
	if mode == "" {
		return fmt.Errorf("access mode not specified")
	}
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
	if path == "" {
		return fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return fmt.Errorf("accessor not specified")
	}
	if mode == "" {
		return fmt.Errorf("access mode not specified")
	}
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

func (s *Server) DeleteAccessControl(ctx context.Context, path string, accessor string) error {
	if path == "" {
		return fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return fmt.Errorf("accessor not specified")
	}
	err := s.svc.DeleteAccessControl(ctx, path, accessor)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryLogs(ctx context.Context, path string) ([]*Log, error) {
	if path == "" {
		return nil, fmt.Errorf("log path not specified")
	}
	ls, err := s.svc.FindLogs(ctx, service.LogFinder{
		EntryPath: &path,
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

func (s *Server) GetLogs(ctx context.Context, path, ctg, name string) ([]*Log, error) {
	if path == "" {
		return nil, fmt.Errorf("log path not specified")
	}
	if ctg == "" {
		return nil, fmt.Errorf("log ctg not specified")
	}
	if name == "" {
		return nil, fmt.Errorf("log name not specified")
	}
	ls, err := s.svc.GetLogs(ctx, path, ctg, name)
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
	if user == "" {
		return nil, fmt.Errorf("user not specified")
	}
	su, err := s.svc.GetUser(ctx, user)
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
	if user == "" {
		return fmt.Errorf("user not specified")
	}
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

func (s *Server) GetGroup(ctx context.Context, group string) (*Group, error) {
	if group == "" {
		return nil, fmt.Errorf("group not specified")
	}
	sgroups, err := s.svc.FindGroups(ctx, service.GroupFinder{Name: &group})
	if err != nil {
		return nil, err
	}
	if len(sgroups) == 0 {
		return nil, fmt.Errorf("group not exist: %v", group)
	}
	sg := sgroups[0]
	g := &Group{
		ID:   sg.ID,
		Name: sg.Name,
	}
	return g, nil
}

func (s *Server) AddGroup(ctx context.Context, group string) error {
	if group == "" {
		return fmt.Errorf("group not specified")
	}
	g := &service.Group{Name: group}
	err := s.svc.AddGroup(ctx, g)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetGroup(ctx context.Context, groupID string, group string) error {
	if groupID == "" {
		return fmt.Errorf("group id not specified")
	}
	if group == "" {
		return fmt.Errorf("group not specified")
	}
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
	if group == "" {
		return nil, fmt.Errorf("group not specified")
	}
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
	if group == "" {
		return fmt.Errorf("group not specified")
	}
	if member == "" {
		return fmt.Errorf("member not specified")
	}
	m := &service.Member{Group: group, Member: member}
	err := s.svc.AddGroupMember(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteGroupMember(ctx context.Context, group, member string) error {
	if group == "" {
		return fmt.Errorf("group not specified")
	}
	if member == "" {
		return fmt.Errorf("member not specified")
	}
	err := s.svc.DeleteGroupMember(ctx, group, member)
	if err != nil {
		return err
	}
	return nil
}

// GetThumbnail gets a thumbnail image of a entry.
func (s *Server) GetThumbnail(ctx context.Context, path string) (*Thumbnail, error) {
	if path == "" {
		return nil, fmt.Errorf("thumbnail path not specified")
	}
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
	if path == "" {
		return fmt.Errorf("thumbnail path not specified")
	}
	if img == nil {
		return fmt.Errorf("thumbnail image not specified")
	}
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
	if path == "" {
		return fmt.Errorf("thumbnail path not specified")
	}
	if img == nil {
		return fmt.Errorf("thumbnail image not specified")
	}
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
	if path == "" {
		return fmt.Errorf("thumbnail path not specified")
	}
	err := s.svc.DeleteThumbnail(ctx, path)
	if err != nil {
		return err
	}
	return nil
}
