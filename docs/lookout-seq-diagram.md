```
sequenceDiagram
    participant GitHub
    participant Server
    participant Bblfsh
    participant DataService (inside Server process)
    participant Analyzer
    Server->>GitHub: Poll: new PRs/Pushes for registered Repos?
    loop Healthcheck
        Server->Server: keep waiting
    end
    Server->>Analyzer: NotifyReviewEvent
    Analyzer->>DataService: GetChanges
    Note left of DataService: req.WantUAST?
    DataService->>Bblfsh: parseFile
    Bblfsh -->> DataService: resp.UAST
    DataService-->>Analyzer: stream Change
    Analyzer-->>Server: Comments
    Server->>GitHub: Post all comments
```

Use https://mermaidjs.github.io/mermaid-live-editor/ to render.
