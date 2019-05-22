package receivepack

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet/accessverifier"
)

type Request struct {
	Data   accessverifier.CustomPayloadData `json:"data"`
	Output string                           `json:"output"`
}

type Response struct {
	Result  string `json:"result"`
	Message string `json:"message"`
}

func (c *Command) processCustomAction(response *accessverifier.Response) error {
	data := response.Payload.Data
	apiEndpoints := data.ApiEndpoints

	if len(apiEndpoints) == 0 {
		return errors.New("Custom action error: Empty Api endpoints")
	}

	c.displayInfoMessage(data.InfoMessage)

	return c.processApiEndpoints(response)
}

func (c *Command) displayInfoMessage(infoMessage string) {
	if infoMessage != "" {
		fmt.Fprintf(c.ReadWriter.Out, "> GitLab: %v\n", infoMessage)
	}
}

func (c *Command) processApiEndpoints(response *accessverifier.Response) error {
	client, err := gitlabnet.GetClient(c.Config)

	if err != nil {
		return err
	}

	data := response.Payload.Data
	request := &Request{Data: data}
	request.Data.UserId = response.UserId

	for _, endpoint := range data.ApiEndpoints {
		response, err := c.performRequest(client, endpoint, request)
		if err != nil {
			return err
		}

		if err := c.displayResult(response.Result); err != nil {
			return err
		}

		// In the context of the git push sequence of events, it's necessary to read
		// stdin in order to capture output to pass onto subsequent commands
		request.Output = c.readStdIn()
	}

	return nil
}

func (c *Command) performRequest(client *gitlabnet.GitlabClient, endpoint string, request *Request) (*Response, error) {
	response, err := client.DoRequest(http.MethodPost, endpoint, request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	cr := &Response{}
	if err := gitlabnet.ParseJSON(response, cr); err != nil {
		return nil, err
	}

	return cr, nil
}

func (c *Command) displayResult(result string) error {
	decodedOutput, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.ReadWriter.Out, string(decodedOutput))
	return nil
}

func (c *Command) readStdIn() string {
	var input string
	fmt.Fscan(c.ReadWriter.In, &input)

	return base64.StdEncoding.EncodeToString([]byte(input))
}
