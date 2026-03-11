# general purpose
a gh extension for simplifying agents interaction with prs on github.
it helps agents become real contributors: read issues, write comments, read review results, respond with reactions and more.

# use cases

## remote control without cli/ui, only through github

story:
- an already running claude code session (a local or cloud one) finishes its work, commits and pushes to github and waits for an input
- a reviewer looks through pr, leaves comments, reactions and suggestions. the review is published with a slash command or smth else that triggers the session again
- the claude code instance awakes by the input, like "github: new review comments had arrived!" 
- the instance uses gh contribute to update its context about the pr
- continues working

result: interaction performed as with a real contributor, without cli

## leaving status through reactions

story:
- a claude session is invoked by notification about github new comments
- the comments are turned into a to-do list for further processing
- as a comment is being processed, the agent marks it with an eye-balls reaction on github
- as the comment is finished - with a rocket reaction
- as the review is finished with all its comments - the review itself is marked with a rocket

result: everyone on github knows the status without checking claude code session

# implementation

- use go language
- use github app installation for auth
- with gh extension, provide only the necessary commands for retrieving info about prs:
  - list comments
  - list resolved, non-resolved
  - etc.
- also provide commands for posting comments and reactions
- if a pr id, branch, project, etc. is not specified - use git to determine all the details

# test

gh is installed, .env has GITHUB_TOKEN with access to this repo. use it for testing.

---

general rule for output: minimalistic md style

# pr output:

- only pr description with meta, without comments
- header with title and pr id
- meta info 
- description is separated with ===

follow example:

```
# test-pr: test gh extension #1
open, by @ivanov-gv, 1 commit `test-pr` -> `main`, no merge conflict
https://github.com/ivanov-gv/gh-contribute/pull/1

Reviewers:  
Assignees: @ivanov-gv  
Labels:  
Projects:  
Milestone:  
Issues:  

===

No description provided.

===
```


# comments output:

- list of issues and reviews with ids for further requests
- meta: author, date, description, reactions
- markdown
- hide issues and reviews that are hidden
- don't expand reviews' comments. create a separate review command for viewing 


follow example:

```
# issue #4038597073 by you (@ivanov-gv-ai-helper[bot]) 
_2026-03-11 11:33:27_  

test comment from gh-contribute 🚀

(1 🚀)  
by you: (1 🚀)  

# issue #4038819817 by @ivanov-gv
_2026-03-11 12:15:54_

> test comment from gh-contribute 🚀
test reply

(1 😕)
by you:

# review #3929204495 by @ivanov-gv
_2026-03-11 12:17:34_

submit review

comments: 3
(1 👀)
by you: 

# review #3929353771 by @ivanov-gv | hidden: Resolved
```
