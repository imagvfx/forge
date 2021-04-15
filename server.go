package forge

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/imagvfx/forge/service"
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

func (s *Server) GetEntry(user, path string) (*Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("path emtpy")
	}
	es, err := s.svc.FindEntries(service.EntryFinder{
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	if len(es) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	if len(es) != 1 {
		return nil, fmt.Errorf("got more than 1 entry")
	}
	e := es[0]
	parentID := -1
	if e.ParentID != nil {
		parentID = *e.ParentID
	}
	ent := &Entry{
		srv:      s,
		id:       e.ID,
		parentID: parentID,
		path:     e.Path,
		typ:      e.Type,
	}
	return ent, nil
}

func (s *Server) getEntry(user string, id int) (*Entry, error) {
	e, err := s.svc.GetEntry(id)
	if err != nil {
		return nil, err
	}
	parentID := -1
	if e.ParentID != nil {
		parentID = *e.ParentID
	}
	ent := &Entry{
		srv:      s,
		id:       e.ID,
		parentID: parentID,
		path:     e.Path,
		typ:      e.Type,
	}
	return ent, nil
}

func (s *Server) SubEntries(user string, path string) ([]*Entry, error) {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return nil, err
	}
	es, err := s.svc.FindEntries(service.EntryFinder{
		ParentID: &ent.id,
	})
	if err != nil {
		return nil, err
	}
	ents := make([]*Entry, 0)
	for _, e := range es {
		parentID := -1
		if e.ParentID != nil {
			parentID = *e.ParentID
		}
		ent := &Entry{
			srv:      s,
			id:       e.ID,
			parentID: parentID,
			path:     e.Path,
			typ:      e.Type,
		}
		ents = append(ents, ent)
	}
	return ents, nil
}

func (s *Server) AddEntry(user, path, typ string) error {
	path = filepath.ToSlash(path)
	parent := filepath.Dir(path)
	p, err := s.GetEntry(user, parent)
	if err != nil {
		return fmt.Errorf("error on parent check: %v", err)
	}
	allow := false
	subtyps := s.cfg.Struct[p.Type()].SubEntryTypes
	for _, subtyp := range subtyps {
		if subtyp == typ {
			allow = true
			break
		}
	}
	if !allow {
		return fmt.Errorf("cannot create a child of type %q from %q", typ, p.Type())
	}
	e := &service.Entry{
		ParentID: &p.id,
		Path:     path,
		Type:     typ,
	}
	props := make([]*service.Property, 0)
	for _, ktv := range s.cfg.Struct[typ].Properties {
		p := &Property{
			srv:       s,
			entryPath: path,
			name:      ktv.Key,
			typ:       ktv.Type,
			value:     ktv.Value,
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
			srv:       s,
			entryPath: path,
			name:      ktv.Key,
			typ:       ktv.Type,
			value:     ktv.Value,
		}
		err := e.Validate()
		if err != nil {
			return err
		}
		envs = append(envs, e.ServiceProperty())
	}
	err = s.svc.AddEntry(user, e, props, envs)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryProperties(user, path string) ([]*Property, error) {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return nil, err
	}
	ps, err := s.svc.FindProperties(service.PropertyFinder{
		EntryID: ent.id,
	})
	if err != nil {
		return nil, err
	}
	props := make([]*Property, 0)
	for _, p := range ps {
		prop := &Property{
			srv:       s,
			id:        p.ID,
			entryID:   p.EntryID,
			entryPath: p.EntryPath,
			name:      p.Name,
			typ:       p.Type,
			value:     p.Value,
		}
		props = append(props, prop)
	}
	return props, nil
}

func (s *Server) getProperty(user string, ent int, name string) (*Property, error) {
	ps, err := s.svc.FindProperties(service.PropertyFinder{
		EntryID: ent,
		Name:    &name,
	})
	if err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	if len(ps) != 1 {
		return nil, fmt.Errorf("got more than 1 property")
	}
	p := ps[0]
	prop := &Property{
		srv:       s,
		id:        p.ID,
		entryID:   p.EntryID,
		entryPath: p.EntryPath,
		name:      p.Name,
		typ:       p.Type,
		value:     p.Value,
	}
	return prop, nil
}

