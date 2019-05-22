package accessverifier

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet/testserver"
)

var (
	repo           = "group/repo"
	action         = commandargs.ReceivePack
	successfulBody = map[string]interface{}{
		"status":             true,
		"gl_id":              "1",
		"gl_repository":      repo,
		"gl_username":        "someuser",
		"git_config_options": []string{"option"},
		"gitaly": map[string]interface{}{
			"repository": map[string]interface{}{
				"storage_name":                     "storage_name",
				"relative_path":                    "relative_path",
				"git_object_directory":             "path/to/git_object_directory",
				"git_alternate_object_directories": []string{"path/to/git_alternate_object_directory"},
				"gl_repository":                    "group/gitaly-repo",
				"gl_project_path":                  "group/project-path",
			},
			"address": "address",
			"token":   "token",
		},
		"git_protocol":        "protocol",
		"payload":             map[string]interface{}{},
		"gl_console_messages": []string{"console", "message"},
	}
	expectedSuccessfulResponse = &Response{
		Success:          true,
		UserId:           "1",
		Repo:             repo,
		Username:         "someuser",
		GitConfigOptions: []string{"option"},
		Gitaly: Gitaly{
			Repo: GitalyRepo{
				StorageName:                   "storage_name",
				RelativePath:                  "relative_path",
				GitObjectDirectory:            "path/to/git_object_directory",
				GitAlternateObjectDirectories: []string{"path/to/git_alternate_object_directory"},
				RepoName:                      "group/gitaly-repo",
				ProjectPath:                   "group/project-path",
			},
			Address: "address",
			Token:   "token",
		},
		GitProtocol:     "protocol",
		Payload:         CustomPayload{},
		ConsoleMessages: []string{"console", "message"},
		StatusCode:      200,
	}
)

func TestSuccessfulResponses(t *testing.T) {
	client, cleanup := setup(t)
	defer cleanup()

	testCases := []struct {
		desc string
		args *commandargs.CommandArgs
	}{
		{
			desc: "Provide key id within the request",
			args: &commandargs.CommandArgs{GitlabKeyId: "1"},
		}, {
			desc: "Provide username within the request",
			args: &commandargs.CommandArgs{GitlabUsername: "first"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := client.Verify(tc.args, action, repo)
			assert.NoError(t, err)

			assert.Equal(t, expectedSuccessfulResponse, result)
		})
	}
}

func TestGetCustomAction(t *testing.T) {
	client, cleanup := setup(t)
	defer cleanup()

	args := &commandargs.CommandArgs{GitlabUsername: "custom"}
	result, err := client.Verify(args, action, repo)
	assert.NoError(t, err)

	expectedSuccessfulResponse.Payload = CustomPayload{
		Action: "geo_proxy_to_primary",
		Data: CustomPayloadData{
			ApiEndpoints: []string{"geo/proxy_git_push_ssh/info_refs", "geo/proxy_git_push_ssh/push"},
			Username:     "custom",
			PrimaryRepo:  "https://repo/path",
			InfoMessage:  "message",
		},
	}
	expectedSuccessfulResponse.StatusCode = 300

	require.True(t, expectedSuccessfulResponse.IsCustomAction())
	assert.Equal(t, expectedSuccessfulResponse, result)
}

func TestErrorResponses(t *testing.T) {
	client, cleanup := setup(t)
	defer cleanup()

	testCases := []struct {
		desc          string
		fakeId        string
		expectedError string
	}{
		{
			desc:          "A response with an error message",
			fakeId:        "2",
			expectedError: "Not allowed!",
		},
		{
			desc:          "A response with bad JSON",
			fakeId:        "3",
			expectedError: "Parsing failed",
		},
		{
			desc:          "An error response without message",
			fakeId:        "4",
			expectedError: "Internal API error (403)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			args := &commandargs.CommandArgs{GitlabKeyId: tc.fakeId}
			resp, err := client.Verify(args, action, repo)

			assert.EqualError(t, err, tc.expectedError)
			assert.Nil(t, resp)
		})
	}
}

func setup(t *testing.T) (*Client, func()) {
	requests := []testserver.TestRequestHandler{
		{
			Path: "/api/v4/internal/allowed",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				b, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()

				require.NoError(t, err)

				var requestBody *Request
				json.Unmarshal(b, &requestBody)

				switch requestBody.Username {
				case "first":
					json.NewEncoder(w).Encode(successfulBody)
				case "second":
					body := map[string]interface{}{
						"status":  false,
						"message": "missing user",
					}
					json.NewEncoder(w).Encode(body)
				case "custom":
					successfulBody["payload"] = map[string]interface{}{
						"action": "geo_proxy_to_primary",
						"data": map[string]interface{}{
							"api_endpoints": []string{"geo/proxy_git_push_ssh/info_refs", "geo/proxy_git_push_ssh/push"},
							"gl_username":   "custom",
							"primary_repo":  "https://repo/path",
							"info_message":  "message",
						},
					}

					w.WriteHeader(http.StatusMultipleChoices)
					json.NewEncoder(w).Encode(successfulBody)
				}

				switch requestBody.KeyId {
				case "1":
					json.NewEncoder(w).Encode(successfulBody)
				case "2":
					w.WriteHeader(http.StatusForbidden)
					body := &gitlabnet.ErrorResponse{
						Message: "Not allowed!",
					}
					json.NewEncoder(w).Encode(body)
				case "3":
					w.Write([]byte("{ \"message\": \"broken json!\""))
				case "4":
					w.WriteHeader(http.StatusForbidden)
				}
			},
		},
	}

	cleanup, url, err := testserver.StartSocketHttpServer(requests)
	require.NoError(t, err)

	client, err := NewClient(&config.Config{GitlabUrl: url})
	require.NoError(t, err)

	return client, cleanup
}
