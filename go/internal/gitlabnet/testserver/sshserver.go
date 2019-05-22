package testserver

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"

	"google.golang.org/grpc"

	pb "gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
)

type testSSHServer struct{}

func (s *testSSHServer) SSHReceivePack(stream pb.SSHService_SSHReceivePackServer) error {
	req, err := stream.Recv()

	if err != nil {
		log.Fatal(err)
	}

	response := []byte("ReceivePack: " + req.GlId + " " + req.Repository.GlRepository)
	stream.Send(&pb.SSHReceivePackResponse{Stdout: response})

	return nil
}

func (s *testSSHServer) SSHUploadPack(stream pb.SSHService_SSHUploadPackServer) error {
	return nil
}

func (s *testSSHServer) SSHUploadArchive(stream pb.SSHService_SSHUploadArchiveServer) error {
	return nil
}

func StartSSHServer() (func(), string, error) {
	tempDir, _ := ioutil.TempDir("", "gitlab-shell-test-api")
	gitalySocketPath := path.Join(tempDir, "gitaly.sock")

	if err := os.MkdirAll(filepath.Dir(gitalySocketPath), 0700); err != nil {
		return nil, "", err
	}

	server := grpc.NewServer()

	listener, err := net.Listen("unix", gitalySocketPath)
	if err != nil {
		return nil, "", err
	}

	pb.RegisterSSHServiceServer(server, &testSSHServer{})

	go server.Serve(listener)

	gitalySocketUrl := "unix:" + gitalySocketPath
	cleanup := func() {
		server.Stop()
		os.RemoveAll(tempDir)
	}

	return cleanup, gitalySocketUrl, nil
}