func (s *Server) AddProperty(user, path string, name, typ, value string) error {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return err
	}
	env := &Property{
		srv:       s,
		entryID:   ent.id,
		entryPath: ent.path,
		name:      name,
		typ:       typ,
		value:     value,
	}
	err = env.Validate()
	if err != nil {
		return err
	}
	err = s.svc.AddProperty(user, env.ServiceProperty())
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetProperty(user, path string, name, value string) error {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return err
	}
	prop, err := s.getProperty(user, ent.id, name)
	if err != nil {
		return err
	}
	prop.value = value // validate the given value
	err = prop.Validate()
	if err != nil {
		return err
	}
	err = s.svc.UpdateProperty(user, service.PropertyUpdater{
		ID:    prop.id,
		Value: &value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryEnvirons(user, path string) ([]*Property, error) {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return nil, err
	}
	ps, err := s.svc.FindEnvirons(service.PropertyFinder{
		EntryID: ent.id,
	})
	if err != nil {
		return nil, err
	}
	props := make([]*Property, 0)
	for _, p := range ps {
		prop := &Property{
			srv:       s,
			id:        p.ID,
			entryID:   p.EntryID,
			entryPath: p.EntryPath,
			name:      p.Name,
			typ:       p.Type,
			value:     p.Value,
		}
		props = append(props, prop)
	}
	return props, nil
}

func (s *Server) getEnviron(user string, ent int, name string) (*Property, error) {
	es, err := s.svc.FindEnvirons(service.PropertyFinder{
		EntryID: ent,
		Name:    &name,
	})
	if err != nil {
		return nil, err
	}
	if len(es) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	if len(es) != 1 {
		return nil, fmt.Errorf("got more than 1 environ")
	}
	e := es[0]
	env := &Property{
		srv:       s,
		id:        e.ID,
		entryID:   e.EntryID,
		entryPath: e.EntryPath,
		name:      e.Name,
		typ:       e.Type,
		value:     e.Value,
	}
	return env, nil
}

func (s *Server) AddEnviron(user, path string, name, typ, value string) error {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return err
	}
	env := &Property{
		srv:       s,
		entryID:   ent.id,
		entryPath: ent.path,
		name:      name,
		typ:       typ,
		value:     value,
	}
	err = env.Validate()
	if err != nil {
		return err
	}
	err = s.svc.AddEnviron(user, env.ServiceProperty())
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetEnviron(user, path string, name, value string) error {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return err
	}
	env, err := s.getEnviron(user, ent.id, name)
	if err != nil {
		return err
	}
	env.value = value // validate the given value
	err = env.Validate()
	if err != nil {
		return err
	}
	err = s.svc.UpdateEnviron(user, service.PropertyUpdater{
		ID:    env.id,
		Value: &value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryAccessControls(user, path string) ([]*AccessControl, error) {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return nil, err
	}
	as, err := s.svc.FindAccessControls(service.AccessControlFinder{
		EntryID: ent.id,
	})
	if err != nil {
		return nil, err
	}
	acs := make([]*AccessControl, 0, len(as))
	for _, a := range as {
		ac := &AccessControl{
			ID:           a.ID,
			EntryID:      a.EntryID,
			EntryPath:    a.EntryPath,
			Accessor:     a.Accessor,
			AccessorType: AccessorType(a.AccessorType),
			Mode:         AccessMode(a.Mode),
		}
		acs = append(acs, ac)
	}
	return acs, nil
}

func (s *Server) AddAccessControl(user, path string, accessor, accessor_type, mode string) error {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return err
	}
	ac := &service.AccessControl{
		EntryID: ent.id,
	}
	switch accessor_type {
	case "user":
		u, err := s.GetUser(accessor)
		if err != nil {
			return err
		}
		ac.UserID = &u.ID
	case "group":
		g, err := s.GetGroup(accessor)
		if err != nil {
			return err
		}
		ac.GroupID = &g.ID
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
	err = s.svc.AddAccessControl(user, ac)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetAccessControl(user string, accessID string, mode string) error {
	id, err := strconv.Atoi(accessID)
	if err != nil {
		return fmt.Errorf("invalid access id: %v", accessID)
	}
	ac := service.AccessControlUpdater{
		ID: id,
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
	err = s.svc.UpdateAccessControl(user, ac)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryLogs(user, path string) ([]*Log, error) {
	ent, err := s.GetEntry(user, path)
	if err != nil {
		return nil, err
	}
	ls, err := s.svc.FindLogs(service.LogFinder{
		EntryID: ent.id,
	})
	if err != nil {
		return nil, err
	}
	logs := make([]*Log, 0)
	for _, l := range ls {
		log := &Log{
			ID:       l.ID,
			EntryID:  l.EntryID,
			User:     l.User,
			Action:   l.Action,
			Category: l.Category,
			Name:     l.Name,
			Type:     l.Type,
			Value:    l.Value,
			When:     l.When,
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (s *Server) GetUser(user string) (*User, error) {
	su, err := s.svc.GetUserByUser(user)
	if err != nil {
		err = fromServiceError(err)
		return nil, err
	}
	u := &User{
		ID:   su.ID,
		User: su.User,
		Name: su.Name,
	}
	return u, nil
}

func (s *Server) AddUser(user string) error {
	u := &service.User{User: user}
	err := s.svc.AddUser(u)
	if err != nil {
		err = fromServiceError(err)
		return err
	}
	return nil
}

func (s *Server) FindAllGroups() ([]*Group, error) {
	sgroups, err := s.svc.FindGroups(service.GroupFinder{})
	if err != nil {
		err = fromServiceError(err)
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

func (s *Server) GetGroup(name string) (*Group, error) {
	sgroups, err := s.svc.FindGroups(service.GroupFinder{Name: &name})
	if err != nil {
		err = fromServiceError(err)
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

func (s *Server) AddGroup(user, group string) error {
	g := &service.Group{Name: group}
	err := s.svc.AddGroup(user, g)
	if err != nil {
		err = fromServiceError(err)
		return err
	}
	return nil
}

func (s *Server) SetGroup(user string, groupID string, group string) error {
	id, err := strconv.Atoi(groupID)
	if err != nil {
		return fmt.Errorf("invalid group id: %v", groupID)
	}
	g := service.GroupUpdater{ID: id, Name: &group}
	err = s.svc.UpdateGroup(user, g)
	if err != nil {
		err = fromServiceError(err)
		return err
	}
	return nil
}

func (s *Server) FindGroupMembers(group string) ([]*Member, error) {
	g, err := s.GetGroup(group)
	if err != nil {
		return nil, err
	}
	svcMembers, err := s.svc.FindGroupMembers(service.MemberFinder{GroupID: &g.ID})
	if err != nil {
		err = fromServiceError(err)
		return nil, err
	}
	members := make([]*Member, 0, len(svcMembers))
	for _, sm := range svcMembers {
		m := &Member{
			ID:      sm.ID,
			GroupID: sm.GroupID,
			Group:   sm.Group,
			UserID:  sm.UserID,
			User:    sm.User,
		}
		members = append(members, m)
	}
	return members, nil
}

func (s *Server) AddGroupMember(user, group, member string) error {
	g, err := s.GetGroup(group)
	if err != nil {
		return err
	}
	u, err := s.GetUser(member)
	if err != nil {
		fmt.Println("here?")
		return err
	}
	m := &service.Member{GroupID: g.ID, UserID: u.ID}
	err = s.svc.AddGroupMember(user, m)
	if err != nil {
		err = fromServiceError(err)
		return err
	}
	return nil
}

func (s *Server) DeleteGroupMember(user string, memberID string) error {
	id, err := strconv.Atoi(memberID)
	if err != nil {
		return fmt.Errorf("invalid member id: %v", memberID)
	}
	err = s.svc.DeleteGroupMember(user, id)
	if err != nil {
		err = fromServiceError(err)
		return err
	}
	return nil
}
