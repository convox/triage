package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	ctx := context.Background()

	if len(os.Args) < 3 {
		usage()
	}

	if err := triage(ctx, os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: triage <repo> <prefix>\n")
	os.Exit(1)
}

func triage(ctx context.Context, orgrepo, prefix string) error {
	g, err := githubClient()
	if err != nil {
		return err
	}

	parts := strings.Split(orgrepo, "/")

	if len(parts) != 2 {
		return fmt.Errorf("invalid repo: %s", orgrepo)
	}

	org := parts[0]
	repo := parts[1]

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	issues := []github.Issue{}

	for {
		i, res, err := g.Search.Issues(ctx, fmt.Sprintf("repo:%s/%s is:issue is:open", org, repo), opts)
		if err != nil {
			return err
		}

		issues = append(issues, i.Issues...)

		if res.NextPage == 0 {
			break
		}

		opts.Page = res.NextPage
	}

	filtered := []github.Issue{}

	for _, issue := range issues {
		found := false

		for _, l := range issue.Labels {
			if strings.HasPrefix(*l.Name, fmt.Sprintf("%s/", prefix)) {
				found = true
				break
			}
		}

		if !found {
			filtered = append(filtered, issue)
		}
	}

	for ix, i := range filtered {
		fmt.Print("\033[H\033[2J")

		fmt.Printf("[%d] %s\n", *i.Number, *i.Title)
		fmt.Println(*i.Body)

		// c, err := githubComments(ctx, org, repo, *i.Number)
		// fmt.Printf("c = %+v\n", c)
		// fmt.Printf("i.Labels = %+v\n", i.Labels)

		fmt.Printf("(%d remaining)\n", len(filtered)-ix)

		v, err := read(prefix)
		if err != nil {
			return err
		}

		switch v {
		case "next":
			continue
		default:
			_, _, err = g.Issues.AddLabelsToIssue(ctx, org, repo, *i.Number, []string{fmt.Sprintf("%s/%s", prefix, v)})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func read(label string) (string, error) {
	fmt.Printf("%s: ", label)
	var s string
	_, err := fmt.Scanf("%s", &s)
	if err != nil {
		return "", err
	}
	return s, nil
}

func githubClient() (*github.Client, error) {
	token, err := githubToken()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client, nil
}

// func githubComments(ctx context.Context, org, repo string, number int) ([]*github.IssueComment, error) {
//   opts := &github.IssueListCommentsOptions{}

//   cc := []*github.IssueComment{}

//   for {
//     c, res, err := g.Issues.ListComments(ctx, org, repo, *i.Number, opts)
//     if err != nil {
//       return nil, err
//     }

//     cc = append(cc, c...)

//     if res.NextPage == 0 {
//       break
//     }

//     opts.Page = res.NextPage
//   }
// }

func githubToken() (string, error) {
	data, err := exec.Command("git", "config", "--get", "github.token").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("set github.token in your .gitconfig")
	}

	return strings.TrimSpace(string(data)), nil
}
