```
sequenceDiagram
    participant sdk as lookout-sdk
    participant Analyzer
    participant Babelfish
    sdk->>Analyzer: NotifyReviewEvent
    Analyzer->>sdk: GetChanges/GetFiles
    sdk->>Babelfish: parseFile (if WantUAST)
    Babelfish-->>sdk: UAST
    sdk-->>Analyzer: Change/File
    Analyzer-->>sdk: Comments

```

Use https://mermaidjs.github.io/mermaid-live-editor/ to render.

