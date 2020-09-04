package vsphere

import (
	"fmt"
	"testing"
)

func TestDatastoreCapacitySuccess(t *testing.T) {
	var err error
	name := "/DC0/datastore/LocalDS_0"
	sim.conn.Datastore, err = sim.conn.GetDatastoreOrDefault(name)
	if err != nil {
		t.Fatalf(err.Error())
	}
	capacity, free, err := sim.conn.DatastoreCapacity()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if capacity < free {
		t.Fatalf("expected capacity(%v) > free(%v)", capacity, free)
	}
}

func TestDatastoreCapacityNoDSSpecified(t *testing.T) {
	sim.conn.Datastore = nil
	_, _, err := sim.conn.DatastoreCapacity()
	if err == nil {
		t.Fatal("received an unexpected nil error")
	}
	if err.Error() != "no datastore specified in connection session" {
		t.Fatalf(err.Error())
	}
}

func TestGetVMTotalStorageSize(t *testing.T) {
	vmName := "DC0_C0_RP0_VM0"
	sizeExpected := 0.001024
	size, err := sim.conn.GetVMTotalStorageSize(vmName)
	if err != nil {
		t.Fatal(err)
	}
	if size != sizeExpected {
		t.Fatalf("expected size: %v, actual: %v", sizeExpected, size)
	}
}

func TestGetVMTotalStorageSizeVMNotFound(t *testing.T) {
	vmName := "i_dont_exist"
	_, err := sim.conn.GetVMTotalStorageSize(vmName)
	if err == nil {
		t.Fatal("received an unexpected nil error")
	}
	if err.Error() != fmt.Sprintf("error finding VM %v: vm '%[1]v' not found", vmName) {
		t.Fatalf(err.Error())
	}

}
