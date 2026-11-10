# Repository history hygiene

Before a sensitive-history cleanup, coordinate with repository maintainers, stop conflicting pushes, and create a private recovery backup. Keep recovery archives outside the active Git object database and out of deployment uploads.

Apply the smallest appropriate history filter. Verify all affected branches and tags, preserve unrelated source files and regression tests, and retain uncommitted work. Use exact-tip leases when updating remote branches so concurrent work is not overwritten.

A branch rewrite does not remove other clones, forks, hosting caches, or pull-request references. Follow the hosting provider's cleanup process and coordinate with collaborators before old history is pushed again. Replace affected clones or carefully rebase their work; do not merge removed history back into the repository.

Keep incident-specific findings, affected identifiers, cleanup status, support correspondence, and recovery locations in private records. Publish only reviewed process guidance and completed remediation summaries. Follow the [artifact handling policy](README.md).
