package handler

import (
	"context"
	"os"

	pb "gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/client"
	"google.golang.org/grpc"
)

// UploadArchive issues a Gitaly upload-archive rpc to the provided address
func UploadArchive(ctx context.Context, conn *grpc.ClientConn, request *pb.SSHUploadArchiveRequest) (int32, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return client.UploadArchive(ctx, conn, os.Stdin, os.Stdout, os.Stderr, request)
}
