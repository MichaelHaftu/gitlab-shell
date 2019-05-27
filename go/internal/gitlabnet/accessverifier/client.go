package accessverifier

import (
	"fmt"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet"
)

const (
	protocol   = "ssh"
	anyChanges = "_any"
)

type Client struct {
	client *gitlabnet.GitlabClient
}

type Request struct {
	Action   commandargs.CommandType `json:"action"`
	Repo     string                  `json:"project"`
	Changes  string                  `json:"changes"`
	Protocol string                  `json:"protocol"`
	KeyId    string                  `json:"key_id,omitempty"`
	Username string                  `json:"username,omitempty"`
}

type GitalyRepo struct {
	StorageName                   string   `json:"storage_name"`
	RelativePath                  string   `json:"relative_path"`
	GitObjectDirectory            string   `json:"git_object_directory"`
	GitAlternateObjectDirectories []string `json:"git_alternate_object_directories"`
	RepoName                      string   `json:"gl_repository"`
	ProjectPath                   string   `json:"gl_project_path"`
}

type Gitaly struct {
	Repo    GitalyRepo `json:"repository"`
	Address string     `json:"address"`
	Token   string     `json:"token"`
}

type CustomPayloadData struct {
	ApiEndpoints []string `json:"api_endpoints"`
	Username     string   `json:"gl_username"`
	PrimaryRepo  string   `json:"primary_repo"`
	InfoMessage  string   `json:"info_message"`
	UserId       string   `json:"gl_id,omitempty"`
}

type CustomPayload struct {
	Action string            `json:"action"`
	Data   CustomPayloadData `json:"data"`
}

type Response struct {
	Success          bool          `json:"status"`
	Message          string        `json:"message"`
	Repo             string        `json:"gl_repository"`
	UserId           string        `json:"gl_id"`
	Username         string        `json:"gl_username"`
	GitConfigOptions []string      `json:"git_config_options"`
	Gitaly           Gitaly        `json:"gitaly"`
	GitProtocol      string        `json:"git_protocol"`
	Payload          CustomPayload `json:"payload"`
	ConsoleMessages  []string      `json:"gl_console_messages"`
	StatusCode       int
}

func NewClient(config *config.Config) (*Client, error) {
	client, err := gitlabnet.GetClient(config)
	if err != nil {
		return nil, fmt.Errorf("Error creating http client: %v", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Verify(args *commandargs.CommandArgs, action commandargs.CommandType, repo string) (*Response, error) {
	request := &Request{Action: action, Repo: repo, Protocol: protocol, Changes: anyChanges}
	if args.GitlabUsername != "" {
		request.Username = args.GitlabUsername
	} else {
		request.KeyId = args.GitlabKeyId
	}

	response, err := c.client.Post("/allowed", request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return parse(response)
}

func parse(hr *http.Response) (*Response, error) {
	response := &Response{}
	if err := gitlabnet.ParseJSON(hr, response); err != nil {
		return nil, err
	}

	response.StatusCode = hr.StatusCode

	return response, nil
}

func (r *Response) IsCustomAction() bool {
	return r.StatusCode == http.StatusMultipleChoices
}
