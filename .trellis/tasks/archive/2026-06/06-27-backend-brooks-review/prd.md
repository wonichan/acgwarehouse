# Backend code quality review using Brooks lint suite

## Goal

Perform comprehensive code quality review of the backend codebase using all four Brooks lint capabilities (review, audit, debt, health) to identify code decay, architecture issues, tech debt, and overall code health.

## Requirements

- Run brooks-review on recent git commits to surface code decay risks and maintainability issues
- Run brooks-audit to map module dependencies, check layering integrity, and flag structural decay
- Run brooks-debt to identify, classify, and prioritize maintainability problems
- Run brooks-health to get overall codebase health dashboard across all four quality dimensions
- Consolidate findings into actionable recommendations
- Preserve review artifacts in task directory for future reference

## Acceptance Criteria

- [ ] brooks-review completed with findings documented
- [ ] brooks-audit completed with architecture analysis documented
- [ ] brooks-debt completed with tech debt assessment documented
- [ ] brooks-health completed with health score documented
- [ ] Consolidated summary of critical issues and recommendations
- [ ] All findings preserved in task directory

## Notes

- Git history is available and should be used for review context
- Focus on backend code (no frontend review needed)
- Each Brooks skill should run independently to get comprehensive coverage