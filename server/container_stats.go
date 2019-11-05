package server

import (
	"path/filepath"
	"time"

	"github.com/cri-o/cri-o/internal/oci"
	crioStorage "github.com/cri-o/cri-o/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func buildContainerStats(stats *oci.ContainerStats, container *oci.Container) *pb.ContainerStats {
	// TODO: Fix this for other storage drivers. This will only work with overlay.
	diffDir := filepath.Join(filepath.Dir(container.MountPoint()), "diff")
	bytesUsed, inodeUsed, err := crioStorage.GetDiskUsageStats(diffDir)
	if err != nil {
		logrus.Warnf("unable to get disk usage for container %s， %s", container.ID(), err)
	}
	return &pb.ContainerStats{
		Attributes: &pb.ContainerAttributes{
			Id:          container.ID(),
			Metadata:    container.Metadata(),
			Labels:      container.Labels(),
			Annotations: container.Annotations(),
		},
		Cpu: &pb.CpuUsage{
			Timestamp:            stats.SystemNano,
			UsageCoreNanoSeconds: &pb.UInt64Value{Value: stats.CPUNano},
		},
		Memory: &pb.MemoryUsage{
			Timestamp:       stats.SystemNano,
			WorkingSetBytes: &pb.UInt64Value{Value: stats.MemUsage},
		},
		WritableLayer: &pb.FilesystemUsage{
			Timestamp:  stats.SystemNano,
			FsId:       &pb.FilesystemIdentifier{Mountpoint: container.MountPoint()},
			UsedBytes:  &pb.UInt64Value{Value: bytesUsed},
			InodesUsed: &pb.UInt64Value{Value: inodeUsed},
		},
	}
}

// ContainerStats returns stats of the container. If the container does not
// exist, the call returns an error.
func (s *Server) ContainerStats(ctx context.Context, req *pb.ContainerStatsRequest) (resp *pb.ContainerStatsResponse, err error) {
	const operation = "container_stats"
	defer func() {
		recordOperation(operation, time.Now())
		recordError(operation, err)
	}()

	container, err := s.GetContainerFromShortID(req.ContainerId)
	if err != nil {
		return nil, err
	}

	stats, err := s.Runtime().ContainerStats(container)
	if err != nil {
		return nil, err
	}

	return &pb.ContainerStatsResponse{Stats: buildContainerStats(stats, container)}, nil
}
