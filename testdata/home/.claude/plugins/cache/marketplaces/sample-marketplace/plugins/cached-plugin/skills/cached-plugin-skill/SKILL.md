---
name: cached-plugin-skill
description: Skill installed in the cache subdirectory (~/.claude/plugins/cache/...)
---

# Cached Plugin Skill

Lives under `~/.claude/plugins/cache/marketplaces/<mkt>/plugins/<plugin>/skills/<skill>/SKILL.md`.
Per Claude Code docs, "plugins are copied to a cache (`~/.claude/plugins/cache`)".
Empirically the non-cache path is what's used today, but scanning both is cheap insurance.
