# OVA Importer

[![Test and Build](https://github.com/jacobweinstock/ovaimporter/workflows/Test%20and%20Build/badge.svg)](https://github.com/jacobweinstock/ovaimporter/actions?query=workflow%3A%22Test+and+Build%22)
[![Go Report](https://goreportcard.com/badge/github.com/jacobweinstock/ovaimporter)](https://goreportcard.com/report/github.com/jacobweinstock/ovaimporter)

Import a local or remote OVA.

## Alternatives

Why not just use `govc`?   
`govc import.ova` requires a spec file, or you get something like `govc: Host did not have any virtual network defined.`  
It has a command to get the spec file `govc import.spec`, but alas, it doesn't take a remote location: `govc: remote path not supported`.  
This means in order to leverage `govc` we'd have to download the ova and then run `govc import.spec` and add a network or find another tool or some other way.  
Downloading the OVA only to turn around and upload it to a vCenter could be a considerable amount of extra time not needed, hence the advent of `ovaimporter`.  

## Usage

### Build

Build the binary by running `make build`

### Run

`ovaimporter` return a response to stdout, and a file (defaults to `./response.json`) in json format.


```bash
ovaimporter 
  --ova https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova \
  --network VM_Net \
  --datacenter Datacenter-01 \
  --datastore Datastore-01 \
  --url 10.96.160.151 \
  --user administrator@vsphere.local \
  --password 'secret'
```


```bash
# Environment variables can be used in place of the CLI flags
OVAIMPORTER_URL=vcenter.example.org \
OVAIMPORTER_USER=myuser \
OVAIMPORTER_PASSWORD=secret \
OVAIMPORTER_OVA=https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova \
OVAIMPORTER_DATACENTER=dc1 \
OVAIMPORTER_DATASTORE=ds1 \
OVAIMPORTER_NETWORK=VM_Net \
ovaimporter
```

##### Response Object

For more details on the data types, the go `importerResponse` struct can be found here: `cmd/response.go`
```json
{
  "alreadyExists": false,
  "errorMsg": "",
  "level": "",
  "msg": "",
  "name": "",
  "responseFile": "",
  "success": true,
  "time": ""
}

```

### Container Image

A container image is available at `docker pull ghcr.io/jacobweinstock/ovaimporter:latest`  

#### USAGE

Build the container image locally with `make build`

```bash
# using cli flags
docker run -it --rm -v ${PWD}/response.json:/response.json ghcr.io/jacobweinstock/ovaimporter \
  --url 10.96.160.151 \
  --user admin \
  --password 'secret' \  
  --ova https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova \
  --datacenter DC01 \
  --datastore DS01 \
  --network VM_Net

# using env vars
docker run -it --rm \
  -e OVAIMPORTER_URL=10.96.160.151 \
  -e OVAIMPORTER_USER=admin \
  -e OVAIMPORTER_PASSWORD='secret' \
  -e OVAIMPORTER_OVA=https://storage.googleapis.com/capv-images/release/v1.17.3/ubuntu-1804-kube-v1.17.3.ova \
  -e OVAIMPORTER_DATACENTER=Datacenter-01 \
  -e OVAIMPORTER_DATASTORE=Datastore-01 \
  -e OVAIMPORTER_NETWORK=VM_Net \
  -v ${PWD}/response.json:/response.json \
  ghcr.io/jacobweinstock/ovaimporter
```