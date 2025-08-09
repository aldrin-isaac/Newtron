# Lessons Learned for Newtron Project

## Session: newtron-20250804-01
- Ensure all code artifacts include the session ID and timestamp for tracking evolution.
- Stick strictly to existing features without adding new ones unless approved.
- Use validator/v10 for validation where applicable, but not forced in initial fixes.
- Maintain DRY by reusing existing helpers like build* functions.
- Separate concerns: Keep backend (device) logic free of CLI-specific code.