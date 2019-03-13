package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	flagPrBranch string
	flagRepo     string
	flagOwner    string
)

func init() {
	flag.StringVar(&flagPrBranch, "pr-branch", "", "pull request branch")
	flag.StringVar(&flagRepo, "repo", "pull-request-bot-test", "")
	flag.StringVar(&flagOwner, "owner", "electricface", "")
}

func fileToTreeEntry(filename string) (github.TreeEntry, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return github.TreeEntry{}, err
	}

	return github.TreeEntry{
		Path:    github.String(filepath.Base(filename)),
		Mode:    github.String("100644"),
		Type:    github.String("blob"),
		Content: github.String(string(content)),
	}, nil
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.Ltime)

	token, err := getToken()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		}))

	client := github.NewClient(tc)

	owner := flagOwner
	repo := flagRepo
	branch := "master"

	var entries []github.TreeEntry
	for _, arg := range flag.Args() {
		log.Println("file:", arg)
		treeEntry, err := fileToTreeEntry(arg)
		if err != nil {
			log.Fatal(err)
		}
		entries = append(entries, treeEntry)
	}

	ref, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		log.Fatal(err)
	}
	baseTree := ref.Object.GetSHA()
	// create tree
	tree, _, err := client.Git.CreateTree(ctx, owner, repo, baseTree, entries)
	if err != nil {
		log.Fatal(err)
	}

	commit := &github.Commit{
		Message: github.String("this is commit message"),
		Tree:    tree,
		Parents: []github.Commit{
			{SHA: github.String(baseTree)},
		},
	}
	// create commit
	comm, _, err := client.Git.CreateCommit(ctx, owner, repo, commit)
	if err != nil {
		log.Fatal(err)
	}
	prBranch := flagPrBranch
	if flagPrBranch == "" {
		log.Fatal("empty pr branch")
	}

	// create ref
	ref, _, err = client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref: github.String("heads/" + prBranch),
		Object: &github.GitObject{
			Type: github.String("commit"),
			SHA:  comm.SHA,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("ref url:", ref.GetURL())

	// create pull request
	head := owner + ":" + prBranch
	log.Println("pr head:", head)
	pr, _, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String("this is pull request title"),
		Head:  github.String(head),
		Base:  github.String(branch),
		Body:  github.String("this is pull request body"),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("pull requst:", pr.GetHTMLURL())
}

func getToken() (string, error) {
	tokenFile := filepath.Join(os.Getenv("HOME"), ".prbot-token")
	tokenData, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	tokenData = bytes.TrimSpace(tokenData)
	return string(tokenData), nil
}
