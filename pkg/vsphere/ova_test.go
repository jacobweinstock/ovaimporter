// +build !integration

package vsphere

import (
	"testing"
)

func TestDeployOVATemplate(t *testing.T) {
	var err error
	sim.conn.Network, err = sim.conn.GetNetworkOrDefault("/DC0/network/VM Network")
	if err != nil {
		t.Fatal(err)
	}
	sim.conn.ResourcePool, err = sim.conn.GetResourcePoolOrDefault("/DC0/host/DC0_H0/Resources")
	if err != nil {
		t.Fatal(err)
	}
	sim.conn.Datastore, err = sim.conn.GetDatastoreOrDefault("/DC0/datastore/LocalDS_0")
	if err != nil {
		t.Fatal(err)
	}
	//templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/DC0_C0_RP0_VM1.ova"
	//templateOVA = "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"
	templateOVA := "https://communities.vmware.com/servlet/JiveServlet/downloadBody/21621-102-3-28798/Tiny Linux VM.ova"

	_, err = sim.conn.DeployOVATemplate(templateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}
}
