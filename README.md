# github-review-helper

[![Build Status](https://travis-ci.org/deiwin/github-review-helper.svg?branch=master)](https://travis-ci.org/deiwin/github-review-helper)
[![Coverage](http://gocover.io/_badge/github.com/deiwin/github-review-helper?0)](http://gocover.io/github.com/deiwin/github-review-helper)
[![GoDoc](https://godoc.org/github.com/deiwin/github-review-helper?status.svg)](https://godoc.org/github.com/deiwin/github-review-helper)

## What?
**github-review-helper** is a little bot that you can set up GitHub hooks for to improve your project's PR review flow.
It currently does 2 things:

1. It observes all PRs and detects if any `fixup!` or `squash!` commits are included in the PR. If there are, it uses
   the GitHub status API to mark these commits as pending.
2. It observes all PR comments (comments on the unified diff or the individual commits don't count) and if it sees a
   command of `!squash`, it tries to *autosquash* (equivalent of running `git rebase --interactive --autosquash`
   manually and instantly closing and saving the interactive rebase editor) all the commits in the PR. It reports
   success/failure through the GitHub status API.

## Quick start
### Create an access token for the bot
This step is nicely [covered in GitHub's own
documentation](https://help.github.com/articles/creating-an-access-token-for-command-line-use/). Create a token
following the guide and mark it down.

### Install and start the bot
The following commands expect you to have Go installed and your *GOPATH* to be properly set up. To compile and install
the bot, run the following commands:
```
go get github.com/deiwin/github-review-helper
cd $GOPATH/github.com/deiwin/github-review-helper
go install
```

The bot requires some environment variables to be set for it to funcion. Let's quickly go over each one to see what it
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
 - Use the **Let me set individual events** option and select the **Issue comment** and **Pull Request** events from the
   list that gets opened
 - Enable the webhook by leaving the **Active** checkbox checked

Click on **Add webhook** to finish the process.

*See the [GitHub
documentation](https://developer.github.com/webhooks/creating/) on creating webhooks for more info.*

### All set! Now try it out
To try it out you can make some changes to your code on a feature branch that you've opened a PR for. Then stage these
changes with `git add`. Now create a *fixup* commit for you current HEAD with `git commit --fixup=@` and push the
changes. You should see a *pending* status next to the *fixup* commit. (If you don't, check the **Recent Deliveries**
section in your webhook's settings to see what went wrong)

Now to try squashing the *fixup* commit, try leaving a comment on the PR with a message of only `!squash`. The bot
should squash the *fixup* commit and push the new changes. It should also update the last commit's status to *success*
saying that all *fixup* commits have been successfully squashed.
