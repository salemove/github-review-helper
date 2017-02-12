package git_test

import (
	"bufio"
	"github.com/salemove/github-review-helper/git"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type file struct {
	Name     string
	Contents string
}

var (
	readme = file{
		Name:     "README.md",
		Contents: "This is a test file\n",
	}
	foo = file{
		Name:     "foo",
		Contents: "foo\n",
	}
	bar = file{
		Name:     "bar",
		Contents: "bar\n",
	}
)

func TestSquash(t *testing.T) {
	skipWithoutGit(t)

	testRepoGit, testRepoDir, cleanup := createTestRepo(t)
	defer cleanup()

	featureBranchName := "feature"
	testRepoGit("checkout", "-b", featureBranchName)

	createFile(t, testRepoDir, foo)
	testRepoGit("add", foo.Name)
	commitToFixMessage := "Add foo"
	testRepoGit("commit", "-m", commitToFixMessage)

	createFile(t, testRepoDir, bar)
	testRepoGit("add", bar.Name)
	testRepoGit("commit", "--fixup=@")

	// Checkout master because git by default doesn't allow pushing onto
	// branches that are currently checked out and we want to confirm that the
	// feature branch is properly updated after the squash.
	testRepoGit("checkout", "master")

	reposDir, cleanup := createTempDir(t)
	defer cleanup()

	gitRepos := git.NewRepos(reposDir)
	repo, err := gitRepos.GetUpdatedRepo(testRepoDir, "my", "test-repo")
	checkError(t, err)
	err = repo.AutosquashAndPush("origin/master", "origin/"+featureBranchName, featureBranchName)
	checkError(t, err)

	// Check that all files still exist in the feature branch and that the
	// fixup commit has been squashed to its parent
	testRepoGit("checkout", featureBranchName)

	checkFile(t, testRepoDir, readme)
	checkFile(t, testRepoDir, foo)
	checkFile(t, testRepoDir, bar)

	headCommitMessage := testRepoGit("show", "-s", "--format=%B", "@")
	if headCommitMessage != commitToFixMessage {
		t.Fatalf(
			"Expected HEAD commit to have message \"%s\", but got \"%s\"",
			commitToFixMessage,
			headCommitMessage,
		)
	}
}

func TestDeleteRemoteBranch(t *testing.T) {
	skipWithoutGit(t)

	testRepoGit, testRepoDir, cleanup := createTestRepo(t)
	defer cleanup()

	featureBranchName := "feature"
	testRepoGit("checkout", "-b", featureBranchName)

	createFile(t, testRepoDir, foo)
	testRepoGit("add", foo.Name)
	commitToFixMessage := "Add foo"
	testRepoGit("commit", "-m", commitToFixMessage)

	// Checkout master because git by default doesn't allow pushing onto or
	// deleting branches that are currently checked out.
	testRepoGit("checkout", "master")

	reposDir, cleanup := createTempDir(t)
	defer cleanup()

	gitRepos := git.NewRepos(reposDir)
	repo, err := gitRepos.GetUpdatedRepo(testRepoDir, "my", "test-repo")
	checkError(t, err)

	err = repo.DeleteRemoteBranch(featureBranchName)
	checkError(t, err)

	branches := getBranches(testRepoGit)
	hasOnlyMaster := len(branches) == 1 && branches[0] == "master"
	if !hasOnlyMaster {
		t.Fatalf(
			"Expected the repo to only have a master branch but it has: %s",
			strings.Join(branches, ", "),
		)
	}
}

func getBranches(git gitClient) []string {
	branchesString := git("for-each-ref", "--format=%(refname:short)", "refs/heads/")
	return strings.Fields(branchesString)
}

type gitClient func(...string) string

func createTestRepo(t *testing.T) (gitClient, string, func()) {
	testRepoDir, cleanup := createTempDir(t)
	testRepoGit := gitForPath(t, testRepoDir)

	testRepoGit("init")
	// Configure username and email that are required for creating commits
	testRepoGit("config", "user.name", "git-test")
	testRepoGit("config", "user.email", "<>")
	createFile(t, testRepoDir, readme)
	testRepoGit("add", readme.Name)
	testRepoGit("commit", "-m", "Init with foo")

	return testRepoGit, testRepoDir, cleanup
}

func checkFile(t *testing.T, dirPath string, f file) {
	path := filepath.Join(dirPath, f.Name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error(f.Name + " does not exist")
		t.Fatal(err)
	}
	contents, err := ioutil.ReadFile(path)
	checkError(t, err)
	if string(contents) != f.Contents {
		t.Fatal(f.Name+" has unexpected contents: ", string(contents))
	}
}

func createFile(t *testing.T, dirPath string, f file) {
	path := filepath.Join(dirPath, f.Name)
	err := ioutil.WriteFile(path, []byte(f.Contents), 0644)
	checkError(t, err)
}

func gitForPath(t *testing.T, repoPath string) gitClient {
	pathArgs := []string{"-C", repoPath}

	return func(args ...string) string {
		allArgs := append(pathArgs, args...)
		cmd := exec.Command("git", allArgs...)

		stdout, err := cmd.StdoutPipe()
		checkError(t, err)
		stderr, err := cmd.StderrPipe()
		checkError(t, err)

		err = cmd.Start()
		checkError(t, err)

		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			t.Logf("stderr: %s\n", stderrScanner.Text())
		}
		if err := stderrScanner.Err(); err != nil {
			t.Errorf("error reading stderr: %v\n", err)
		}

		out, err := ioutil.ReadAll(stdout)
		checkError(t, err)

		if err = cmd.Wait(); err != nil {
			t.Fatalf(
				"Running command \"git %s\" failed: %v\n", strings.Join(allArgs, " "),
				err,
			)
		}
		return strings.TrimSpace(string(out))
	}
}

func skipWithoutGit(t *testing.T) {
	_, err := exec.LookPath("git")
	hasGit := err == nil
	if !hasGit {
		t.Skip("Could not find git on PATH")
	}
}

func createTempDir(t *testing.T) (string, func()) {
	path, err := ioutil.TempDir("", "git-test")
	if err != nil {
		t.Fatal("Failed to create temporary directory for the test", err)
	}
	cleanup := func() {
		os.RemoveAll(path)
	}
	return path, cleanup
}

func checkError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
