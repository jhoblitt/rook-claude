# Fanning out subagents

Width, dispatch, and refill order for any fan-out — a review panel, a triage
batch, a sweep's scan or implement wave.

Width is ~6–8 agents in flight per SESSION — not per sweep, corpus, or
phase — with the rest queued and launched as slots free. Independent agents
are dispatched in ONE message; one per response is serial, whatever the
width.

Count AGENTS, not tasks: a nested fan-out spends a whole panel from that one
budget, so a stage spawning panels runs one panel at a time rather than one
per item. When a slot frees, give it to a downstream stage of something
already in flight before the next queued item; otherwise a "spawn verifiers
as each reviewer completes" pipeline silently degrades into a barrier once
the queue is longer than the budget. A confirmation gate bounds cost, not
width — state both.

This is the normative statement of the width and of one-message dispatch;
the skills and agent definitions carry only their own nested-fan-out delta.
