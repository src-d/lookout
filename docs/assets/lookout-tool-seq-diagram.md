```
sequenceDiagram
    participant tool as lookout-tool
    participant Analyzer
    participant Babelfish
    tool->>Analyzer: NotifyReviewEvent
    Analyzer->>tool: GetChanges/GetFiles
    tool->>Babelfish: parseFile (if WantUAST)
    Babelfish-->>tool: UAST
    tool-->>Analyzer: Change/File
    Analyzer-->>tool: Comments

```

Use https://mermaidjs.github.io/mermaid-live-editor/ to render.

