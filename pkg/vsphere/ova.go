package vsphere

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// DeployInfo is data for a deployed OVA
type DeployInfo struct {
	TemplateName  string
	VMObject      *object.VirtualMachine
	AlreadyExists bool
}

// DeployOVATemplates deploys multiple OVAs asynchronously
func (s *Session) DeployOVATemplates(templatePaths ...string) (map[string]DeployInfo, error) {
	templatePaths = sliceDedup(templatePaths)
	numOVAs := len(templatePaths)
	result := make(map[string]DeployInfo, numOVAs)
	resultMutex := sync.Mutex{}

	var g errgroup.Group
	for _, template := range templatePaths {
		if template == "" {
			continue
		}
		template := template
		g.Go(func() error {
			r, err := s.DeployOVATemplate(template)
			if err != nil {
				return err
			}
			resultMutex.Lock()
			result[template] = r
			resultMutex.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return result, err
	}
	return result, nil

}

// DeployOVATemplate uploads ova and makes it a template
func (s *Session) DeployOVATemplate(templatePath string) (DeployInfo, error) {
	// TODO validate session has no nil values
	var result DeployInfo
	templateName := strings.TrimSuffix(path.Base(templatePath), ".ova")
	result.TemplateName = templateName
	ctx := context.TODO()
	vSphereClient := s.Conn
	finder := find.NewFinder(vSphereClient.Client, true)
	finder.SetDatacenter(s.Datacenter)
	foundTemplate, err := finder.VirtualMachine(ctx, templateName)
	if err == nil {
		result.AlreadyExists = true
		result.VMObject = foundTemplate
		return result, nil
	}

	networks := []types.OvfNetworkMapping{
		{
			Name:    "nic0",
			Network: s.Network.Reference(),
		},
	}

	cisp := types.OvfCreateImportSpecParams{
		DiskProvisioning:   "thin",
		EntityName:         templateName,
		IpAllocationPolicy: "dhcpPolicy",
		IpProtocol:         "IPv4",
		OvfManagerCommonParams: types.OvfManagerCommonParams{
			DeploymentOption: "",
			Locale:           "US"},
		PropertyMapping: nil,
		// We need to give it a network spec, even though we don't need/want networks since we overwrite them at clone time.
		// govmomi complains that the network spec is missing otherwise (can't create the import spec).
		NetworkMapping: networks,
	}

	vm, err := createVirtualMachine(ctx, cisp, templatePath, s)
	if err != nil {
		return result, errors.WithMessagef(err, "unable to create virtual machine from %v", templateName)
	}

	// Remove NICs from virtual machine before marking it as template

	/*
		if err := removeNICs(ctx, vm); err != nil {
			return nil, errors.WithMessagef(err, "unable to remove NICs from template %v", templateName)
		}
	*/

	if err := vm.MarkAsTemplate(ctx); err != nil {
		return result, errors.Wrapf(err, "unable to mark virtual machine as a template %v", templateName)
	}
	result.VMObject = vm

	return result, nil
}

func createVirtualMachine(ctx context.Context, cisp types.OvfCreateImportSpecParams, ovaPath string, vSphere *Session) (*object.VirtualMachine, error) {
	vSphereClient := vSphere.Conn

	ovaClient, err := newOVA(vSphereClient, ovaPath)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to create ova client")
	}

	spec, err := ovaClient.getImportSpec(ctx, ovaPath, vSphere.ResourcePool, vSphere.Datastore, cisp)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to create import spec for template (%s)", ovaPath)
	}
	if spec.Error != nil {
		return nil, errors.New(fmt.Sprintf("unable to create import spec for template, %v", spec.Error))
	}
	switch s := spec.ImportSpec.(type) {
	case *types.VirtualMachineImportSpec:
		if s.ConfigSpec.VAppConfig != nil {
			if s.ConfigSpec.VAppConfig.GetVmConfigSpec().OvfSection != nil {
				s.ConfigSpec.VAppConfig.GetVmConfigSpec().OvfSection = nil
			}
		}
	}

	lease, err := vSphere.ResourcePool.ImportVApp(ctx, spec.ImportSpec, vSphere.Folder, nil)
	if err != nil {
		return nil, errors.Wrap(err, "1 unable to import the template")
	}

	info, err := lease.Wait(ctx, spec.FileItem)
	if err != nil {
		return nil, errors.Wrap(err, "2 unable to import the template")
	}

	u := lease.StartUpdater(ctx, info)
	defer u.Done()

	for _, i := range info.Items {
		err = ovaClient.upload(ctx, lease, i, ovaPath)
		if err != nil {
			return nil, errors.WithMessagef(err, "3 unable to import the template")
		}
	}

	err = lease.Complete(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "4 unable to import the template")
	}

	moref := &info.Entity

	vm := object.NewVirtualMachine(vSphereClient.Client, *moref)

	return vm, nil
}

func openLocal(path string) (io.ReadCloser, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, errors.Wrap(err, "error opening local file")
	}

	s, err := f.Stat()
	if err != nil {
		return nil, 0, errors.Wrap(err, "error stat on local file")
	}

	return f, s.Size(), nil
}

func isRemotePath(path string) bool {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return true
	}
	return false
}

type tapeArchiveEntry struct {
	io.Reader
	f io.Closer
}

