package git_test

import (
	"strings"
	"testing"
)

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

	// Checkout master because git by default doesn't allow deleting branches
	// that are currently checked out.
	testRepoGit("checkout", "master")

	repo, cleanup := cloneTestRepo(t, testRepoDir)
	defer cleanup()

	err := repo.DeleteRemoteBranch(featureBranchName)
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

func TestDeleteRemoteBranch_noBranch(t *testing.T) {
	skipWithoutGit(t)

	_, testRepoDir, cleanup := createTestRepo(t)
	defer cleanup()

	repo, cleanup := cloneTestRepo(t, testRepoDir)
	defer cleanup()

	nonExistentBranchName := "feature"
	err := repo.DeleteRemoteBranch(nonExistentBranchName)
	if err == nil {
		t.Fatal("Expected deletion of a non-existent branch to fail")
	}
}

func getBranches(git gitClient) []string {
	branchesString := git("for-each-ref", "--format=%(refname:short)", "refs/heads/")
	return strings.Fields(branchesString)
}
