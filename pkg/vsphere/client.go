package vsphere

import (
	"context"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"

	"github.com/vmware/govmomi"
)

// Session contains vsphere connection and object data
type Session struct {
	Conn         *govmomi.Client
	Datacenter   *object.Datacenter
	Datastore    *object.Datastore
	Folder       *object.Folder
	ResourcePool *object.ResourcePool
	Network      object.NetworkReference
	Ctx          context.Context
}

// NewClient returns a new vsphere Session
func NewClient(ctx context.Context, server string, username string, password string) (*Session, error) {
	sm := new(Session)
	if !strings.HasPrefix(server, "https://") && !strings.HasPrefix(server, "http://") {
		server = "https://" + server
	}
	nonAuthURL, err := url.Parse(server)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse vCenter url %s", server)
	}
	if !strings.HasSuffix(nonAuthURL.Path, "sdk") {
		nonAuthURL.Path = nonAuthURL.Path + "sdk"
	}
	authenticatedURL, err := url.Parse(nonAuthURL.String())
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse vCenter url %s", nonAuthURL.String())
	}
	client, err := govmomi.NewClient(ctx, nonAuthURL, true)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create new vSphere client against %v", nonAuthURL.String())
	}
	authenticatedURL.User = url.UserPassword(username, password)
	if err = client.Login(ctx, authenticatedURL.User); err != nil {
		return nil, errors.Wrap(err, "unable to login to vSphere")
	}
	sm.Conn = client
	sm.Ctx = ctx

	return sm, nil
}

// GetDatacenterOrDefault returns the govmomi object for a datacenter
func (s *Session) GetDatacenterOrDefault(name string) (*object.Datacenter, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	datacenter, err := finder.DatacenterOrDefault(s.Ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding datacenter %s", name)
	}
	return datacenter, err
}

// GetDatastoreOrDefault returns the govmomi object for a datastore
func (s *Session) GetDatastoreOrDefault(name string) (*object.Datastore, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	datastore, err := finder.DatastoreOrDefault(s.Ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding datastore %s", name)
	}
	return datastore, err
}

// GetNetworkOrDefault returns the govmomi object for a network
func (s *Session) GetNetworkOrDefault(name string) (object.NetworkReference, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	network, err := finder.NetworkOrDefault(s.Ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding network %s", name)
	}
	return network, err
}

// GetResourcePoolOrDefault returns the govmomi object for a resource pool
func (s *Session) GetResourcePoolOrDefault(name string) (*object.ResourcePool, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	resourcePool, err := finder.ResourcePoolOrDefault(s.Ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding resource pool %s", name)
	}
	return resourcePool, err
}

// GetVM returns the govmomi object for a virtual machine
func (s *Session) GetVM(name string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	vm, err := finder.VirtualMachine(s.Ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding VM %s", name)
	}
	return vm, err
}

// GetFolderOrDefault returns a single folder object
func (s *Session) GetFolderOrDefault(name string) (*object.Folder, error) {
	finder := find.NewFinder(s.Conn.Client, true)
	finder.SetDatacenter(s.Datacenter)
	desiredFolder, err := finder.FolderOrDefault(s.Ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding folder %s", name)
	}
	return desiredFolder, err
}
