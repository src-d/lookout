```
sequenceDiagram
    participant GitHub
    participant lookout
    participant Analyzer
    participant Babelfish
    lookout->>GitHub: Polling
    GitHub-->>lookout: PR/Push events
    lookout->>Analyzer: NotifyReviewEvent
    Analyzer->>lookout: GetChanges/GetFiles
    lookout->>Babelfish: parseFile (if WantUAST)
    Babelfish-->>lookout: UAST
    lookout-->>Analyzer: Change/File
    Analyzer-->>lookout: Comments
    lookout->>GitHub: Post comments
```

Use https://mermaidjs.github.io/mermaid-live-editor/ to render.
