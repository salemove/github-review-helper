# Four rules for effective collaboration

Whether you're working in a large team or solo, effective collaboration is
predicated upon keeping a clean version control history. (Yes, dealing with
your own past follies also counts as collaboration.) Here we'll consider a
clean history to be one in which commits are _Conventional_,
_Context-Providing_, _Complete_, and _Focused_. Each of these four rules (or
guidelines) is described in detail below.

It is important to note that these rules only apply for commits that are shared
and/or eventually merged into a stable branch (usually master). But this is not
the only situation in which commits can be used - quick and dirty commits are
very helpful when considered as drafts. These rules are not meant to dissuade
one from committing early and committing often. During the early divergent
phase of problem solving, when many different approaches are considered,
following these rules would only inhibit the creative process.

## The rules

### Conventional

**Every commit message should follow the conventions of its project.**

There are a [few very basic conventions][0] that almost every project follows,
but the specifics vary greatly from project to project. What's important, is to
follow whatever conventions have been agreed upon. Are all commit titles (the
first line of the commit message) in the project in imperative mood? Every new
title should also be in the imperative mood. Are commit titles prefixed by the
general nature of the change (e.g. documentation/bug/feature/refactoring)?
Every new commit should also have the prefix.

### Context-Providing

**Every commit message should explain the commit's reason for being.**

The message should provide (within reason) all possible context for the commit.
Here are some of the questions that the message should answer:

- Why is this change necessary?
- What assumptions is it based on?
- Were any alternative solutions considered?
- Why was this solution chosen from among the alternatives?
- Are there any non-obvious implications that this change has?

References to other relevant information may be included. Some examples:

- A ref to another commit. E.g. a bug fix may refer to the commit that
  introduced the bug for providing additional context. Usually the abbreviated
  SHA (e.g. `c71598f`) of the commit is sufficient, but more comprehensive
  styles, such as the [one used for git itself][1], can be considered.
- Link to library/language or any other documentation. Often it also makes
  sense to actually quote the most relevant bits within the message as well, to
  avoid losing the context when the link stops working.
- Link or a reference to the relevant ticket/issue for which this commit was
  created.

### Complete

**Every commit should be complete.**

This means that every commit should leave the system in a state where: 1) the
code compiles, 2) its tests pass (note that requiring end-to-end tests to pass
isn't always reasonable, but lower level tests should always pass), and 3) the
linter does not complain. This means, for example, that a bug fix and the
related test changes should be in the same commit.

### Focused

**Every commit should only do a _single thing_.**

What this means precisely is often arguable, but what it means in general is
usually well understood. For example, refactoring should be separate from an
addition of a feature. Formatting changes should be separate from bug fixes. A
commit should only fix a single bug at a time, not two or three.

Note that keeping commits _Complete_ forces one towards bigger (less in terms
of lines and more in terms of logical changes) commits, whereas keeping them
_Focused_ pushes towards smaller commits. This is so because essentially
they're different perspectives or gages for reaching the same ultimate goal -
having a commit that does just enough and not a bit more.

## Benefits

The benefits are best illustrated by looking at different phases or processes
of development and seeing how following the specific rules affects that
phase/process.

### Code review

**_Conventional_** commits are easy to grasp at a glance. Without having to
know the idiosyncrasies of the author, it is usually clear, in broad terms,
what the commit does. Just by looking at its title. Also, _Conventional_ commit
messages should never break the tooling. E.g. if a commit message's body is not
wrapped at the recommended 72 characters, then the message becomes hard to read
both on GitHub and in the terminal.

**_Context-Providing_** commits are just a joy to review. The author has
already answered the most likely questions the reviewer might think of while
looking at the diff. This reduces a lot of time consuming back-and-forth. In
addition, if the commit does not provide enough context, then often reviewers
may feel overwhelmed by the change and only check it for syntax errors and
typos, without delving into the bigger questions. This way the team might miss
out on important insights.

**_Complete_** commits minimize the amount of information the reviewer has to
keep in their head. For example, if a function's interface is changed in one
commit and its callers are changed in another, then the reviewer must either
remember how the function was exactly changed, when looking at the callers, or
they must jump between the two commits. Both options complicate the process and
lower the chance of catching issues. In addition, when commits are _Complete_,
the reviewer can compile the code and run the tests to verify the correctness
of _every_ commit.

**_Focused_** commits often make it possible to review at all, with any sort of
confidence. If formatting changes are clumped together with a feature addition,
then important parts of the new feature might be buried in the diff of the dumb
formatting change and might be missed by the reviewer. If a few bug fixes, a
new feature, and some refactoring is all in a single commit, then comprehensive
reviewing is usually impossible and any reviews that the commit does get are
necessarily shallow.

### Debugging or understanding existing code

This section covers the general concept of understanding already existing (or
old) code, either for debugging or some other purpose. Most commonly deeper
understanding is required for debugging purposes, but there are other reasons
as well. For example, figuring out why a feature was developed in a certain
way, when implementing something similar in another part of the system. In the
following text we'll use the word "debugging", but the benefits equally apply
to the other purposes as well.

**Debugging is very similar to reviewing, so many of the same benefits apply.**
However, because debugging usually happens a while after the change was made,
the author may no longer be available or they might not remember the specifics.
For this reason, it is of paramount importance that the commits be
**_Context-Providing_**. There just isn't anyone to ask anymore.

Debugging situations also illustrate why it's often (but not always) more
beneficial to put the context into the commit and not into comments in the
code. Namely, the code comments might have been changed or deleted by this
time. And although the original code comments are still available within the
diff of the commit, if it's not a practice to check the commits for the context
(which is usually the case, if commits don't provide enough context) then the
diff will likely not be looked at.

**_Complete_** commits make it possible to use `git bisect`, which may turn out
to be very helpful for finding the source of certain issues.

## Inspiration and additional reading

- [Guidelines for submitting patches to the git project][2]
- [Popular commit message conventions][3]
- [More thoughts on good commit messages][4]


[0]: https://git-scm.com/docs/git-commit#_discussion
[1]: https://github.com/git/git/blob/8b2efe2a0fd93b8721879f796d848a9ce785647f/Documentation/SubmittingPatches#L129-L139
[2]: https://github.com/git/git/blob/master/Documentation/SubmittingPatches
[3]: https://chris.beams.io/posts/git-commit/
[4]: http://who-t.blogspot.com.ee/2009/12/on-commit-messages.html
