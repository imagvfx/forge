package forge

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"sort"
	"strings"

	"golang.org/x/image/draw"
)

type Server struct {
	svc Service
	cfg *Config
}

func NewServer(svc Service, cfg *Config) *Server {
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
	ent, err := s.svc.GetEntry(ctx, path)
	if err != nil {
		return nil, err
	}
	return ent, nil
}

func (s *Server) SubEntries(ctx context.Context, path string) ([]*Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("entry path not specified")
	}
	ents, err := s.svc.FindEntries(ctx, EntryFinder{
		ParentPath: &path,
	})
	if err != nil {
		return nil, err
	}
	return ents, nil
}

// ParentEntries returns parent entries from root to a given entry (but without the entry).
func (s *Server) ParentEntries(ctx context.Context, path string) ([]*Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("entry path not specified")
	}
	ents, err := s.svc.FindEntries(ctx, EntryFinder{
		ChildPath: &path,
	})
	sort.Slice(ents, func(i, j int) bool {
		return strings.Compare(ents[i].Path, ents[j].Path) < 0
	})
	if err != nil {
		return nil, err
	}
	return ents, nil
}

func (s *Server) SearchEntries(ctx context.Context, path, query string) ([]*Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("entry path not specified")
	}
	ents, err := s.svc.SearchEntries(ctx, EntrySearcher{
		SearchRoot: path,
		Keywords:   strings.Fields(query),
	})
	if err != nil {
		return nil, err
	}
	return ents, nil
}

func (s *Server) CountAllSubEntries(ctx context.Context, path string) (int, error) {
	if path == "" {
		return 0, fmt.Errorf("entry path not specified")
	}
	return s.svc.CountAllSubEntries(ctx, path)
}

func (s *Server) AddEntry(ctx context.Context, path, typ string) error {
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	e := &Entry{
		Path: path,
		Type: typ,
	}
	err := s.svc.AddEntry(ctx, e)
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
	return nil
}

