# Tracing (Experimental)

The experimental tracing feature in `jjui` enhances the visualization of revision lanes in the log graph. When enabled, it highlights the lanes associated with the currently selected revision and dims revisions that are outside of those lanes, making it easier to follow the ancestry and relationships in complex histories.

## Enabling Tracing

To enable tracing, add the following section to your configuration file:

```toml
[ui.tracer]
enabled = true
```

## How It Works

- **Lane Highlighting:** When you select a revision, all lanes (branches) related to that revision are highlighted in the log graph.
- **Dimming:** Revisions that do not belong to the selected lanes are fainted, helping you focus on the relevant part of your repository's history.

## Limitations

- It's off by default
- Works only with the curved graph style