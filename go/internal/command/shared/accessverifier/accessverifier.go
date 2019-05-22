package accessverifier

import (
	"fmt"

	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/commandargs"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/command/readwriter"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/config"
	"gitlab.com/gitlab-org/gitlab-shell/go/internal/gitlabnet/accessverifier"
)

type Response = accessverifier.Response

type Command struct {
	Config     *config.Config
	Args       *commandargs.CommandArgs
	ReadWriter *readwriter.ReadWriter
}

func (c *Command) Verify(action commandargs.CommandType, repo string) (*Response, error) {
	client, err := accessverifier.NewClient(c.Config)
	if err != nil {
		return nil, err
	}

	response, err := client.Verify(c.Args, action, repo)
	if err != nil {
		return nil, err
	}

	c.displayConsoleMessages(response.ConsoleMessages)

	if response.Success {
		return response, nil
	} else {
		return nil, fmt.Errorf(response.Message)
	}
}

func (c *Command) displayConsoleMessages(messages []string) {
	var msgString string
	for _, msg := range messages {
		msgString += "> GitLab: " + msg + "\n"
	}

	if msgString != "" {
		fmt.Fprint(c.ReadWriter.Out, msgString)
	}
}