func (s *Server) ArchiveEntry(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	err := s.svc.ArchiveEntry(ctx, path)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UnarchiveEntry(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	err := s.svc.UnarchiveEntry(ctx, path)
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

func (s *Server) DeleteEntryRecursive(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("entry path not specified")
	}
	err := s.svc.DeleteEntryRecursive(ctx, path)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) FindEntryTypes(ctx context.Context) ([]string, error) {
	names, err := s.svc.FindEntryTypes(ctx)
	if err != nil {
		return nil, err
	}
	return names, nil
}

func (s *Server) FindBaseEntryTypes(ctx context.Context) ([]string, error) {
	names, err := s.svc.FindBaseEntryTypes(ctx)
	if err != nil {
		return nil, err
	}
	return names, nil
}

func (s *Server) FindOverrideEntryTypes(ctx context.Context) ([]string, error) {
	names, err := s.svc.FindOverrideEntryTypes(ctx)
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

func (s *Server) Defaults(ctx context.Context, entType string) ([]*Default, error) {
	if entType == "" {
		return nil, fmt.Errorf("entry type name not specified")
	}
	defaults, err := s.svc.FindDefaults(ctx, DefaultFinder{EntryType: &entType})
	if err != nil {
		return nil, err
	}
	return defaults, nil
}

func (s *Server) AddDefault(ctx context.Context, entType, ctg, name, typ, value string) error {
	if entType == "" {
		return fmt.Errorf("default entry type not specified")
	}
	if ctg == "" {
		return fmt.Errorf("default category not specified")
	}
	if name == "" {
		return fmt.Errorf("default name not specified")
	}
	if typ == "" {
		return fmt.Errorf("default type not specified")
	}
	d := &Default{
		EntryType: entType,
		Category:  ctg,
		Name:      name,
		Type:      typ,
		Value:     value,
	}
	err := s.svc.AddDefault(ctx, d)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateDefault(ctx context.Context, entType, ctg, name, typ, value string) error {
	if entType == "" {
		return fmt.Errorf("default entry type not specified")
	}
	if ctg == "" {
		return fmt.Errorf("default category not specified")
	}
	if name == "" {
		return fmt.Errorf("default name not specified")
	}
	if typ == "" {
		return fmt.Errorf("default type not specified")
	}
	upd := DefaultUpdater{
		EntryType: entType,
		Category:  ctg,
		Name:      name,
		Type:      &typ,
		Value:     &value,
	}
	err := s.svc.UpdateDefault(ctx, upd)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteDefault(ctx context.Context, entType, ctg, name string) error {
	if entType == "" {
		return fmt.Errorf("default entry type not specified")
	}
	if ctg == "" {
		return fmt.Errorf("default category not specified")
	}
	if name == "" {
		return fmt.Errorf("default name not specified")
	}
	err := s.svc.DeleteDefault(ctx, entType, ctg, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Globals(ctx context.Context, entType string) ([]*Global, error) {
	if entType == "" {
		return nil, fmt.Errorf("entry type name not specified")
	}
	globals, err := s.svc.FindGlobals(ctx, GlobalFinder{EntryType: &entType})
	if err != nil {
		return nil, err
	}
	return globals, nil
}

func (s *Server) GetGlobal(ctx context.Context, entType, name string) (*Global, error) {
	if entType == "" {
		return nil, fmt.Errorf("entry type name not specified")
	}
	g, err := s.svc.GetGlobal(ctx, entType, name)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Server) AddGlobal(ctx context.Context, entType, name, typ, value string) error {
	if entType == "" {
		return fmt.Errorf("global entry type not specified")
	}
	if name == "" {
		return fmt.Errorf("global name not specified")
	}
	if typ == "" {
		return fmt.Errorf("global type not specified")
	}
	sg := &Global{
		EntryType: entType,
		Name:      name,
		Type:      typ,
		Value:     value,
	}
	err := s.svc.AddGlobal(ctx, sg)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateGlobal(ctx context.Context, entType, name, typ, value string) error {
	if entType == "" {
		return fmt.Errorf("global entry type not specified")
	}
	if name == "" {
		return fmt.Errorf("global name not specified")
	}
	if typ == "" {
		return fmt.Errorf("global type not specified")
	}
	upd := GlobalUpdater{
		EntryType: entType,
		Name:      name,
		Type:      &typ,
		Value:     &value,
	}
	err := s.svc.UpdateGlobal(ctx, upd)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteGlobal(ctx context.Context, entType, name string) error {
	if entType == "" {
		return fmt.Errorf("global entry type not specified")
	}
	if name == "" {
		return fmt.Errorf("global name not specified")
	}
	err := s.svc.DeleteGlobal(ctx, entType, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) EntryProperties(ctx context.Context, path string) ([]*Property, error) {
	if path == "" {
		return nil, fmt.Errorf("entry path not specified")
	}
	props, err := s.svc.EntryProperties(ctx, path)
	if err != nil {
		return nil, err
	}
	sort.Slice(props, func(i, j int) bool {
		a := props[i]
		b := props[j]
		return a.Name <= b.Name
	})
	return props, nil
}

func (s *Server) GetProperty(ctx context.Context, path string, name string) (*Property, error) {
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
	return p, nil
}

func (s *Server) UpdateProperty(ctx context.Context, path string, name, value string) error {
	if path == "" {
		return fmt.Errorf("property path not specified")
	}
	if name == "" {
		return fmt.Errorf("property name not specified")
	}
	err := s.svc.UpdateProperty(ctx, PropertyUpdater{
		EntryPath: path,
		Name:      name,
		Value:     &value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateProperties(ctx context.Context, upds []PropertyUpdater) error {
	// Note it directly uses UpdateProperty unlike others methods here.
	// I will change to use service instead of Server in the future.
	return s.svc.UpdateProperties(ctx, upds)
}

func (s *Server) EntryEnvirons(ctx context.Context, path string) ([]*Property, error) {
	if path == "" {
		return nil, fmt.Errorf("environ path not specified")
	}
	envs, err := s.svc.EntryEnvirons(ctx, path)
	if err != nil {
		return nil, err
	}
	sort.Slice(envs, func(i, j int) bool {
		a := envs[i]
		b := envs[j]
		cmp := strings.Compare(a.EntryPath, b.EntryPath)
		if cmp != 0 {
			return cmp < 0
		}
		return a.Name <= b.Name
	})
	return envs, nil
}

func (s *Server) GetEnviron(ctx context.Context, path, name string) (*Property, error) {
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
	return e, nil
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
	env := &Property{
		EntryPath: path,
		Name:      name,
		Type:      typ,
		Value:     value,
	}
	err := s.svc.AddEnviron(ctx, env)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateEnviron(ctx context.Context, path string, name, value string) error {
	if path == "" {
		return fmt.Errorf("environ path not specified")
	}
	if name == "" {
		return fmt.Errorf("environ name not specified")
	}
	err := s.svc.UpdateEnviron(ctx, PropertyUpdater{
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

func (s *Server) EntryAccessList(ctx context.Context, path string) ([]*Access, error) {
	if path == "" {
		return nil, fmt.Errorf("access control path not specified")
	}
	acls, err := s.svc.EntryAccessList(ctx, path)
	if err != nil {
		return nil, err
	}
	sort.Slice(acls, func(i, j int) bool {
		a := acls[i]
		b := acls[j]
		cmp := strings.Compare(a.EntryPath, b.EntryPath)
		if cmp != 0 {
			return cmp < 0
		}
		return a.Name <= b.Name
	})
	return acls, nil
}

func (s *Server) GetAccess(ctx context.Context, path string, accessor string) (*Access, error) {
	if path == "" {
		return nil, fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return nil, fmt.Errorf("accessor not specified")
	}
	acl, err := s.svc.GetAccess(ctx, path, accessor)
	if err != nil {
		return nil, err
	}
	return acl, nil
}

func (s *Server) AddAccess(ctx context.Context, path string, accessor, mode string) error {
	if path == "" {
		return fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return fmt.Errorf("accessor not specified")
	}
	if mode == "" {
		return fmt.Errorf("access mode not specified")
	}
	switch mode {
	case "r":
	case "rw":
	default:
		return fmt.Errorf("unknown access type")
	}
	ac := &Access{
		EntryPath: path,
		Name:      accessor,
		Value:     mode,
	}
	err := s.svc.AddAccess(ctx, ac)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateAccess(ctx context.Context, path, accessor, mode string) error {
	if path == "" {
		return fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return fmt.Errorf("accessor not specified")
	}
	if mode == "" {
		return fmt.Errorf("access mode not specified")
	}
	switch mode {
	case "r":
	case "rw":
	default:
		return fmt.Errorf("unknown access type")
	}
	ac := AccessUpdater{
		EntryPath: path,
		Name:      accessor,
		Value:     &mode,
	}
	err := s.svc.UpdateAccess(ctx, ac)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteAccess(ctx context.Context, path string, accessor string) error {
	if path == "" {
		return fmt.Errorf("access control path not specified")
	}
	if accessor == "" {
		return fmt.Errorf("accessor not specified")
	}
	err := s.svc.DeleteAccess(ctx, path, accessor)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) IsAdmin(ctx context.Context, user string) (bool, error) {
	if user == "" {
		return false, fmt.Errorf("user not specified")
	}
	admin, err := s.svc.IsAdmin(ctx, user)
	if err != nil {
		return false, err
	}
	return admin, nil
}

func (s *Server) EntryLogs(ctx context.Context, path string) ([]*Log, error) {
	if path == "" {
		return nil, fmt.Errorf("log path not specified")
	}
	logs, err := s.svc.FindLogs(ctx, LogFinder{
		EntryPath: &path,
	})
	if err != nil {
		return nil, err
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
	logs, err := s.svc.GetLogs(ctx, path, ctg, name)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *Server) Users(ctx context.Context) ([]*User, error) {
	users, err := s.svc.FindUsers(ctx, UserFinder{})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Server) GetUser(ctx context.Context, user string) (*User, error) {
	if user == "" {
		return nil, fmt.Errorf("user not specified")
	}
	u, err := s.svc.GetUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Server) AddUser(ctx context.Context, u *User) error {
	if u == nil {
		return fmt.Errorf("nil user")
	}
	if u.Name == "" {
		return fmt.Errorf("user not specified")
	}
	err := s.svc.AddUser(ctx, u)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) UpdateUserCalled(ctx context.Context, user, called string) error {
	if user == "" {
		return fmt.Errorf("user not specified")
	}
	err := s.svc.UpdateUserCalled(ctx, user, called)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetUserSetting(ctx context.Context, user string) (*UserSetting, error) {
	if user == "" {
		return nil, fmt.Errorf("user not specified")
	}
	us, err := s.svc.GetUserSetting(ctx, user)
	if err != nil {
		return nil, err
	}
	return us, nil
}

func (s *Server) UpdateUserSetting(ctx context.Context, user, key string, value any) error {
	upd := UserSettingUpdater{
		User:  user,
		Key:   key,
		Value: value,
	}
	err := s.svc.UpdateUserSetting(ctx, upd)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) AddUserDataSection(ctx context.Context, user, section string) error {
	err := s.svc.AddUserDataSection(ctx, user, section)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetUserDataSection(ctx context.Context, user, section string) (*UserDataSection, error) {
	sec, err := s.svc.GetUserDataSection(ctx, user, section)
	if err != nil {
		return nil, err
	}
	return sec, nil
}

func (s *Server) DeleteUserDataSection(ctx context.Context, user, section string) error {
	err := s.svc.DeleteUserDataSection(ctx, user, section)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) FindUserData(ctx context.Context, find UserDataFinder) ([]*UserDataSection, error) {
	data, err := s.svc.FindUserData(ctx, find)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Server) GetUserData(ctx context.Context, user, section, key string) (string, error) {
	value, err := s.svc.GetUserData(ctx, user, section, key)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Server) SetUserData(ctx context.Context, user, section, key, value string) error {
	err := s.svc.SetUserData(ctx, user, section, key, value)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) DeleteUserData(ctx context.Context, user, section, key string) error {
	err := s.svc.DeleteUserData(ctx, user, section, key)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) FindAllGroups(ctx context.Context) ([]*Group, error) {
	groups, err := s.svc.FindGroups(ctx, GroupFinder{})
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (s *Server) GetGroup(ctx context.Context, group string) (*Group, error) {
	if group == "" {
		return nil, fmt.Errorf("group not specified")
	}
	groups, err := s.svc.FindGroups(ctx, GroupFinder{Name: &group})
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("group not exist: %v", group)
	}
	g := groups[0]
	return g, nil
}

func (s *Server) AddGroup(ctx context.Context, g *Group) error {
	if g == nil {
		return fmt.Errorf("nil group")
	}
	if g.Name == "" {
		return fmt.Errorf("group not specified")
	}
	sg := &Group{
		Name:   g.Name,
		Called: g.Called,
	}
	err := s.svc.AddGroup(ctx, sg)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) RenameGroup(ctx context.Context, name string, newName string) error {
	if name == "" {
		return fmt.Errorf("name of group not specified")
	}
	if newName == "" {
		return fmt.Errorf("new name of group not specified")
	}
	g := GroupUpdater{Name: name, NewName: &newName}
	err := s.svc.UpdateGroup(ctx, g)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) FindGroupMembers(ctx context.Context, group string) ([]*Member, error) {
	if group == "" {
		return nil, fmt.Errorf("group not specified")
	}
	members, err := s.svc.FindGroupMembers(ctx, MemberFinder{Group: group})
	if err != nil {
		return nil, err
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
	m := &Member{Group: group, Member: member}
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
	thumb, err := s.svc.GetThumbnail(ctx, path)
	if err != nil {
		return nil, err
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
	err = s.svc.AddThumbnail(ctx, &Thumbnail{
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
	err = s.svc.UpdateThumbnail(ctx, ThumbnailUpdater{
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
