package accessverifier

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/readwriter"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet/accessverifier"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet/testserver"
)

var (
	repo   = "group/repo"
	action = commandargs.ReceivePack
)

func setup(t *testing.T) (*Command, *bytes.Buffer, func()) {
	requests := []testserver.TestRequestHandler{
		{
			Path: "/api/v4/internal/allowed",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				b, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()

				require.NoError(t, err)

				var requestBody *accessverifier.Request
				json.Unmarshal(b, &requestBody)

				if requestBody.KeyId == "1" {
					successfulBody := map[string]interface{}{
						"gl_console_messages": []string{"console", "message"},
					}
					json.NewEncoder(w).Encode(successfulBody)
				} else {
					body := map[string]interface{}{
						"status":  false,
						"message": "missing user",
					}
					json.NewEncoder(w).Encode(body)
				}
			},
		},
	}

	cleanup, url, err := testserver.StartSocketHttpServer(requests)
	require.NoError(t, err)

	buffer := &bytes.Buffer{}

	readWriter := &readwriter.ReadWriter{Out: buffer}
	cmd := &Command{Config: &config.Config{GitlabUrl: url}, ReadWriter: readWriter}

	return cmd, buffer, cleanup
}

func TestMissingUser(t *testing.T) {
	cmd, _, cleanup := setup(t)
	defer cleanup()

	cmd.Args = &commandargs.CommandArgs{GitlabKeyId: "2"}
	_, err := cmd.Verify(action, repo)

	assert.Equal(t, "missing user", err.Error())
}

func TestConsoleMessages(t *testing.T) {
	cmd, buffer, cleanup := setup(t)
	defer cleanup()

	cmd.Args = &commandargs.CommandArgs{GitlabKeyId: "1"}
	cmd.Verify(action, repo)

	assert.Equal(t, "> GitLab: console\n> GitLab: message\n", buffer.String())
}
