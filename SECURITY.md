# Security Policy

## Reporting a vulnerability

skillscope reads SKILL.md files from your filesystem and can copy /
move / delete them. The interesting attack surface is anything that
could trick the tool into:

- Writing to a scope marked `ReadOnly` (plugin scopes).
- Following a symlink out of a known scope root.
- Parsing malicious YAML in a way that corrupts other skills' state.

If you find a way to do any of the above — or any other security
issue you'd rather not file in public — please open a private
advisory at:

  https://github.com/heidihowilson/skillscope/security/advisories/new

Or email the maintainer directly at the address listed on the
GitHub profile. We aim to respond within a week.

Please don't file public issues for security bugs. Public bug reports
for non-security issues are very welcome.

## Supported versions

Only the latest minor release is supported. We don't backport fixes.

## Scope of "safe by default"

- Operations refuse to follow symlinks (using `os.Lstat`).
- Operations refuse to write to scopes marked `ReadOnly`.
- Delete demands the user type the skill name exactly. No `--force`.
- The scanner never executes any code from a SKILL.md — it only
  parses YAML frontmatter and reads the markdown body.
