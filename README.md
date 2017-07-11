# github-review-helper

[![Build Status](https://travis-ci.org/salemove/github-review-helper.svg?branch=master)](https://travis-ci.org/salemove/github-review-helper)
[![Coverage](http://gocover.io/_badge/github.com/salemove/github-review-helper?0)](http://gocover.io/github.com/salemove/github-review-helper)
[![GoDoc](https://godoc.org/github.com/salemove/github-review-helper?status.svg)](https://godoc.org/github.com/salemove/github-review-helper)

## What?

**See [here](doc/intro.md)** for a high-level introduction.

**github-review-helper** is a little bot that you can set up GitHub hooks for to improve your project's PR review flow.
It currently does 4 things:

1. It observes all PRs and detects if any `fixup!` or `squash!` commits are
   included in the PR. If there are, it uses the GitHub status API to mark the
   PR as **pending** with `review/squash` context. If there are no *fixup* or
   *squash* commits, it marks the PR as **success**. This allows one to set the
   `review/squash` **success** status as required in the repo's GitHub settings
   to make sure no PR that includes *fixup* or *squash* commits gets
   accidentally merged.
2. It observes all PR comments (comments on the unified diff or the individual
   commits don't count) and if it sees a command of `!squash`, it tries to
   *autosquash* (equivalent of running `git rebase --interactive --autosquash`
   manually and instantly closing and saving the interactive rebase editor) all
   the commits in the PR. Success/failure will be reflected by the
   `review/squash` status.
3. Similarly to `!squash`, it also listens for `!check` commands. The `!check`
   command can be used to force the bot to (re-)check the current PR for
   `fixup!` and `squash!` commits. This can be useful when some webhooks didn't
   reach the bot properly or when you have reason to believe that the bot
   didn't correctly evaluate your PR automatically. Which can sometimes happen,
   because the bot is fast and can at times fetch data from the GitHub API
   before that data has been updated, causing the bot to make it's judgment
   based on outdated data.
4. It listens for `!merge` commands. `!merge` command will squash the PR
   (exactly like `!squash` would) if needed and will then merge the PR as soon
   as all required status checks are marked as "success". If any of the status
   checks fail after that, the bot will cancel the merging process (indicated
   by a 'merging' label on the PR) and will notify the PR's author.

## Quick start
### Create an access token for the bot
This step is nicely [covered in GitHub's own
documentation](https://help.github.com/articles/creating-an-access-token-for-command-line-use/). Create a token
following the guide and mark it down.

### Run the bot from a docker image
```
docker run \
  -e GITHUB_ACCESS_TOKEN="the-access-token-you-created-above" \
  -e GITHUB_SECRET="a-secret" \
  -v ~/.ssh:/etc/secret-volume \
  -p 4567:80 \
  salemove/github-review-helper
```

Note that the snippet above mounts your local `~/.ssh` folder as a volume into
the Docker container. This is required for the bot to be able to connect to
your repositories using git. It will use the `known_hosts` file from that
mounted folder for making sure that your connection to github.com is secure and
the `id_rsa` file for the SSH identity.

### Install and start the bot (if you don't want to use docker)
The following commands expect you to have Go installed and your *GOPATH* to be properly set up. To compile and install
the bot, run the following commands:
```
go get github.com/salemove/github-review-helper
cd $GOPATH/github.com/salemove/github-review-helper
go install
```

The bot requires some environment variables to be set for it to function. Let's quickly go over each one to see what it
is and why it's needed.

 - `PORT`: The port the bot will be listening for connections on
 - `GITHUB_ACCESS_TOKEN`: The token we created in a previous step. This required to authenticate your account with
   GitHub.
 - `GITHUB_SECRET`: Another secret token that we will later use to configure GitHub webhooks for the bot. This will help
   us make sure that all the requests are coming only from GitHub. [GitHub
   suggests](https://developer.github.com/webhooks/securing/#setting-your-secret-token) running `ruby -rsecurerandom -e
   'puts SecureRandom.hex(20)'` to generate this token.

Now let's start the bot (you can replace `$GOPATH/bin/github-review-helper` with just `github-review-helper` if you have
go executables on your path):
```
 PORT=4567 GITHUB_ACCESS_TOKEN="the-acces-token-you-created-above" GITHUB_SECRET="a-secret" $GOPATH/bin/github-review-helper
```
PS: *The bot also needs git to be available on path and it expects the user the command is run under to have ssh access
to the repositories it is used for.*

### Set up a tunnel
Leave the bot running and let's now set up a tunnel to localhost.  This example depends on [ngrok](https://ngrok.com/)
being installed and available on the system (as do the official [GitHub webhook
docs](https://developer.github.com/webhooks/configuring/#using-ngrok)) to make the bot publicly accessible by GitHub. So
go ahead and install it if you haven't already. When you're done, you can create a tunnel by running:
```
ngrok 4567
```
You should see something like the following in the output:
```
Forwarding    http://7e9ea9dc.ngrok.com -> 127.0.0.1:4567
```
Note down the `http://*.ngrok.com` URL.

### Configure the webhook
To set up a repository webhook on GitHub, head over to the **Settings** page of your repository, and click on **Webhooks &
services**. After that, click on **Add webhook**. Then:

 - Enter the ngrok address you marked down earlier as the **Payload URL**
 - Leave **Content type** to be `application/json`
 - Enter the secret token you created before and used to start the bot as the **Secret**
 - Use the **Let me set individual events** option and select the **Issue comment**, **Pull Request**, and **Status**
   events from the list that gets opened
 - Enable the webhook by leaving the **Active** checkbox checked

Click on **Add webhook** to finish the process.

*See the [GitHub
documentation](https://developer.github.com/webhooks/creating/) on creating webhooks for more info.*

### [Optional] Make `review/squash` **success** required

If you wish to have the merge button disabled for PRs with *fixup* and *squash* commits in them, then make this status
required. This can be done by going to the repo's **Settings** on GitHub, then going to the **Branches** section and
selecting the branch you wish to protect from the **Protected branches** dropdown (or clicking on **Edit** next to the
branch, if it's already protected). Now checking the **Require status checks to pass before merging** checkbox and then
the `review/squash` context from the displayed list. NB: The bot must have had a change to check at least one PR for the
context to appear in the list.

*See the [GitHub
documentation](https://help.github.com/articles/enabling-required-status-checks/) for a visual guide.*

### All set! Now try it out
To try it out you can make some changes to your code on a feature branch that you've opened a PR for. Then stage these
changes with `git add`. Now create a *fixup* commit for you current HEAD with `git commit --fixup=@` and push the
changes. You should see a *pending* status next to the *fixup* commit. (If you don't, check the **Recent Deliveries**
section in your webhook's settings to see what went wrong)

Now to try squashing the *fixup* commit, try leaving a comment on the PR with a message of only `!squash`. The bot
should squash the *fixup* commit and push the new changes. It should also update the last commit's status to *success*
saying that all *fixup* commits have been successfully squashed.
