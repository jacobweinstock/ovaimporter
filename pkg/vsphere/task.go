package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/object"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// DatastoreCapacity gets the datastore capacity and free space in KB from vcenter and returns it in GB
// referenced from https://github.com/vmware/govmomi/blob/master/govc/datastore/info.go
func (s *Session) DatastoreCapacity() (capacity, free float64, err error) {
	var dss []mo.Datastore
	var summary types.DatastoreSummary

	pc := s.Conn.PropertyCollector()

	if s.Datastore == nil {
		return capacity, free, errors.New("no datastore specified in connection session")
	}
	refs := []types.ManagedObjectReference{s.Datastore.Reference()}
	err = pc.Retrieve(s.Ctx, refs, nil, &dss)
	if err != nil {
		return capacity, free, errors.Wrapf(err, fmt.Sprintf("error retrieving datastore details: %v", s.Datastore.String()))
	}
	if len(dss) > 0 {
		summary = dss[0].Summary
		return float64(summary.Capacity) / (1 << 30), float64(summary.FreeSpace) / (1 << 30), nil
	}
	return capacity, free, errors.New(fmt.Sprintf("datastore(%v) properties not found: %v", dss, s.Datastore.String()))
}

// GetVMTotalStorageSize returns the total VM storage size (in GB) of all attached disks to a VM
func (s *Session) GetVMTotalStorageSize(vmName string) (size float64, err error) {
	vm, err := s.GetVM(vmName)
	if err != nil {
		return size, err
	}
	devices, err := vm.Device(s.Ctx)
	if err != nil {
		return size, err
	}
	var totalSize int64
	for _, elem := range devices {
		switch md := elem.(type) {
		case *types.VirtualDisk:
			totalSize += md.CapacityInKB
		}
	}
	size = float64(totalSize) / 1e6
	return size, err
}

func getProperties(ctx context.Context, vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, errors.Wrap(err, "unable to get virtual machine properties")
	}
	return &props, nil
}
