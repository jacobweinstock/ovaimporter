// +build integration

package vsphere

import (
	"context"
	"testing"
	"time"
)

func TestDeployOVATemplateRealVcenter(t *testing.T) {
	t.Skip("skipping test, real vCenter needed.")

	timeout := 5 * time.Minute
	ctx, can := context.WithTimeout(context.Background(), timeout)
	defer can()
	cl, err := NewClient(ctx, "10.96.160.151", "administrator@vsphere.local", "NetApp1!!")
	if err != nil {
		t.Fatalf(err.Error())
	}
	cl.Datacenter, err = cl.GetDatacenterOrDefault("NetApp-HCI-Datacenter-01")
	if err != nil {
		t.Fatalf(err.Error())
	}

	/*
		all, err := cl.CreateVMFolder("/NetApp-HCI-Datacenter-01/vm/cake/two/three")
		if err != nil {
			t.Fatalf(err.Error())
		}
		fmt.Println(all["three"].InventoryPath)
		t.Fail()


			dcf, err := cl.Datacenter.Folders(context.TODO())
			if err != nil {
				t.Fatalf(err.Error())
			}

			fobj, err := dcf.VmFolder.CreateFolder(context.TODO(), "one/two/three")
			if err != nil {
				t.Fatalf(err.Error())
			}
			fmt.Println(fobj.InventoryPath)
			fmt.Println(fobj)



				cl.Network, err = cl.GetNetworkOrDefault("NetApp HCI VDS 01-HCI_Internal_mNode_Network")
				if err != nil {
					t.Fatalf(err.Error())
				}
				cl.Datastore, err = cl.GetDatastoreOrDefault("NetApp-HCI-Datastore-02")
				if err != nil {
					t.Fatalf(err.Error())
				}

				cl.Folder, err = cl.GetFolderOrDefault("k8s")
				if err != nil {
					t.Fatalf(err.Error())
				}
	*/

	//cl.ResourcePool, err = cl.GetResourcePoolOrDefault("*/Resources")
	//if err != nil {
	//	t.Fatalf(err.Error())
	//}

	//templateName := "ubuntu-1804-kube-v1.17.3"
	/*
		templateOVA := "https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova"
		_, err = cl.DeployOVATemplate(templateOVA)
		if err != nil {
			t.Fatalf(err.Error())
		}
	*/

	templateOVA := "https://communities.vmware.com/servlet/JiveServlet/downloadBody/21621-102-3-28798/Tiny Linux VM.ova"
	_, err = cl.DeployOVATemplate(templateOVA)
	if err != nil {
		t.Fatalf(err.Error())
	}
}
