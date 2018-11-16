```
sequenceDiagram
    participant GitHub
    participant Server
    participant DataService
    participant Bblfsh
    participant Analyzer
    loop registered Repos
        Server ->> GitHub: polling PRs/Push events
        loop registered Analyzers
            Server ->> +Analyzer: (gRPC) NotifyReviewEvent / NotifyPushEvent
            Analyzer ->> +DataService: (gRPC) GetChanges / GetFiles
            opt req.WantUAST?
                DataService ->> +Bblfsh: (gRPC) Parse
                Bblfsh -->> -DataService: uast.Node
            end
            DataService -->> -Analyzer: stream of Change / File
            Analyzer -->> -Server: Comments
        end
    Server ->> GitHub: Post all comments
    end
```

Use https://mermaidjs.github.io/mermaid-live-editor/ to render.
