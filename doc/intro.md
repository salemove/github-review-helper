# Improving GitHub reviews with fixup commits and a helpful bot

_github-review-helper_ is a tool that aims, as the name implies, to improve the
GitHub PR review workflow. This document explains what exactly do GitHub PRs
need help with and how _github-review-helper_ addresses those problems.

## The problem

Let's imagine Alice and Bob working together on a software project. Let's
further imagine that Alice has just managed to track down and fix a bug that's
been troubling them both for a while. Proud of her achievement, Alice commits
her changes, creates a PR, and asks Bob to give it a look, assigning him as the
reviewer.

Bob, ever so observant, notices that one of the tests that Alice has added
couldn't possibly pass. He leaves a comment pointing out the issue and
suggesting a solution. Few moments later their helpful CI also chimes in,
notifying of the test failure by leaving a little red cross on the PR.

Now, the problem lies in the question of how Alice should proceed from here?
Most teams/projects have adopted one of the following practices.

- Alice adds another commit to the PR, named "Fix tests". The PR will later be
  merged to master and that "Fix tests" commit will be reachable from master.
- Alice again adds a "Fix tests" commit to the PR, but when the PR is ready to
  be merged it is first squashed into a single commit and only then merged.
- Alice changes the commit that introduced the issue, amending the commit to
  include her test fixes.

All three approaches have their merits and problems. Let's go through all of
them one by one.

**Adding a new commit that will later be merged to master.**

We believe that in order to collaborate effectively, all commits that get
merged into master should follow a few basic [rules][0]. One of these rules
states that commits should be [complete][1]. Having changes that break a test
and changes that fix that test in two separate commits breaks that rule. When
this rule is broken, then cherry-picking or reverting the change becomes much
more difficult and tracking down bugs also becomes more difficult, because
tools like `git bisect` might not work as expected. In addition, usage of `git
blame` becomes a lot more tedious when this rule is broken often, because
related changes can not be expected to have come from the same commit.

**Adding a new commit that will be squashed before merging into master.**

This approach has gained popularity since GitHub added this option to their PR
merge button. And it does avoid the problem of completeness introduced by the
previous approach, but it has it's own problems. When a PR is very small, then
this approach actually works amazingly well. It breaks down, however, when a PR
has a bit more going on. Its main problem is that it breaks the rule which
states that commits should be [focused][2], they should only do _one thing_ (a
specialization of the more general Single Responsibility Principle). Following
this rule makes the use of `git blame` a joy and PR reviews productive. Both
benefits which will be lost when the rule is broken, as it would be with a
squash. Furthermore, a lot of valuable context and documentation (that
[hopefully][3] was included in the commit message) can be lost with the squash.

**Amending the original commit.**

This approach requires the contributors to a project to be a bit more familiar
with git than the previous approaches, but the commits that will eventually be
merged to master will be void of the problems caused by the previous two
approaches. With this approach, commits can be kept both complete and focused
on a single thing, which makes this approach the preferred solution for many
teams where the members are reasonably comfortable with git. But this approach
is also not without flaws. It sometimes makes it very hard to answer the
question that comes up often in PR reviews: "What exactly did you change?"
Coming back to our example with Alice and Bob, if Alice were to use this
approach, then after she pushes the new version of her commits, that is all
that Bob will see. He will not be able to compare the new version of the PR to
the old (pre-review) version, neither will he be able to see what exactly did
Alice change in response to his comments.

This problem of not being able to see a PR's evolution is not something that's
easily noticed, unless one has worked with other tools that do provide this
visibility.

## A solution

Inspired by how other tools, such as [Gerrit][4], for example, manage to avoid
the issues described above, _github-review-helper_ was built to solve this
problem without requiring its users to leave the GitHub platform. What
_github-review-helper_ essentially does is it combines the three approaches
described above. It does so in such a way that we get to keep the benefits of
every approach while leaving behind their downsides. It manages to do so at the
cost of a little added complexity.

Let's see how it works by revisiting our example, with Alice and Bob using
 _github-review-helper_ this time around.

Once Alice has locally fixed the issue with the tests, she will then put these
test fixes into a fixup commit. A fixup commit is like any other commit, except
it has a very specific title, which will refer to another commit. Alice will
create the fixup commit so that it will refer to the original commit that broke
the test. She will then push her changes and ask Bob to review the PR again.
Notice that at this point the workflow has been very similar to the first two
approaches described above, except that instead of the commit's title being
"Fix tests", it'll now be something like "fixup! Fix annoying bug".

Now when Bob looks at the PR again, he can see all the changes that Alice made
after the initial review by looking at the fixup commit's diff. He likes the
way Alice has chosen to fix the tests and because he is just as anxious as
Alice is to get this bug fixed, he quickly approves the PR.

With the PR approved and CI checks passing, it is almost ready to be merged.
But we don't want to simply merge it as-is, because that would also merge the
fixup commit into master which is problematic for all the reasons previously
discussed. In fact, if the repository is properly configured, Alice will not
_be able_ to merge the PR if it includes any fixup commits. So now, instead of
clicking the merge button on the PR, Alice will write a comment that says just
"!merge". She will then see how the "Fix annoying bug" and "fixup! Fix annoying
bug" commits are squashed into a single "Fix annoying bug" commit, while all
other commits remain intact. Then, once the CI finishes re-checking the PR
after the squash, Alice sees that the PR is automatically merged.

## Onward

If you feel like you or your team is affected by this problem, then briefly
read up about [fixup commits][5] if you need to and head over to the
[readme](../README.md) for instructions on setting up _github-review-helper_
for your own project.

And if you feel like you are affected by this problem, but don't like the
solution provided here, then don't despair - there's plenty of other, more
comprehensive solutions out there. Check out [Reviewable][6] and [Gerrit][4],
for example.

[0]: rules.md
[1]: rules.md#complete
[2]: rules.md#focused
[3]: rules.md#context-providing
[4]: https://www.gerritcodereview.com/
[5]: https://git-scm.com/docs/git-commit#git-commit---fixupltcommitgt
[6]: https://reviewable.io/