func (t *tapeArchiveEntry) Close() error {
	return t.f.Close()
}

type ova interface {
	upload(ctx context.Context, lease *nfc.Lease, item nfc.FileItem, ovaPath string) error
	getImportSpec(ctx context.Context, ovaPath string, resourcePool mo.Reference, datastore mo.Reference, cisp types.OvfCreateImportSpecParams) (*types.OvfCreateImportSpecResult, error)
}

type handler struct {
	client *govmomi.Client
}

// newOVA returns a new ova client
func newOVA(client *govmomi.Client, basePath string) (ova, error) {
	_, err := url.Parse(basePath)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing url %s", basePath)
	}

	return &handler{
		client: client,
	}, nil
}

func (h *handler) getImportSpec(ctx context.Context, ovaPath string, resourcePool mo.Reference, datastore mo.Reference, cisp types.OvfCreateImportSpecParams) (*types.OvfCreateImportSpecResult, error) {
	m := ovf.NewManager(h.client.Client)

	o, err := h.readOvf("*.ovf", ovaPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to read OVF file from %s", ovaPath)
	}
	g := types.OvfParseDescriptorParams{}
	p, err := m.ParseDescriptor(ctx, string(o), g)
	if err == nil {
		cisp.NetworkMapping[0].Name = p.Network[0].Name
	}

	return m.CreateImportSpec(ctx, string(o), resourcePool, datastore, cisp)
}

func (h *handler) upload(ctx context.Context, lease *nfc.Lease, item nfc.FileItem, ovaPath string) error {
	file := item.Path

	f, size, err := h.openOva(file, ovaPath)
	if err != nil {
		return errors.WithMessage(err, "unable to open OVA")
	}
	defer f.Close()

	opts := soap.Upload{
		ContentLength: size,
	}

	return lease.Upload(ctx, item, f, opts)
}

func (h *handler) readOvf(name string, ovaPath string) ([]byte, error) {
	tarReader, _, err := h.openOva(name, ovaPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to open OVA file %s", ovaPath)
	}
	defer tarReader.Close()

	return ioutil.ReadAll(tarReader)
}

func (h *handler) openOva(name string, ovaPath string) (io.ReadCloser, int64, error) {
	f, _, err := h.openFile(ovaPath)
	if err != nil {
		return nil, 0, errors.WithMessagef(err, "error opening ova path %v", ovaPath)
	}

	tarReader := tar.NewReader(f)

	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, errors.Wrap(err, "error reading ova")
		}

		matched, err := path.Match(name, path.Base(h.Name))
		if err != nil {
			return nil, 0, errors.Wrap(err, "error reading ova")
		}

		if matched {
			return &tapeArchiveEntry{tarReader, f}, h.Size, nil
		}
	}

	_ = f.Close()

	return nil, 0, errors.Wrap(os.ErrNotExist, "error opening ova")
}

func (h *handler) openFile(path string) (io.ReadCloser, int64, error) {
	if isRemotePath(path) {
		return h.openRemote(path)
	}
	return openLocal(path)
}

func (h *handler) openRemote(link string) (io.ReadCloser, int64, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Error parsing url %s", link)
	}
	rdr, num, err := h.client.Client.Download(context.TODO(), u, &soap.DefaultDownload)
	return rdr, num, errors.Wrapf(err, "error downloading %v", u)

}

func removeNICs(ctx context.Context, vm *object.VirtualMachine) error {
	vmProps, err := getProperties(ctx, vm)
	if err != nil {
		return errors.Wrap(err, "unable to get virtual machine properties")
	}
	virtualDeviceList := object.VirtualDeviceList(vmProps.Config.Hardware.Device)
	nicDevices := virtualDeviceList.SelectByType((*types.VirtualEthernetCard)(nil))
	if len(nicDevices) == 0 {
		return nil
	}

	var deviceConfigSpecs []types.BaseVirtualDeviceConfigSpec
	for _, dev := range nicDevices {
		bvEthCard, ok := dev.(types.BaseVirtualEthernetCard)
		if !ok {
			return errors.New("device is not a base virtual ethernet card")
		}
		ethCard := bvEthCard.GetVirtualEthernetCard()
		spec := &types.VirtualDeviceConfigSpec{}
		spec.Operation = types.VirtualDeviceConfigSpecOperationRemove
		spec.Device = ethCard
		deviceConfigSpecs = append(deviceConfigSpecs, spec)
	}

	vmConfigSpec := types.VirtualMachineConfigSpec{}
	vmConfigSpec.DeviceChange = deviceConfigSpecs

	task, err := vm.Reconfigure(ctx, vmConfigSpec)
	if err != nil {
		return errors.Wrapf(err, "could not reconfigure vm %s (%s)", vm.InventoryPath, vm.Reference())
	}

	if err := task.Wait(ctx); err != nil {
		return errors.Wrapf(err, "failed waiting on vm reconfigure task for %s (%s)", vm.InventoryPath, vm.Reference())
	}

	return nil
}

func sliceDedup(list []string) []string {
	sort.Strings(list)
	j := 0
	for i := 1; i < len(list); i++ {
		if list[j] == list[i] {
			continue
		}
		j++
		list[j] = list[i]
	}
	return list[:j+1]
}
