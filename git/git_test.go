package git_test

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/salemove/github-review-helper/git"
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

type gitClient func(...string) string

func cloneTestRepo(t *testing.T, testRepoDir string) (git.Repo, func()) {
	reposDir, cleanup := createTempDir(t)

	gitRepos := git.NewRepos(reposDir)
	repo, err := gitRepos.GetUpdatedRepo(testRepoDir, "my", "test-repo")
	checkError(t, err)

	return repo, cleanup
}

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
