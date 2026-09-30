# Claude Code — mcp-go-context

At the start of substantive work in this repository, call `resume-context` with the working path.

Pass that same path to `save-decision` and `remember-conversation`. Files and Git remain the source of truth; use the handoff for the objective, pending work and next step.

After a milestone, before switching to another client, or before ending work, call `save-handoff` with the revision returned by the last resume. If the result says `CONFLICT`, preserve both versions and reconcile them. Do not promote a handoff; promote only durable decisions.
