# Issue tracker: Linear

Issues and specs for this repo live in Linear, project **Kana**. The workspace and team are named
in the local, uncommitted `CLAUDE.md` ("Agent skills" section); use those values wherever a tool
needs a team. Do not write workspace or team names into committed files, commit messages or PR
bodies. Use the Linear MCP tools (`mcp__plugin_linear_linear__*`). GitHub holds code and pull
requests only; do not create GitHub issues.

## Conventions

- **Create an issue**: `save_issue` with the team from `CLAUDE.md`, `project: "Kana"`, a title and
  a Markdown description. New issues start in **Backlog**.
- **Read an issue**: `get_issue` with its identifier (team key plus number); comments via
  `list_comments`.
- **List issues**: `list_issues` with `project: "Kana"` plus `label` / `state` filters.
- **Sub-issue**: `save_issue` with `parentId` set to the parent's identifier.
- **Comment**: `save_comment` on the issue.
- **Apply / remove labels**: `save_issue` with the full `labels` list.
- **Status**: Backlog → Todo → In Progress → Done; Canceled / Duplicate for closed-without-doing.
- **Close**: set the state to Done (or Canceled), with a comment explaining why.
- **No issue identifiers in the repository.** Identifiers contain the team key, so never put
  them in commit messages, PR titles or bodies, branch names or committed files. Link the other
  way round: attach the PR or commit URL to the Linear issue (comment or link), so the issue
  points at the code, not the code at the issue.
- Do not paste private word-list contents into issues.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if external PRs on GitHub should be triaged like
issues.)_

## When a skill says "publish to the issue tracker"

Create a Linear issue in the team from `CLAUDE.md`, project Kana.

## When a skill says "fetch the relevant ticket"

Read it as in **Read an issue** above.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue with **child** issues as tickets.

- **Map**: a Kana issue labelled `wayfinder:map` holding Notes / Decisions-so-far / Fog.
- **Child ticket**: a sub-issue of the map (`parentId`), labelled `wayfinder:<type>`
  (`research` / `prototype` / `grilling` / `task`); assigned to the driving dev once claimed.
- **Blocking**: Linear's native "blocked by" relations (`save_issue` with `blockedBy`). A ticket is
  unblocked when every blocker is Done or Canceled.
- **Frontier query**: open children of the map without open blockers and without assignee; first
  in map order wins.
- **Claim**: assign to `me`, the session's first write.
- **Resolve**: comment the answer, set Done, then append a context pointer (gist + link) to the
  map's Decisions-so-far.
