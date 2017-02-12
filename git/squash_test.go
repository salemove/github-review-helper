package git_test

import "testing"

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

	repo, cleanup := cloneTestRepo(t, testRepoDir)
	defer cleanup()

	err := repo.AutosquashAndPush("origin/master", "origin/"+featureBranchName, featureBranchName)
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
