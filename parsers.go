package main

import "encoding/json"

type messageRepository struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	SSHURL string `json:"ssh_url"`
}

func parseIssueComment(body []byte) (IssueComment, error) {
	var message struct {
		Issue struct {
			Number      int `json:"Number"`
			PullRequest struct {
				URL string `json:"url"`
			} `json:"pull_request"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"issue"`
		Repository messageRepository `json:"repository"`
		Comment    struct {
			Body string `json:"body"`
		} `json:"comment"`
	}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return IssueComment{}, err
	}
	return IssueComment{
		IssueNumber:   message.Issue.Number,
		Comment:       message.Comment.Body,
		IsPullRequest: message.Issue.PullRequest.URL != "",
		Repository: Repository{
			Owner: message.Repository.Owner.Login,
			Name:  message.Repository.Name,
			URL:   message.Repository.SSHURL,
		},
		User: User{
			Login: message.Issue.User.Login,
		},
	}, nil
}

func parsePullRequestEvent(body []byte) (PullRequestEvent, error) {
	var message struct {
		Action      string `json:"action"`
		Number      int    `json:"number"`
		PullRequest struct {
			Head struct {
				SHA        string            `json:"sha"`
				Repository messageRepository `json:"repo"`
			} `json:"head"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"pull_request"`
		Repository messageRepository `json:"repository"`
	}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return PullRequestEvent{}, err
	}
	return PullRequestEvent{
		IssueNumber: message.Number,
		Action:      message.Action,
		Head: PullRequestBranch{
			SHA: message.PullRequest.Head.SHA,
			Repository: Repository{
				Owner: message.PullRequest.Head.Repository.Owner.Login,
				Name:  message.PullRequest.Head.Repository.Name,
				URL:   message.PullRequest.Head.Repository.SSHURL,
			},
		},
		Repository: Repository{
			Owner: message.Repository.Owner.Login,
			Name:  message.Repository.Name,
			URL:   message.Repository.SSHURL,
		},
		User: User{
			Login: message.PullRequest.User.Login,
		},
	}, nil
}

func parseStatusEvent(body []byte) (StatusEvent, error) {
	var message struct {
		SHA      string `json:"sha"`
		State    string `json:"state"`
		Branches []struct {
			Commit struct {
				SHA string `json:"sha"`
			} `json:"commit"`
		} `json:"branches"`
		Repository messageRepository `json:"repository"`
	}

	err := json.Unmarshal(body, &message)
	if err != nil {
		return StatusEvent{}, err
	}

	branches := make([]Branch, len(message.Branches))
	for i, branch := range message.Branches {
		branches[i] = Branch{
			SHA: branch.Commit.SHA,
		}
	}
	return StatusEvent{
		SHA:      message.SHA,
		State:    message.State,
		Branches: branches,
		Repository: Repository{
			Owner: message.Repository.Owner.Login,
			Name:  message.Repository.Name,
			URL:   message.Repository.SSHURL,
		},
	}, nil
}
