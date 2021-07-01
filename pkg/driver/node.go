/*
Copyright 2021 The OpenEBS Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"context"
	"fmt"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/utils/mount"

	"github.com/openebs/dynamic-nfs-provisioner/pkg/kubernetes/client"
)

const (
	FsTypeNfs = "nfs"
)

type nodeServer struct {
	driver  *Driver
	mounter mount.Interface
	client  *client.Client
}

func NewNodeServer(d *Driver, mounter mount.Interface) csi.NodeServer {
	client := client.New()
	return &nodeServer{
		driver:  d,
		mounter: mounter,
		client:  client,
	}
}

// NodeGetInfo returns node details
func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	node, err := getNode(ns.client, ns.driver.nodeID)
	if err != nil {
		klog.Errorf("failed to get the node %s", ns.driver.nodeID)
		return nil, err
	}
	/*
	 * The driver will support all the keys and values defined in the node's label.
	 * if nodes are labeled with the below keys and values
	 * map[beta.kubernetes.io/arch:amd64 beta.kubernetes.io/os:linux kubernetes.io/arch:amd64 kubernetes.io/hostname:pawan-node-1 kubernetes.io/os:linux node-role.kubernetes.io/worker:true openebs.io/zone:  zone1 openebs.io/zpool:ssd]
	 * The driver will support below key and values
	 * {
	 *      beta.kubernetes.io/arch:amd64
	 *      beta.kubernetes.io/os:linux
	 *      kubernetes.io/arch:amd64
	 *      kubernetes.io/hostname:pawan-node-1
	 *      kubernetes.io/os:linux
	 *      node-role.kubernetes.io/worker:true
	 *      openebs.io/zone:zone1
	 *      openebs.io/zpool:ssd
	 * }
	 */

	// support all the keys that node has
	topology := node.Labels

	// add driver's topology key
	topology[TopologyNodenameKey] = ns.driver.nodeID

	return &csi.NodeGetInfoResponse{
		NodeId: ns.driver.nodeID,
		AccessibleTopology: &csi.Topology{
			Segments: topology,
		},
	}, nil
}

// NodeGetCapabilities returns capabilities of the node plugin
func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
			/*
				{
					Type: &csi.NodeServiceCapability_Rpc{
						Rpc: &csi.NodeServiceCapability_RPC{
							Type: csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
						},
					},
				},
			*/
		},
	}, nil
}

// NodeStageVolume mounts the volume on the staging path
func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeUnstageVolume unmounts the volume from the staging path
func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeExpandVolume resizes the filesystem if required
func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodePublishVolume publishes (mounts) the volume at the corresponding node at a given path
func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// validate the request
	err := validateNodePulishReq(req)
	if err != nil {
		return nil, err
	}

	target := req.GetTargetPath()

	server := req.GetVolumeContext()[NodeParamServer]
	path := req.GetVolumeContext()[NodeParamPath]
	source := fmt.Sprintf("%s:%s", server, path)

	// mount option
	mountOptions := req.GetVolumeCapability().GetMount().GetMountFlags()
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	if err := ns.mounter.Mount(source, target, FsTypeNfs, mountOptions); err != nil {
		if os.IsPermission(err) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unpublishes (unmounts) the volume
// from the corresponding node from the given path
func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	// validate the request
	err := validateNodeUnpulishReq(req)
	if err != nil {
		return nil, err
	}

	target := req.GetTargetPath()

	notMnt, err := ns.mounter.IsLikelyNotMountPoint(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "Targetpath not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if notMnt {
		return nil, status.Errorf(codes.NotFound, "Volume not mounted")
	}

	err = mount.CleanupMountPoint(target, ns.mounter, false)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("could not unmount %q, err=%v", target, err))
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetVolumeStats returns statistics for the given volume
func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	var sfs unix.Statfs_t

	volID := req.GetVolumeId()
	path := req.GetVolumePath()

	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume Id missing in request")
	}
	if len(path) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume path missing in request")
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "path=%s does not exist", path)
		}
		return nil, status.Errorf(codes.Internal, "error accessing path=%s, err=%v", path, err)
	}

	if err := unix.Statfs(path, &sfs); err != nil {
		return nil, status.Errorf(codes.Internal, "statfs on %s failed, err=%v", path, err)
	}

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Unit:      csi.VolumeUsage_BYTES,
				Available: int64(sfs.Bavail) * int64(sfs.Bsize),
				Total:     int64(sfs.Blocks) * int64(sfs.Bsize),
				Used:      int64(sfs.Blocks-sfs.Bfree) * int64(sfs.Bsize),
			},
			{
				Unit:      csi.VolumeUsage_INODES,
				Available: int64(sfs.Ffree),
				Total:     int64(sfs.Files),
				Used:      int64(sfs.Files - sfs.Ffree),
			},
		},
	}, nil
}

func validateNodePulishReq(req *csi.NodePublishVolumeRequest) error {
	if req.GetVolumeCapability() == nil {
		return status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}

	if len(req.GetVolumeId()) == 0 {
		return status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if len(req.GetTargetPath()) == 0 {
		return status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return nil
}

func validateNodeUnpulishReq(req *csi.NodeUnpublishVolumeRequest) error {
	if len(req.GetVolumeId()) == 0 {
		return status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if len(req.GetTargetPath()) == 0 {
		return status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return nil
}

func getNode(c *client.Client, name string) (*corev1.Node, error) {
	cl, err := c.Clientset()
	if err != nil {
		return nil, err
	}

	return cl.CoreV1().Nodes().Get(name, metav1.GetOptions{})
}
